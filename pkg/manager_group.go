package pkg

import (
	"sort"
)

type GroupIndicatorType int

// Constants
const (
	IND_NORMAL GroupIndicatorType = iota // 0: Main scrollable data
	IND_NOT_VISIBLE
	IND_VISIBLE_RANGE
	IND_VISIBLE_CONTROL
)

type RowGroup struct {
	ID          int // Unique identifier for this group
	ModIdxStart int
	ModIdxEnd   int
	Level       int  // Outline level (1-7)
	ParentID    int  // ID of parent group, -1 if top-level
	IsCollapsed bool // Current collapse state
	ControlRow  int  // Row where +/- button appears (EndModelRow + 1)
}

type ColGroup struct {
	ID          int
	ModIdxStart int // 0-based
	ModIdxEnd   int // 0-based
	Level       int
	ParentID    int
	IsCollapsed bool
	ControlCol  int
}

type GroupState struct {
	RangeVisible   bool
	ControlVisible bool
}

type GroupManager struct {
	rowGroups     []RowGroup
	colGroups     []ColGroup
	rowGroupIndex map[int]*RowGroup
	rowGroupState map[int]GroupIndicatorType

	colGroupIndex map[int]*ColGroup
	colGroupState map[int]GroupIndicatorType
}

func NewGroupManager() *GroupManager {
	return &GroupManager{
		rowGroups:     []RowGroup{},
		colGroups:     []ColGroup{},
		rowGroupIndex: make(map[int]*RowGroup),
		rowGroupState: make(map[int]GroupIndicatorType),
		colGroupIndex: make(map[int]*ColGroup),
		colGroupState: make(map[int]GroupIndicatorType),
	}
}

// Group operations
func (gm *GroupManager) ExpandGroup(groupIndex int, isRow bool) {
	if isRow {
		gm.rowGroups[groupIndex].IsCollapsed = false
	} else {
		gm.colGroups[groupIndex].IsCollapsed = false
	}
	// Update CoordinateManager
	//gm.cm.SetRowGroups(gm.rowGroups)
}

func (gm *GroupManager) CollapseGroup(groupIndex int, isRow bool) {
	if isRow {
		gm.rowGroups[groupIndex].IsCollapsed = true
	} else {
		gm.colGroups[groupIndex].IsCollapsed = true
	}
}

func (gm *GroupManager) GetOutlineDepth(isRow bool) int {
	return 0
}

func (gm *GroupManager) buildRowGroupsFromOutlineLevels(outlineLevels map[int]int, hiddenRows map[int]bool) {
	if len(outlineLevels) == 0 {
		gm.rowGroups = []RowGroup{}
		gm.rowGroupIndex = make(map[int]*RowGroup)
		return
	}

	// Step 1: Sort row indices
	rows := make([]int, 0, len(outlineLevels))
	for row := range outlineLevels {
		rows = append(rows, row)
	}
	sort.Ints(rows)

	// Step 2: Find max level
	maxLevel := 0
	for _, level := range outlineLevels {
		if level > maxLevel {
			maxLevel = level
		}
	}

	// Step 3: Create groups for each level (from lowest to highest)
	type groupRange struct {
		start, end, level int
	}
	var ranges []groupRange

	for targetLevel := 1; targetLevel <= maxLevel; targetLevel++ {
		i := 0
		for i < len(rows) {
			// Skip rows below this target level
			if outlineLevels[rows[i]] < targetLevel {
				i++
				continue
			}

			// Found start of a sequence at this level
			start := rows[i]
			end := start

			// Extend while consecutive AND at target level or higher
			j := i + 1
			for j < len(rows) &&
				rows[j] == rows[j-1]+1 &&
				outlineLevels[rows[j]] >= targetLevel {
				end = rows[j]
				j++
			}

			// Create group for this sequence at this level
			ranges = append(ranges, groupRange{
				start: start,
				end:   end,
				level: targetLevel,
			})

			// Move to end of this sequence
			i = j
		}
	}

	// Step 4: Convert to RowGroup with hierarchy
	groups := make([]RowGroup, 0, len(ranges))
	groupIndex := make(map[int]*RowGroup)

	for id, rng := range ranges {
		grp := RowGroup{
			ID:          id,
			ModIdxStart: rng.start,
			ModIdxEnd:   rng.end,
			Level:       rng.level,
			ParentID:    -1,
			IsCollapsed: false,
			ControlRow:  rng.end + 1,
		}

		// Find parent: look for containing group at IMMEDIATELY lower level
		for j := 0; j < id; j++ {
			candidate := &groups[j]
			// Parent must have level exactly 1 less than child
			if candidate.Level == grp.Level-1 &&
				candidate.ModIdxStart <= grp.ModIdxStart &&
				candidate.ModIdxEnd >= grp.ModIdxEnd {
				grp.ParentID = j
				break // Found immediate parent
			}
		}

		rowModIdxStart := grp.ModIdxStart
		rowModIdxEnd := grp.ModIdxEnd
		collapsed := true
		for i := rowModIdxStart; i <= rowModIdxEnd; i++ {
			if !hiddenRows[i] {
				collapsed = false
				break
			}
		}
		grp.IsCollapsed = collapsed
		groups = append(groups, grp)
		groupIndex[grp.ID] = &groups[len(groups)-1]
	}

	gm.rowGroups = groups
	gm.rowGroupIndex = groupIndex

	//for id, outline := range outlineLevels {
	//	fmt.Printf("%s, %s, %d, %s, %d\n", "OUTLINE:", "modIdx", id, "level", outline)
	//}

	//for _, grp := range gm.rowGroupIndex {
	//	fmt.Printf("%s,%s, %d, %s, %d, %s, %d, %s, %d, %s, %d\n", "GroupIndex:", "id", grp.ID, "controlRow", grp.ControlRow, "Parent", grp.ParentID, "modStart", grp.ModIdxStart, "modEnd", grp.ModIdxEnd)
	//}

}

// Similar for columns
func (gm *GroupManager) buildColGroupsFromOutlineLevels(outlineLevels map[int]int, hiddenCols map[int]bool) {
	if len(outlineLevels) == 0 {
		gm.colGroups = []ColGroup{}
		gm.colGroupIndex = make(map[int]*ColGroup)
		return
	}

	// Step 1: Sort row indices
	cols := make([]int, 0, len(outlineLevels))
	for col := range outlineLevels {
		cols = append(cols, col)
	}
	sort.Ints(cols)

	// Step 2: Find max level
	maxLevel := 0
	for _, level := range outlineLevels {
		if level > maxLevel {
			maxLevel = level
		}
	}

	// Step 3: Create groups for each level (from lowest to highest)
	type groupRange struct {
		start, end, level int
	}
	var ranges []groupRange

	for targetLevel := 1; targetLevel <= maxLevel; targetLevel++ {
		i := 0
		for i < len(cols) {
			// Skip rows below this target level
			if outlineLevels[cols[i]] < targetLevel {
				i++
				continue
			}

			// Found start of a sequence at this level
			start := cols[i]
			end := start

			// Extend while consecutive AND at target level or higher
			j := i + 1
			for j < len(cols) &&
				cols[j] == cols[j-1]+1 &&
				outlineLevels[cols[j]] >= targetLevel {
				end = cols[j]
				j++
			}

			// Create group for this sequence at this level
			ranges = append(ranges, groupRange{
				start: start,
				end:   end,
				level: targetLevel,
			})

			// Move to end of this sequence
			i = j
		}
	}

	// Step 4: Convert to RowGroup with hierarchy
	groups := make([]ColGroup, 0, len(ranges))
	groupIndex := make(map[int]*ColGroup)

	for id, rng := range ranges {
		grp := ColGroup{
			ID:          id,
			ModIdxStart: rng.start,
			ModIdxEnd:   rng.end,
			Level:       rng.level,
			ParentID:    -1,
			IsCollapsed: false,
			ControlCol:  rng.end + 1,
		}

		// Find parent: look for containing group at IMMEDIATELY lower level
		for j := 0; j < id; j++ {
			candidate := &groups[j]
			// Parent must have level exactly 1 less than child
			if candidate.Level == grp.Level-1 &&
				candidate.ModIdxStart <= grp.ModIdxStart &&
				candidate.ModIdxEnd >= grp.ModIdxEnd {
				grp.ParentID = j
				break // Found immediate parent
			}
		}
		colModIdxStart := grp.ModIdxStart
		colModIdxEnd := grp.ModIdxEnd
		collapsed := true
		for i := colModIdxStart; i <= colModIdxEnd; i++ {
			if !hiddenCols[i] {
				collapsed = false
				break
			}
		}
		grp.IsCollapsed = collapsed

		groups = append(groups, grp)
		groupIndex[grp.ID] = &groups[len(groups)-1]
	}

	gm.colGroups = groups
	gm.colGroupIndex = groupIndex
}

func (gm *GroupManager) IsColGroupHidden(groupID int) bool {
	if gm.isColGroupCollapsed(groupID) {

	}
	return true
}

func (gm *GroupManager) isColGroupCollapsed(groupID int) bool {
	grp, exists := gm.colGroupIndex[groupID]
	if !exists {
		return false
	}

	if grp.IsCollapsed {
		return true
	}

	if grp.ParentID != -1 {
		return gm.isColGroupCollapsed(grp.ParentID)
	}

	return false
}

func (gm *GroupManager) isRowGroupCollapsed(groupID int) bool {
	grp, exists := gm.rowGroupIndex[groupID]
	if !exists {
		return false
	}

	if grp.IsCollapsed {
		return true
	}

	if grp.ParentID != -1 {
		return gm.isRowGroupCollapsed(grp.ParentID)
	}

	return false
}

func (gm *GroupManager) GetMaxRowGroupLevel() int {
	maxLevel := 0
	for _, grp := range gm.rowGroups {
		if grp.Level > maxLevel {
			maxLevel = grp.Level
		}
	}
	return maxLevel
}

func (gm *GroupManager) GetMaxColGroupLevel() int {
	maxLevel := 0
	for _, grp := range gm.colGroups {
		if grp.Level > maxLevel {
			maxLevel = grp.Level
		}
	}
	return maxLevel
}

func (gm *GroupManager) buildColGroupsVisibleState(cm *CoordinateManager) {

	grpStates := make(map[int]GroupIndicatorType)

	for _, grp := range gm.colGroups {
		state := GroupState{}
		if grp.ParentID != -1 && gm.isColGroupCollapsed(grp.ParentID) {
			continue
		}

		for col := grp.ModIdxStart; col <= grp.ModIdxEnd; col++ {
			if cm.GetColVisIdxFromModIdx(col) != -1 {
				//anyVisible = true
				state.RangeVisible = true
				break
			}
		}

		if cm.GetColVisIdxFromModIdx(grp.ControlCol) != -1 {
			state.ControlVisible = true
		}

		if state.ControlVisible && state.RangeVisible {
			grpStates[grp.ID] = IND_NORMAL
		} else if state.ControlVisible {
			grpStates[grp.ID] = IND_VISIBLE_CONTROL
		} else if state.RangeVisible {
			grpStates[grp.ID] = IND_VISIBLE_RANGE
		} else {
			grpStates[grp.ID] = IND_NOT_VISIBLE
		}

	}
	gm.colGroupState = grpStates
}

func (gm *GroupManager) buildRowGroupsVisibleState(cm *CoordinateManager) {

	grpStates := make(map[int]GroupIndicatorType)

	for _, grp := range gm.rowGroups {
		state := GroupState{}
		if grp.ParentID != -1 && gm.isRowGroupCollapsed(grp.ParentID) {
			continue
		}

		for row := grp.ModIdxStart; row <= grp.ModIdxEnd; row++ {
			if cm.GetRowVisIdxFromModIdx(row) != -1 {
				//anyVisible = true
				state.RangeVisible = true
				break
			}
		}

		if cm.GetRowVisIdxFromModIdx(grp.ControlRow) != -1 {
			state.ControlVisible = true
		}

		if state.ControlVisible && state.RangeVisible {
			grpStates[grp.ID] = IND_NORMAL
		} else if state.ControlVisible {
			grpStates[grp.ID] = IND_VISIBLE_CONTROL
		} else if state.RangeVisible {
			grpStates[grp.ID] = IND_VISIBLE_RANGE
		} else {
			grpStates[grp.ID] = IND_NOT_VISIBLE
		}

	}
	gm.rowGroupState = grpStates
}

func (gm *GroupManager) GetVisibleColGroups(cm *CoordinateManager) ([]ColGroup, map[int]GroupState) {
	visible := []ColGroup{}

	grpStates := make(map[int]GroupState)

	for _, grp := range gm.colGroups {
		state := GroupState{}
		if grp.ParentID != -1 && gm.isColGroupCollapsed(grp.ParentID) {
			continue
		}

		anyVisible := false
		for col := grp.ModIdxStart; col <= grp.ModIdxEnd; col++ {
			if cm.GetColVisIdxFromModIdx(col) != -1 {
				anyVisible = true
				state.RangeVisible = true
				break
			}
		}

		// Also check control position
		if cm.GetColVisIdxFromModIdx(grp.ControlCol) != -1 {
			state.ControlVisible = true
			anyVisible = true
		}

		if anyVisible {
			visible = append(visible, grp)
			grpStates[grp.ID] = state
		}
	}

	return visible, grpStates
}

func (gm *GroupManager) GetVisibleRowGroups(cm *CoordinateManager) []RowGroup {
	visible := []RowGroup{}

	for _, grp := range gm.rowGroups {
		if grp.ParentID != -1 && gm.isRowGroupCollapsed(grp.ParentID) {
			continue
		}

		anyVisible := false
		for row := grp.ModIdxStart; row <= grp.ModIdxEnd; row++ {
			if cm.GetRowVisIdxFromModIdx(row) != -1 {
				anyVisible = true
				break
			}
		}

		if cm.GetRowVisIdxFromModIdx(grp.ControlRow) != -1 {
			anyVisible = true
		}

		if anyVisible {
			visible = append(visible, grp)
		}
	}

	return visible
}

func (gm *GroupManager) IsRowGroupIndicatorVisible(group *RowGroup, grid *WorkSheetData, viewport Viewport) bool {
	if group.ParentID != -1 && gm.isRowGroupCollapsed(group.ParentID) {
		return false
	}

	if grid.HiddenRows[group.ControlRow] {
		return false
	}

	if group.ControlRow < viewport.FirstRowVisIdx || group.ControlRow > viewport.LastRowVisIdx {
		return false
	}

	return true
}

func (gm *GroupManager) IsColGroupIndicatorVisible(group *ColGroup, grid *WorkSheetData, viewport Viewport) bool {
	if group.ParentID != -1 && gm.isColGroupCollapsed(group.ParentID) {
		return false
	}

	if grid.HiddenColumns[group.ControlCol] {
		return false
	}

	if group.ControlCol < viewport.FirstColVisIdx || group.ControlCol > viewport.LastColVisIdx {
		return false
	}

	return true
}
