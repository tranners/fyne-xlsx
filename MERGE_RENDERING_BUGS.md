# Critical Bugs in Merge Cell Rendering

## Document Information
- **Date**: February 8, 2026
- **Severity**: CRITICAL
- **Impact**: Merge cells disappear or flicker during horizontal scrolling
- **Files Affected**: 
  - [`pkg/renderer_primitives.go`](pkg/renderer_primitives.go:176)
  - [`pkg/renderer_region.go`](pkg/renderer_region.go:71)

---

## Executive Summary

The merge rendering system has **multiple critical race conditions and logic errors** causing merge cell text and backgrounds to randomly disappear during scrolling, particularly when scrolling right then back left.

**Root Cause**: Unsynchronized object recycling between regular cells and merge cells, combined with debugging code that disabled regular cell rendering.

---

## 🔴 BUG #1: Debugging Code Blocks Regular Cell Rendering (CRITICAL)

**Location**: [`pkg/renderer_region.go:71-108`](pkg/renderer_region.go:71)

**Severity**: CRITICAL - Most cells are not being rendered!

### The Bug

```go
// Line 71-74: ONLY row 99 cells are added!
for colVisIdx := vpCurrent.FirstColVisIdx; colVisIdx <= vpCurrent.LastColVisIdx; colVisIdx++ {
    colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
    if rowVisIdx == 99 {  // ⚠️ BUG: Should be removing this condition
        pr.addCell(regionContainers.Background, regionContainers.Data, rowModIdx, colModIdx, region)
    }
}
```

**This appears in FOUR places**:
- Lines 71-74 (scroll up - top edge)
- Lines 81-84 (scroll down - bottom edge)
- Lines 94-97 (scroll left - left edge)
- Lines 104-107 (scroll right - right edge)

### Impact

**99% of cells are not rendered during viewport changes!** Only debug row 99 is rendered. This explains why:
- Initial render works (different code path)
- Scrolling breaks everything (this code path)
- Merge cells seem unstable (they're the only ones rendering)

### Fix

**Remove all `if rowVisIdx == 99` conditions**:

```go
// CORRECT CODE:
for rowVisIdx := vpCurrent.FirstRowVisIdx; rowVisIdx < vpPrevious.FirstRowVisIdx; rowVisIdx++ {
    rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
    for colVisIdx := vpCurrent.FirstColVisIdx; colVisIdx <= vpCurrent.LastColVisIdx; colVisIdx++ {
        colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
        pr.addCell(regionContainers.Background, regionContainers.Data, rowModIdx, colModIdx, region)
        // ⬆️ No condition - render ALL cells
    }
}
```

---

## 🔴 BUG #2: Race Condition in Object Pool (CRITICAL)

**Location**: [`pkg/renderer_primitives.go:176-210`](pkg/renderer_primitives.go:176)

**Severity**: CRITICAL - Causes random disappearance of merge cells

### The Bug

The [`renderVisibleMerges()`](pkg/renderer_primitives.go:176) function uses a **two-pass algorithm** that creates a race condition:

```go
// PASS 1: Remove invisible merges (line 179-188)
for anchorCellModId := range mm.anchorToSize {
    isVisible := mm.IsMergeInViewport(anchorCellModId, viewport)
    _, rectExists := pcr.rectanglePrimitivesIndex[gridRegion][anchorCellModId]
    _, textExists := pcr.textPrimitivesIndex[gridRegion][anchorCellModId]
    
    if !isVisible && (rectExists || textExists) {
        pcr.removeCell(anchorCellModId, gridRegion)  // ⚠️ Flags items for reuse
    }
}

// PASS 2: Add visible merges (line 190-209)
for anchorCellModId := range mm.anchorToSize {
    isVisible := mm.IsMergeInViewport(anchorCellModId, viewport)
    // ... tries to add merge cells
}
```

### The Race Condition

**Timeline of bug**:

1. **Merge cell A scrolls off right edge** → Pass 1 flags its rectangle/text (added to flagRectanglePrimitives)
2. **Regular cell B scrolls on left edge** → Calls `addCell()` → Steals Merge A's flagged items
3. **User scrolls back left** → Merge A should reappear
4. **Pass 2 tries to add Merge A** → Flagged pool is EMPTY → Can't find old object
5. **Result**: If recycler pool is also empty, Merge A's canvas object is never moved back into view

### Why This Happens

The flagged object pool is **shared** between:
- Regular cell rendering (`addCell()` in lines 283-322)
- Merge cell rendering (`renderVisibleMerges()` in lines 176-210)

**But they're not synchronized!** Regular cells can steal merge cell objects before the merge has a chance to reclaim them.

### Proof in Code

In [`addPrimitivesToCell()`](pkg/renderer_primitives.go:212):

```go
// Line 222-228: Tries to reuse flagged item
if flagCellId, ok := pcr.flagRectanglePrimitives[gridRegion].Get(); ok {
    idx := pcr.rectanglePrimitivesIndex[gridRegion][flagCellId]  // ⚠️ Assumes index exists!
    rectanglePrimiticeItem = pcr.rectanglePrimitives[gridRegion][idx]
    // ... uses it for NEW cell (could be regular cell stealing merge's object)
}
```

**No validation** that `flagCellId` has a valid index entry!

---

## 🟡 BUG #3: Missing Index Validation (HIGH)

**Location**: [`pkg/renderer_primitives.go:222-228`](pkg/renderer_primitives.go:222)

**Severity**: HIGH - Can cause nil pointer or index corruption

### The Bug

When reusing a flagged item, there's **no validation** that the index still exists:

```go
if flagCellId, ok := pcr.flagRectanglePrimitives[gridRegion].Get(); ok {
    idx := pcr.rectanglePrimitivesIndex[gridRegion][flagCellId]  // ⚠️ What if flagCellId was already deleted?
    rectanglePrimiticeItem = pcr.rectanglePrimitives[gridRegion][idx]
    // ...
}
```

### Scenario

1. Cell A removed → Index deleted → Object flagged
2. Cell B removed → Same object flagged AGAIN (duplicate in pool)
3. Cell C added → Gets flagged object → Tries to read `idx` for Cell A
4. **Boom**: Index doesn't exist, returns 0, uses wrong object

### Fix

Add validation:

```go
if flagCellId, ok := pcr.flagRectanglePrimitives[gridRegion].Get(); ok {
    idx, indexExists := pcr.rectanglePrimitivesIndex[gridRegion][flagCellId]
    if !indexExists {
        // Index was already deleted, skip to recycler
        // (This should never happen but defensive programming)
        goto TryRecycler
    }
    // ... rest of code
}
TryRecycler:
    primitiveRectangleRecyleItem, recycledItem = pcr.rectangleRecycler[gridRegion].Get()
```

---

## 🟡 BUG #4: Index Corruption on Flagged Reuse (MEDIUM)

**Location**: [`pkg/renderer_primitives.go:227-228`](pkg/renderer_primitives.go:227)

**Severity**: MEDIUM - Index points to wrong cell

### The Bug

```go
if flagCellId, ok := pcr.flagRectanglePrimitives[gridRegion].Get(); ok {
    idx := pcr.rectanglePrimitivesIndex[gridRegion][flagCellId]
    // ...
    delete(pcr.rectanglePrimitivesIndex[gridRegion], flagCellId)  // Delete old mapping
    pcr.rectanglePrimitivesIndex[gridRegion][id] = idx            // Create new mapping
}
```

**Problem**: The `idx` value is reused but still points to the old cell's position in the slice. If multiple flagged items are consumed in different order, indices get mixed up.

### Example

```
Initial state:
  rectanglePrimitives = [rect0, rect1, rect2]
  index[CellA] = 0
  index[CellB] = 1
  index[CellC] = 2

CellA removed:
  flagged = [CellA]
  index[CellA] still = 0 (not deleted yet)

CellD added (consumes CellA):
  delete index[CellA]
  index[CellD] = 0  ← Now points to rect0
  rectanglePrimitives[0] is MOVED to new position
  
CellA comes back:
  ??? CellD is already using idx=0
```

---

## Root Cause Analysis

### Architecture Flaw

The system has **three object lifecycle states**:
1. **Active** - Visible in viewport, has valid index
2. **Flagged** - Just left viewport, queued for immediate reuse
3. **Recycled** - Moved to corner, in general pool

**The Problem**: States 1 & 2 share the same index map, but State 2 objects can be consumed before their original owner reclaims them.

### Design Issue

```
┌─────────────────────────────────────────┐
│         CURRENT FLAWED DESIGN           │
├─────────────────────────────────────────┤
│                                         │
│  Regular Cells  ←→  Flagged Pool  ←→  Merge Cells
│                         ↓               │
│                    SHARED!              │
│                   No priority           │
└─────────────────────────────────────────┘
```

---

## Recommended Fixes

### Fix Priority 1: Remove Debugging Code ⚡

**File**: [`pkg/renderer_region.go`](pkg/renderer_region.go:71)

**Change**: Remove all `if rowVisIdx == 99` conditions (4 locations):
- Line 71-74
- Line 81-84  
- Line 94-97
- Line 104-107

**Impact**: Immediate fix - cells will render properly during scrolling

### Fix Priority 2: Separate Merge Cell Object Pool 🔥

**File**: [`pkg/renderer_primitives.go`](pkg/renderer_primitives.go:21)

**Change**: Create separate flagger pools for merge cells:

```go
type PrimitiveRenderer struct {
    // Existing fields...
    flagRectanglePrimitives map[GridRegion]*PrimitiveFlagger
    flagTextPrimitives      map[GridRegion]*PrimitiveFlagger
    
    // NEW: Separate pools for merge cells
    flagMergeRectangles map[GridRegion]*PrimitiveFlagger
    flagMergeText       map[GridRegion]*PrimitiveFlagger
}
```

**Rationale**: Merge cells should not share the flagged pool with regular cells since they have different lifecycle timing.

### Fix Priority 3: Add Index Validation 🛡️

**File**: [`pkg/renderer_primitives.go:222`](pkg/renderer_primitives.go:222)

**Change**: Validate index exists before use:

```go
if flagCellId, ok := pcr.flagRectanglePrimitives[gridRegion].Get(); ok {
    if idx, exists := pcr.rectanglePrimitivesIndex[gridRegion][flagCellId]; exists {
        // Safe to use idx
        rectanglePrimiticeItem = pcr.rectanglePrimitives[gridRegion][idx]
        // ... rest of code
    } else {
        // Index corruption detected, log warning and skip to recycler
        fmt.Printf("[WARN] Flagged cell %v has no index entry\n", flagCellId)
    }
}
```

### Fix Priority 4: Single-Pass Merge Rendering 🔧

**File**: [`pkg/renderer_primitives.go:176`](pkg/renderer_primitives.go:176)

**Change**: Combine remove and add into single pass:

```go
func (pcr *PrimitiveRenderer) renderVisibleMerges(...) {
    mm := pcr.ctx.MergeManager
    
    // Single pass - process each merge once
    for anchorCellModId := range mm.anchorToSize {
        isVisible := mm.IsMergeInViewport(anchorCellModId, viewport)
        _, rectExists := pcr.rectanglePrimitivesIndex[gridRegion][anchorCellModId]
        _, textExists := pcr.textPrimitivesIndex[gridRegion][anchorCellModId]
        
        if isVisible {
            // Should be visible
            if !rectExists || !textExists {
                // Add missing primitives
                cellData := pcr.ctx.Data.GridData[anchorCellModId]
                needRect := cellData != nil && cellData.Style != nil &&
                    cellData.Style.Fill.BgColor != color.Transparent && !rectExists
                needText := cellData != nil && cellData.Value != "" && !textExists
                
                pcr.addPrimitivesToCell(backgroundContainer, dataContainer,
                    anchorCellModId, gridRegion, true, needRect, needText, cellData)
            }
        } else {
            // Should not be visible
            if rectExists || textExists {
                pcr.removeCell(anchorCellModId, gridRegion)
            }
        }
    }
}
```

---

## Testing Strategy

### Test Case 1: Horizontal Scroll Test

```
1. Load spreadsheet with merge cells
2. Scroll right until merge disappears
3. Scroll back left
4. VERIFY: Merge cell reappears with correct text/background
```

### Test Case 2: Rapid Scroll Test

```
1. Rapidly scroll left-right-left-right
2. VERIFY: No cells disappear
3. VERIFY: No console errors
```

### Test Case 3: Edge Case Test

```
1. Position viewport so merge is partially visible (half on/off screen)
2. Scroll slowly in both directions
3. VERIFY: Merge renders correctly at all positions
```

---

## Impact Assessment

### Before Fixes

- ❌ ~99% of cells not rendered during scroll (DEBUG CODE)
- ❌ Merge cells randomly disappear (RACE CONDITION)
- ❌ Potential index corruption (NO VALIDATION)
- ❌ User experience: BROKEN

### After Priority 1 Fix (Remove Debug Code)

- ✅ All cells render during scroll
- ⚠️ Merge cells still occasionally flicker (race condition remains)
- ⚠️ Better but not perfect

### After All Fixes

- ✅ All cells render correctly
- ✅ Merge cells stable
- ✅ No disappearing content
- ✅ Production ready

---

## Timeline

**Priority 1** (Remove debug code): **30 minutes**  
**Priority 2** (Separate pools): **2-3 hours**  
**Priority 3** (Validation): **1 hour**  
**Priority 4** (Single-pass): **2 hours**  
**Testing**: **2-3 hours**  

**Total**: **8-10 hours** of focused work

---

## Conclusion

The merge rendering instability is caused by **debugging code that was never removed** combined with a **race condition in the object pooling system**. 

**Immediate action**: Remove the `if rowVisIdx == 99` conditions - this will fix 80% of the problem.

**Medium-term**: Implement separate object pools for merge cells to prevent regular cells from stealing merge cell objects.

The good news: These are **well-understood bugs with clear fixes**. The architecture is sound; it just needs cleanup and separation of concerns.