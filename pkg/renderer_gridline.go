package pkg

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

type LineOrientation int

const (
	Horizontal LineOrientation = iota
	Vertical
)

var borderGridlineColor = color.NRGBA{R: 212, G: 212, B: 212, A: 225}

type SegmentRequiredFn func(primaryVisIdx, secondaryVisIdx int, region GridRegion) bool

type Edge struct {
	Startmost int // index in LineConfig of left-most segment in this row, or -1
	Endmost   int // index in LineConfig of right-most segment in this row, or -1
}

type Edges map[int]Edge // key = logical row boundary index (usually row number or row+1)

type SegmentConfig struct {
	PrimaryAxis    int // row index for horizontal; col index for vertical
	SecondaryStart int // col start for horizontal; row start for vertical
	SecondaryEnd   int // col end for horizontal; row end for vertical
	PrevLineId     int
	NextLineId     int
	UnPositioned   bool
}

type LineItems struct {
	Lines       []*canvas.Line // parallel to LineConfig — recycle these objects
	LinesConfig []SegmentConfig
	Edges       Edges
}

func NewLineItems() *LineItems {
	return &LineItems{
		Lines:       []*canvas.Line{},
		LinesConfig: []SegmentConfig{},
		Edges:       make(map[int]Edge),
	}
}

type RecycleManager struct {
	Flagger  *GridlineFlagger
	Recycler *GridlineRecycler
}

type GridlineFlagger struct {
	items []int
}

type GridLineRenderer struct {
	ctx *RenderContext

	LineItems map[GridRegion]map[LineOrientation]*LineItems
	Managers  map[GridRegion]map[LineOrientation]*RecycleManager
}

func NewGridLineRenderer(ctx *RenderContext) *GridLineRenderer {
	return &GridLineRenderer{
		ctx: ctx,
		LineItems: map[GridRegion]map[LineOrientation]*LineItems{
			RegionMain: {
				Horizontal: NewLineItems(),
				Vertical:   NewLineItems(),
			},
			RegionFixedCorner: {
				Horizontal: NewLineItems(),
				Vertical:   NewLineItems(),
			},
			RegionFrozenRows: {
				Horizontal: NewLineItems(),
				Vertical:   NewLineItems(),
			},
			RegionFrozenCols: {
				Horizontal: NewLineItems(),
				Vertical:   NewLineItems(),
			},
		},
		Managers: map[GridRegion]map[LineOrientation]*RecycleManager{
			RegionMain: {
				Horizontal: NewLineManager(),
				Vertical:   NewLineManager(),
			},
			RegionFixedCorner: {
				Horizontal: NewLineManager(),
				Vertical:   NewLineManager(),
			},
			RegionFrozenRows: {
				Horizontal: NewLineManager(),
				Vertical:   NewLineManager(),
			},
			RegionFrozenCols: {
				Horizontal: NewLineManager(),
				Vertical:   NewLineManager(),
			},
		},
	}
}

func (cyl *GridlineFlagger) Get() (int, bool) {
	if len(cyl.items) > 0 {
		last := len(cyl.items) - 1
		c := cyl.items[last]
		cyl.items = cyl.items[:last]
		return c, true
	}
	return -1, false
}

func (pf *GridlineFlagger) Put(id int) {
	pf.items = append(pf.items, id)
}

func (pf *GridlineFlagger) Reset() {
	pf.items = pf.items[:0]
}

type GridlineRecycler struct {
	items []GridlineRecyclerItem
}

type GridlineRecyclerItem struct {
	id  int
	obj *canvas.Line
}

func (cyl *GridlineRecycler) Get() (GridlineRecyclerItem, bool) {
	if len(cyl.items) > 0 {
		last := len(cyl.items) - 1
		c := cyl.items[last]
		cyl.items = cyl.items[:last]
		return c, true
	}
	obj := canvas.NewLine(borderGridlineColor)
	return GridlineRecyclerItem{id: -1, obj: obj}, false
}

func (cyl *GridlineRecycler) Put(id int) {
	item := GridlineRecyclerItem{id: id}
	cyl.items = append(cyl.items, item)
}

func (cyl *GridlineRecycler) Size() int {

	return len(cyl.items)
}

func NewLineManager() *RecycleManager {
	return &RecycleManager{
		Flagger:  NewGridlineFlagger(),
		Recycler: NewGridlineRecycler(),
	}
}

func NewGridlineRecycler() *GridlineRecycler {
	return &GridlineRecycler{
		items: []GridlineRecyclerItem{},
	}
}

func NewGridlineFlagger() *GridlineFlagger {
	return &GridlineFlagger{
		items: []int{},
	}
}

func (gl *GridLineRenderer) stashFlaggedItems(region GridRegion, orientation LineOrientation) {
	managers := gl.Managers
	flagger := managers[region][orientation].Flagger
	recycler := managers[region][orientation].Recycler

	h := gl.LineItems[region][orientation]

	for _, itemId := range flagger.items {
		recycler.Put(itemId)

		h.LinesConfig[itemId] = SegmentConfig{
			PrimaryAxis:    -1,
			SecondaryStart: -1,
			SecondaryEnd:   -1,
			PrevLineId:     -1,
			NextLineId:     -1,
			UnPositioned:   false,
		}
	}
}

func (gl *GridLineRenderer) fyneMoveStashedItems(region GridRegion, orientation LineOrientation) {
	flagger := gl.Managers[region][orientation].Flagger

	h := gl.LineItems[region][orientation]

	for _, itemId := range flagger.items {
		obj := h.Lines[itemId]
		obj.Move(fyne.NewPos(-9999, -9999))

	}
	// empty it
	flagger.Reset()
}

func (gl *GridLineRenderer) Remove(primaryVisIdx int, region GridRegion, orientation LineOrientation) {
	flagger := gl.Managers[region][orientation].Flagger

	h := gl.LineItems[region][orientation]
	edge, ok := h.Edges[primaryVisIdx]
	if !ok {
		return
	}

	currIdx := edge.Startmost
	for currIdx != -1 {
		nextIdx := h.LinesConfig[currIdx].NextLineId

		h.LinesConfig[currIdx].NextLineId = -1
		h.LinesConfig[currIdx].PrevLineId = -1

		flagger.Put(currIdx)
		currIdx = nextIdx
	}

	// zap the row edges
	delete(h.Edges, primaryVisIdx)
}

func (gl *GridLineRenderer) Add(container *fyne.Container, primaryVisIdx int, secondaryVisIdxMin, secondaryVisIdxMax int, region GridRegion, orientation LineOrientation) {
	var fn func(int, int, GridRegion) bool

	flagger := gl.Managers[region][orientation].Flagger
	recycler := gl.Managers[region][orientation].Recycler

	h := gl.LineItems[region][orientation]

	if orientation == Horizontal {
		fn = gl.isGridLineRequiredH
	} else {
		fn = gl.isGridLineRequiredV
	}

	segments := gl.getSegments(
		primaryVisIdx,
		secondaryVisIdxMin,
		secondaryVisIdxMax,
		region,
		fn,
	)

	if len(segments) == 0 {
		return
	}

	var leftmost, rightmost int = -1, -1
	var prevIdx int = -1

	for _, ns := range segments {
		var idx int
		if flaggedItemId, exist := flagger.Get(); exist {
			seg := &h.LinesConfig[flaggedItemId]
			seg.PrimaryAxis = primaryVisIdx
			seg.SecondaryStart = ns.Start
			seg.SecondaryEnd = ns.End
			seg.PrevLineId = prevIdx
			seg.NextLineId = -1
			seg.UnPositioned = true
			idx = flaggedItemId
		} else {
			primitiveGridLineRecyleItem, recycledItem := recycler.Get()

			if recycledItem {
				seg := &h.LinesConfig[primitiveGridLineRecyleItem.id]
				seg.PrimaryAxis = primaryVisIdx
				seg.SecondaryStart = ns.Start
				seg.SecondaryEnd = ns.End
				seg.PrevLineId = prevIdx
				seg.NextLineId = -1
				seg.UnPositioned = true

				idx = primitiveGridLineRecyleItem.id
			} else {
				seg := SegmentConfig{
					PrimaryAxis:    primaryVisIdx,
					SecondaryStart: ns.Start,
					SecondaryEnd:   ns.End,
					PrevLineId:     prevIdx,
					NextLineId:     -1,
					UnPositioned:   true,
				}

				h.Lines = append(h.Lines, primitiveGridLineRecyleItem.obj)
				h.LinesConfig = append(h.LinesConfig, seg)

				idx = len(h.Lines) - 1

				container.Add(primitiveGridLineRecyleItem.obj)
			}
		}

		if prevIdx != -1 {
			h.LinesConfig[prevIdx].NextLineId = idx
		}

		if leftmost == -1 {
			leftmost = idx
		}
		rightmost = idx

		prevIdx = idx

	}
	h.Edges[primaryVisIdx] = Edge{Startmost: leftmost, Endmost: rightmost}
}

func (gl *GridLineRenderer) getSegments(rowVisIdx, colVisIdxMin, colVisIdxMax int, region GridRegion, fn SegmentRequiredFn) []struct{ Start, End int } {
	var segments []struct{ Start, End int }
	var currStart int = -1
	for colVisIdx := colVisIdxMin; colVisIdx <= colVisIdxMax; colVisIdx++ {
		drawn := fn(rowVisIdx, colVisIdx, region)
		if drawn {
			if currStart == -1 {
				currStart = colVisIdx
			}
		} else {
			if currStart != -1 {
				segments = append(segments, struct{ Start, End int }{currStart, colVisIdx - 1}) // End is exclusive.
				currStart = -1
			}
		}
	}
	if currStart != -1 {
		segments = append(segments, struct{ Start, End int }{currStart, colVisIdxMax})
	}
	return segments
}

func (gl *GridLineRenderer) TrimStart(
	primaryVisIdx int,
	secondaryVisIdxMin int,
	region GridRegion,
	orientation LineOrientation) {
	flagger := gl.Managers[region][orientation].Flagger

	h := gl.LineItems[region][orientation]
	edge, ok := h.Edges[primaryVisIdx]
	if !ok {
		return
	}

	currIdx := edge.Startmost
	for currIdx != -1 {
		seg := &h.LinesConfig[currIdx]
		if seg.SecondaryEnd < secondaryVisIdxMin {
			flagger.Put(currIdx)

			edge.Startmost = seg.NextLineId // Update.
			if edge.Startmost == -1 {
				edge.Endmost = -1 // Empty row.
			} else {
				h.LinesConfig[seg.NextLineId].PrevLineId = -1
			}
		} else if seg.SecondaryStart < secondaryVisIdxMin {
			h.LinesConfig[currIdx].SecondaryStart = secondaryVisIdxMin
			h.LinesConfig[currIdx].UnPositioned = true
			break
		} else {
			break // Later ones are fine.
		}
		currIdx = seg.NextLineId
	}
	if edge.Startmost == -1 {
		delete(h.Edges, primaryVisIdx)
	} else {
		h.Edges[primaryVisIdx] = edge
	}
}

func (gl *GridLineRenderer) TrimEnd(
	primaryVisIdx int,
	secondaryVisIdxMax int,
	region GridRegion,
	orientation LineOrientation,
) {
	flagger := gl.Managers[region][orientation].Flagger

	h := gl.LineItems[region][orientation]
	edge, ok := h.Edges[primaryVisIdx]
	if !ok {
		return
	}

	// Start from the rightmost segment and walk left
	currIdx := edge.Endmost
	for currIdx != -1 {
		seg := &h.LinesConfig[currIdx]
		if seg.SecondaryStart > secondaryVisIdxMax {
			flagger.Put(currIdx)

			// Update rightmost to the previous segment
			edge.Endmost = seg.PrevLineId
			if edge.Endmost == -1 {
				edge.Startmost = -1 // Row is now empty
			} else {
				h.LinesConfig[seg.PrevLineId].NextLineId = -1
			}
		} else if seg.SecondaryEnd > secondaryVisIdxMax {
			h.LinesConfig[currIdx].SecondaryEnd = secondaryVisIdxMax
			h.LinesConfig[currIdx].UnPositioned = true
			break
		} else {
			break
		}

		currIdx = seg.PrevLineId
	}

	// Final cleanup
	if edge.Endmost == -1 {
		delete(h.Edges, primaryVisIdx)
	} else {
		h.Edges[primaryVisIdx] = edge
	}
}

func (gl *GridLineRenderer) GrowStart(
	container *fyne.Container,
	primaryVisIdx int,
	newSecondayVisIdxMin int, // the new, smaller min column (inclusive)
	region GridRegion,
	orientation LineOrientation,
) {
	var fn func(int, int, GridRegion) bool
	flagger := gl.Managers[region][orientation].Flagger
	recycler := gl.Managers[region][orientation].Recycler

	h := gl.LineItems[region][orientation]

	edge, _ := h.Edges[primaryVisIdx]

	currLeftIdx := edge.Startmost

	if orientation == Horizontal {
		fn = gl.isGridLineRequiredH
	} else {
		fn = gl.isGridLineRequiredV
	}

	segments := gl.getSegments(
		primaryVisIdx,
		newSecondayVisIdxMin,
		h.LinesConfig[currLeftIdx].SecondaryStart-1,
		region,
		fn,
	)

	n := len(segments)

	var newIdx int = -1
	for i := n - 1; i >= 0; i-- {
		segment := segments[i]
		if segment.End+1 == h.LinesConfig[currLeftIdx].SecondaryStart {
			// extend existing segment
			h.LinesConfig[currLeftIdx].SecondaryStart = segment.Start
			h.LinesConfig[currLeftIdx].UnPositioned = true
		} else {
			// create new segment
			if flaggedItemId, exist := flagger.Get(); exist {
				// hot
				h.LinesConfig[edge.Startmost].PrevLineId = flaggedItemId

				seg := &h.LinesConfig[flaggedItemId]
				seg.PrimaryAxis = primaryVisIdx
				seg.SecondaryStart = segment.Start
				seg.SecondaryEnd = segment.End
				seg.PrevLineId = -1
				seg.NextLineId = edge.Startmost
				seg.UnPositioned = true

				edge.Startmost = flaggedItemId
				h.Edges[primaryVisIdx] = edge
			} else {
				recItem, recycled := recycler.Get()
				if recycled {
					h.LinesConfig[edge.Startmost].PrevLineId = recItem.id

					seg := &h.LinesConfig[recItem.id]
					seg.PrimaryAxis = primaryVisIdx
					seg.SecondaryStart = segment.Start
					seg.SecondaryEnd = segment.End
					seg.PrevLineId = -1
					seg.NextLineId = edge.Startmost
					seg.UnPositioned = true

					edge.Startmost = recItem.id
					h.Edges[primaryVisIdx] = edge
				} else {
					h.Lines = append(h.Lines, recItem.obj)
					newIdx = len(h.Lines) - 1

					h.LinesConfig[edge.Startmost].PrevLineId = newIdx

					seg := SegmentConfig{
						PrimaryAxis:    primaryVisIdx,
						SecondaryStart: segment.Start,
						SecondaryEnd:   segment.End,
						PrevLineId:     -1,
						NextLineId:     edge.Startmost,
						UnPositioned:   true,
					}

					h.LinesConfig = append(h.LinesConfig, seg)

					edge.Startmost = newIdx
					h.Edges[primaryVisIdx] = edge

					container.Add(recItem.obj)
				}
			}
		}
	}
}

func (gl *GridLineRenderer) GrowEnd(
	container *fyne.Container,
	primaryVisIdx int,
	newSecondaryVisIdxMax int, // the new, larger max column (inclusive)
	region GridRegion,
	orientation LineOrientation,
) {
	var fn func(int, int, GridRegion) bool
	flagger := gl.Managers[region][orientation].Flagger
	recycler := gl.Managers[region][orientation].Recycler

	h := gl.LineItems[region][orientation]

	edge, _ := h.Edges[primaryVisIdx]

	currRightIdx := edge.Endmost

	if orientation == Horizontal {
		fn = gl.isGridLineRequiredH
	} else {
		fn = gl.isGridLineRequiredV
	}

	segments := gl.getSegments(
		primaryVisIdx,
		h.LinesConfig[currRightIdx].SecondaryEnd+1,
		newSecondaryVisIdxMax,
		region,
		fn,
	)

	n := len(segments)

	var newIdx int = -1
	for i := 0; i < n; i++ {
		segment := segments[i]
		if segment.Start-1 == h.LinesConfig[currRightIdx].SecondaryEnd {
			// extend existing segment
			h.LinesConfig[currRightIdx].SecondaryEnd = segment.End
			h.LinesConfig[currRightIdx].UnPositioned = true
		} else {
			if flaggedItemId, exist := flagger.Get(); exist {
				h.LinesConfig[edge.Endmost].NextLineId = flaggedItemId

				seg := &h.LinesConfig[flaggedItemId]
				seg.PrimaryAxis = primaryVisIdx
				seg.SecondaryStart = segment.Start
				seg.SecondaryEnd = segment.End
				seg.PrevLineId = edge.Endmost
				seg.NextLineId = -1
				seg.UnPositioned = true

				edge.Endmost = flaggedItemId
				h.Edges[primaryVisIdx] = edge
			} else {
				recItem, recycled := recycler.Get()
				if recycled {
					h.LinesConfig[edge.Endmost].NextLineId = recItem.id

					seg := &h.LinesConfig[recItem.id]
					seg.PrimaryAxis = primaryVisIdx
					seg.SecondaryStart = segment.Start
					seg.SecondaryEnd = segment.End
					seg.PrevLineId = edge.Endmost
					seg.NextLineId = -1
					seg.UnPositioned = true

					edge.Endmost = recItem.id
					h.Edges[primaryVisIdx] = edge
				} else {
					h.Lines = append(h.Lines, recItem.obj)
					newIdx = len(h.Lines) - 1

					h.LinesConfig[edge.Endmost].NextLineId = newIdx

					seg := SegmentConfig{
						PrimaryAxis:    primaryVisIdx,
						SecondaryStart: segment.Start,
						SecondaryEnd:   segment.End,
						PrevLineId:     edge.Endmost,
						NextLineId:     -1,
						UnPositioned:   true,
					}

					h.LinesConfig = append(h.LinesConfig, seg)

					edge.Endmost = newIdx
					h.Edges[primaryVisIdx] = edge

					container.Add(recItem.obj)
				}
			}

		}
	}
}

func (gl *GridLineRenderer) positionGridlines(region GridRegion, orientation LineOrientation, renderAbsolute bool) {
	cm := gl.ctx.CoordManager

	h := gl.LineItems[region][orientation]

	if region == RegionMain {
		for itemId, config := range h.LinesConfig {
			if config.PrimaryAxis != -1 && config.UnPositioned {
				gl.posAbsoluteGridline(itemId, config, region, orientation)
			}
		}
		return
	}

	if renderAbsolute {
		for itemId, config := range h.LinesConfig {
			if config.PrimaryAxis != -1 {
				gl.posAbsoluteGridline(itemId, config, region, orientation)
			}
		}
	} else {
		for itemId, config := range h.LinesConfig {
			if config.PrimaryAxis != -1 {
				if config.UnPositioned {
					gl.posAbsoluteGridline(itemId, config, region, orientation)
				} else {
					// DELTA movement
					item := h.Lines[itemId]
					if region == RegionFrozenCols {
						deltaY := cm.GetScrollDeltaY()
						item.Position1.Y -= deltaY
						item.Position2.Y -= deltaY
					} else if region == RegionFrozenRows {
						deltaX := cm.GetScrollDeltaX()
						item.Position1.X -= deltaX
						item.Position2.X -= deltaX
					}

				}
			}
		}
	}
}

func (gl *GridLineRenderer) posAbsoluteGridline(id int, config SegmentConfig, region GridRegion, orientation LineOrientation) {
	cm := gl.ctx.CoordManager

	h := gl.LineItems[region][orientation]
	item := h.Lines[id]

	if orientation == Horizontal {
		// Horizontal: Y is fixed
		y := cm.GetRowPixelPosEndY(region, cm.GetRowModIdxFromVisIdx(config.PrimaryAxis))
		item.Position1.Y = y
		item.Position1.X = cm.GetColPixelPosX(region, cm.GetColModIdxFromVisIdx(config.SecondaryStart))
		item.Position2.Y = y
		item.Position2.X = cm.GetColPixelPosEndX(region, cm.GetColModIdxFromVisIdx(config.SecondaryEnd))
	} else {
		// Vertical: X is fixed
		x := cm.GetColPixelPosEndX(region, cm.GetColModIdxFromVisIdx(config.PrimaryAxis))
		item.Position1.X = x
		item.Position1.Y = cm.GetRowPixelPosY(region, cm.GetRowModIdxFromVisIdx(config.SecondaryStart))
		item.Position2.X = x
		item.Position2.Y = cm.GetRowPixelPosEndY(region, cm.GetRowModIdxFromVisIdx(config.SecondaryEnd))
	}

	h.LinesConfig[id].UnPositioned = false
}

func (gl *GridLineRenderer) isGridLineRequiredH(rowVisIdx, colVisIdx int, region GridRegion) bool {

	pr := gl.ctx.PrimitiveRenderer
	cm := gl.ctx.CoordManager

	mm := gl.ctx.MergeManager

	id := CellID{Row: rowVisIdx, Col: colVisIdx}
	if mergeVisRange, exists := mm.GetMergedRangeByVisId(id); exists {
		if rowVisIdx == mergeVisRange.VisRowEnd {
			anchorVisIdx := CellID{mergeVisRange.VisRowStart, mergeVisRange.VisColStart}
			if hasBackground := mm.hasMergeRangeBackgroundByVisAnchorId(anchorVisIdx); hasBackground {
				return false
			}
		} else {
			// in the merged range somewhere
			return false
		}
	} else {
		rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
		colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
		cellID := CellID{Row: rowModIdx, Col: colModIdx}

		if _, hasBackground := pr.rectanglePrimitivesIndex[region][cellID]; hasBackground {
			return false
		}
	}

	// check the other cell, bordering the gridline
	rowVisIdx++
	id = CellID{Row: rowVisIdx, Col: colVisIdx}
	if mergeVisRange, exists := mm.GetMergedRangeByVisId(id); exists {
		if rowVisIdx == mergeVisRange.VisRowStart {
			anchorVisIdx := CellID{mergeVisRange.VisRowStart, mergeVisRange.VisColStart}
			if hasBackground := mm.hasMergeRangeBackgroundByVisAnchorId(anchorVisIdx); hasBackground {
				return false
			}
		} else {
			// in the merged range somewhere
			return false
		}
	} else {
		if rowModIdx, safe := cm.GetRowModIdxFromVisIdxSafe(rowVisIdx); safe {
			colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
			cellID := CellID{Row: rowModIdx, Col: colModIdx}

			if _, hasBackground := pr.rectanglePrimitivesIndex[region][cellID]; hasBackground {
				return false
			}
		}

	}
	return true
}

func (gl *GridLineRenderer) isGridLineRequiredV(rowVisIdx, colVisIdx int, region GridRegion) bool {

	pr := gl.ctx.PrimitiveRenderer
	cm := gl.ctx.CoordManager

	mm := gl.ctx.MergeManager

	id := CellID{Row: rowVisIdx, Col: colVisIdx}
	if mergeVisRange, exists := mm.GetMergedRangeByVisId(id); exists {
		if rowVisIdx == mergeVisRange.VisRowEnd {
			anchorVisIdx := CellID{mergeVisRange.VisRowStart, mergeVisRange.VisColStart}
			if hasBackground := mm.hasMergeRangeBackgroundByVisAnchorId(anchorVisIdx); hasBackground {
				return false
			}
		} else {
			// in the merged range somewhere
			return false
		}
	} else {
		rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
		colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
		cellID := CellID{Row: rowModIdx, Col: colModIdx}

		if _, hasBackground := pr.rectanglePrimitivesIndex[region][cellID]; hasBackground {
			return false
		}
	}

	// check the other cell, bordering the gridline
	rowVisIdx++
	id = CellID{Row: rowVisIdx, Col: colVisIdx}
	if mergeVisRange, exists := mm.GetMergedRangeByVisId(id); exists {
		if rowVisIdx == mergeVisRange.VisRowStart {
			anchorVisIdx := CellID{mergeVisRange.VisRowStart, mergeVisRange.VisColStart}
			if hasBackground := mm.hasMergeRangeBackgroundByVisAnchorId(anchorVisIdx); hasBackground {
				return false
			}
		} else {
			// in the merged range somewhere
			return false
		}
	} else {
		if rowModIdx, safe := cm.GetRowModIdxFromVisIdxSafe(rowVisIdx); safe {
			colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
			cellID := CellID{Row: rowModIdx, Col: colModIdx}

			if _, hasBackground := pr.rectanglePrimitivesIndex[region][cellID]; hasBackground {
				return false
			}
		}

	}
	return true
}
