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

var borderGridlineColor = color.NRGBA{R: 212, G: 212, B: 212, A: 215} // slightly non-opaque alpha

type GLSegmentRequiredFn func(primaryVisIdx, secondaryVisIdx int, region GridRegion) bool

type GLEdge struct {
	Startmost int // index in LineConfig of start-most segment in this row or col, or -1
	Endmost   int // index in LineConfig of end-most segment in this row or col, or -1
}

type GLEdges map[int]GLEdge // key = logical row boundary index (usually row number or row+1)

type GLSegmentConfig struct {
	PrimaryAxis    int // row index for horizontal; col index for vertical
	SecondaryStart int // col start for horizontal; row start for vertical
	SecondaryEnd   int // col end for horizontal; row end for vertical
	PrevLineId     int
	NextLineId     int
	UnPositioned   bool
}

type GLLineItems struct {
	Lines       []*canvas.Line // parallel to LineConfig — recycle these objects
	LinesConfig []GLSegmentConfig
	Edges       GLEdges
}

func NewLineItems() *GLLineItems {
	return &GLLineItems{
		Lines:       []*canvas.Line{},
		LinesConfig: []GLSegmentConfig{},
		Edges:       make(map[int]GLEdge),
	}
}

type GLRecycleManager struct {
	HotPool  *GLHotPool
	ColdPool *GLColdPool
}

type GLHotPool struct {
	items []int
}

type GLColdPool struct {
	items []int
}

type GridLineRenderer struct {
	ctx *RenderContext

	LineItems        map[GridRegion]map[LineOrientation]*GLLineItems
	Managers         map[GridRegion]map[LineOrientation]*GLRecycleManager
	PendingPositions map[GridRegion]map[LineOrientation]int
}

func NewGridLineRenderer(ctx *RenderContext) *GridLineRenderer {
	return &GridLineRenderer{
		ctx: ctx,
		LineItems: map[GridRegion]map[LineOrientation]*GLLineItems{
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
		Managers: map[GridRegion]map[LineOrientation]*GLRecycleManager{
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
		PendingPositions: map[GridRegion]map[LineOrientation]int{
			RegionMain:        {Horizontal: 0, Vertical: 0},
			RegionFixedCorner: {Horizontal: 0, Vertical: 0},
			RegionFrozenRows:  {Horizontal: 0, Vertical: 0},
			RegionFrozenCols:  {Horizontal: 0, Vertical: 0},
		},
	}
}

func (cyl *GLHotPool) Get() (int, bool) {
	if len(cyl.items) > 0 {
		last := len(cyl.items) - 1
		c := cyl.items[last]
		cyl.items = cyl.items[:last]
		return c, true
	}
	return -1, false
}

func (pf *GLHotPool) Put(id int) {
	pf.items = append(pf.items, id)
}

func (pf *GLHotPool) Reset() {
	pf.items = pf.items[:0]
}

func (cyl *GLColdPool) Get() (int, bool) {
	if len(cyl.items) > 0 {
		last := len(cyl.items) - 1
		c := cyl.items[last]
		cyl.items = cyl.items[:last]
		return c, true
	}
	return -1, false
}

func (cyl *GLColdPool) Put(id int) {
	cyl.items = append(cyl.items, id)
}

func (cyl *GLColdPool) Size() int {
	return len(cyl.items)
}

func NewLineManager() *GLRecycleManager {
	return &GLRecycleManager{
		HotPool:  NewGridlineHotPool(),
		ColdPool: NewGridlineColdPool(),
	}
}

func NewGridlineColdPool() *GLColdPool {
	return &GLColdPool{
		items: []int{},
	}
}

func NewGridlineHotPool() *GLHotPool {
	return &GLHotPool{
		items: []int{},
	}
}

func (gl *GridLineRenderer) SetWaterMark(region GridRegion, orientation LineOrientation) {
	gl.PendingPositions[region][orientation] = len(gl.LineItems[region][orientation].Lines)
}
func (gl *GridLineRenderer) fyneMoveHotPoolItems(region GridRegion, orientation LineOrientation) {
	managers := gl.Managers
	hotPool := managers[region][orientation].HotPool
	coldPool := managers[region][orientation].ColdPool

	h := gl.LineItems[region][orientation]

	for _, itemId := range hotPool.items {
		coldPool.Put(itemId)

		h.LinesConfig[itemId] = GLSegmentConfig{
			PrimaryAxis:    -1,
			SecondaryStart: -1,
			SecondaryEnd:   -1,
			PrevLineId:     -1,
			NextLineId:     -1,
			UnPositioned:   false,
		}
		obj := h.Lines[itemId]
		obj.Move(fyne.NewPos(-9999, -9999))
	}
	hotPool.Reset()
}

func (gl *GridLineRenderer) fyneAddContent(c *fyne.Container, region GridRegion, o LineOrientation) {
	from := gl.PendingPositions[region][o]
	lines := gl.LineItems[region][o].Lines
	for i := from; i < len(lines); i++ {
		lines[i] = canvas.NewLine(borderGridlineColor)
		c.Add(lines[i])
	}
}

func (gl *GridLineRenderer) Remove(primaryVisIdx int, region GridRegion, orientation LineOrientation) {
	hotPool := gl.Managers[region][orientation].HotPool

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

		hotPool.Put(currIdx)
		currIdx = nextIdx
	}
	delete(h.Edges, primaryVisIdx)
}

func (gl *GridLineRenderer) Add(primaryVisIdx int, secondaryVisIdxMin, secondaryVisIdxMax int, region GridRegion, orientation LineOrientation) {
	var fn func(int, int, GridRegion) bool

	hotPool := gl.Managers[region][orientation].HotPool
	coldPool := gl.Managers[region][orientation].ColdPool

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
		if flaggedItemId, exist := hotPool.Get(); exist {
			seg := &h.LinesConfig[flaggedItemId]
			seg.PrimaryAxis = primaryVisIdx
			seg.SecondaryStart = ns.Start
			seg.SecondaryEnd = ns.End
			seg.PrevLineId = prevIdx
			seg.NextLineId = -1
			seg.UnPositioned = true
			idx = flaggedItemId
		} else {
			primitiveGridLineRecyleItemId, recycledItem := coldPool.Get()

			if recycledItem {
				seg := &h.LinesConfig[primitiveGridLineRecyleItemId]
				seg.PrimaryAxis = primaryVisIdx
				seg.SecondaryStart = ns.Start
				seg.SecondaryEnd = ns.End
				seg.PrevLineId = prevIdx
				seg.NextLineId = -1
				seg.UnPositioned = true

				idx = primitiveGridLineRecyleItemId
			} else {
				seg := GLSegmentConfig{
					PrimaryAxis:    primaryVisIdx,
					SecondaryStart: ns.Start,
					SecondaryEnd:   ns.End,
					PrevLineId:     prevIdx,
					NextLineId:     -1,
					UnPositioned:   true,
				}

				h.Lines = append(h.Lines, nil)
				h.LinesConfig = append(h.LinesConfig, seg)

				idx = len(h.Lines) - 1
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
	h.Edges[primaryVisIdx] = GLEdge{Startmost: leftmost, Endmost: rightmost}
}

func (gl *GridLineRenderer) getSegments(primaryVisIdx, secondaryVisIdxMin, secondaryVisIdxMax int, region GridRegion, fn GLSegmentRequiredFn) []struct{ Start, End int } {
	var segments []struct{ Start, End int }
	var currStart int = -1
	for visIdx := secondaryVisIdxMin; visIdx <= secondaryVisIdxMax; visIdx++ {
		drawn := fn(primaryVisIdx, visIdx, region)
		if drawn {
			if currStart == -1 {
				currStart = visIdx
			}
		} else {
			if currStart != -1 {
				segments = append(segments, struct{ Start, End int }{currStart, visIdx - 1}) // End is exclusive.
				currStart = -1
			}
		}
	}
	if currStart != -1 {
		segments = append(segments, struct{ Start, End int }{currStart, secondaryVisIdxMax})
	}
	return segments
}

func (gl *GridLineRenderer) TrimStart(
	primaryVisIdx int,
	secondaryVisIdxMin int,
	region GridRegion,
	orientation LineOrientation) {
	hotPool := gl.Managers[region][orientation].HotPool

	h := gl.LineItems[region][orientation]
	edge, ok := h.Edges[primaryVisIdx]
	if !ok {
		return
	}

	currIdx := edge.Startmost
	for currIdx != -1 {
		seg := &h.LinesConfig[currIdx]
		if seg.SecondaryEnd < secondaryVisIdxMin {
			h.LinesConfig[currIdx].NextLineId = -1
			h.LinesConfig[currIdx].PrevLineId = -1
			hotPool.Put(currIdx)

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
	hotPool := gl.Managers[region][orientation].HotPool

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
			h.LinesConfig[currIdx].NextLineId = -1
			h.LinesConfig[currIdx].PrevLineId = -1
			hotPool.Put(currIdx)

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
	primaryVisIdx int,
	newSecondayVisIdxMin int,
	region GridRegion,
	orientation LineOrientation,
) {
	var fn func(int, int, GridRegion) bool
	hotPool := gl.Managers[region][orientation].HotPool
	coldPool := gl.Managers[region][orientation].ColdPool

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

	for i := n - 1; i >= 0; i-- {
		segment := segments[i]
		if segment.End+1 == h.LinesConfig[currLeftIdx].SecondaryStart {
			// extend existing segment
			h.LinesConfig[currLeftIdx].SecondaryStart = segment.Start
			h.LinesConfig[currLeftIdx].UnPositioned = true
		} else {
			// create new segment
			if flaggedItemId, exist := hotPool.Get(); exist {
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
				recItemId, recycled := coldPool.Get()
				if recycled {
					h.LinesConfig[edge.Startmost].PrevLineId = recItemId

					seg := &h.LinesConfig[recItemId]
					seg.PrimaryAxis = primaryVisIdx
					seg.SecondaryStart = segment.Start
					seg.SecondaryEnd = segment.End
					seg.PrevLineId = -1
					seg.NextLineId = edge.Startmost
					seg.UnPositioned = true

					edge.Startmost = recItemId
					h.Edges[primaryVisIdx] = edge
				} else {

					h.Lines = append(h.Lines, nil)
					h.LinesConfig[edge.Startmost].PrevLineId = len(h.Lines) - 1

					seg := GLSegmentConfig{
						PrimaryAxis:    primaryVisIdx,
						SecondaryStart: segment.Start,
						SecondaryEnd:   segment.End,
						PrevLineId:     -1,
						NextLineId:     edge.Startmost,
						UnPositioned:   true,
					}

					h.LinesConfig = append(h.LinesConfig, seg)

					edge.Startmost = len(h.Lines) - 1
					h.Edges[primaryVisIdx] = edge
				}
			}
		}
	}
}

func (gl *GridLineRenderer) GrowEnd(
	primaryVisIdx int,
	newSecondaryVisIdxMax int, // the new, larger max column (inclusive)
	region GridRegion,
	orientation LineOrientation,
) {
	var fn func(int, int, GridRegion) bool
	hotPool := gl.Managers[region][orientation].HotPool
	coldPool := gl.Managers[region][orientation].ColdPool

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

	for i := 0; i < n; i++ {
		segment := segments[i]
		if segment.Start-1 == h.LinesConfig[currRightIdx].SecondaryEnd {
			// extend existing segment
			h.LinesConfig[currRightIdx].SecondaryEnd = segment.End
			h.LinesConfig[currRightIdx].UnPositioned = true
		} else {
			if flaggedItemId, exist := hotPool.Get(); exist {
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
				recItemId, recycled := coldPool.Get()
				if recycled {
					h.LinesConfig[edge.Endmost].NextLineId = recItemId

					seg := &h.LinesConfig[recItemId]
					seg.PrimaryAxis = primaryVisIdx
					seg.SecondaryStart = segment.Start
					seg.SecondaryEnd = segment.End
					seg.PrevLineId = edge.Endmost
					seg.NextLineId = -1
					seg.UnPositioned = true

					edge.Endmost = recItemId
					h.Edges[primaryVisIdx] = edge
				} else {
					h.Lines = append(h.Lines, nil)
					h.LinesConfig[edge.Endmost].NextLineId = len(h.Lines) - 1

					seg := GLSegmentConfig{
						PrimaryAxis:    primaryVisIdx,
						SecondaryStart: segment.Start,
						SecondaryEnd:   segment.End,
						PrevLineId:     edge.Endmost,
						NextLineId:     -1,
						UnPositioned:   true,
					}

					h.LinesConfig = append(h.LinesConfig, seg)

					edge.Endmost = len(h.Lines) - 1
					h.Edges[primaryVisIdx] = edge
				}
			}

		}
	}
}

func (gl *GridLineRenderer) fynePositionGridlines(region GridRegion, orientation LineOrientation, renderAbsolute bool) {
	cm := gl.ctx.CoordManager

	li := gl.LineItems[region][orientation]

	if region == RegionMain {
		for itemId, config := range li.LinesConfig {
			if config.PrimaryAxis != -1 && config.UnPositioned {
				gl.posAbsoluteGridline(itemId, config, region, orientation, li)
			}
		}
		return
	}

	if renderAbsolute {
		for itemId, config := range li.LinesConfig {
			if config.PrimaryAxis != -1 {
				gl.posAbsoluteGridline(itemId, config, region, orientation, li)
			}
		}
	} else {
		for itemId, config := range li.LinesConfig {
			if config.PrimaryAxis != -1 {
				if config.UnPositioned {
					gl.posAbsoluteGridline(itemId, config, region, orientation, li)
				} else {
					// DELTA movement
					item := li.Lines[itemId]
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

func (gl *GridLineRenderer) posAbsoluteGridline(id int, config GLSegmentConfig, region GridRegion, orientation LineOrientation, h *GLLineItems) {
	cm := gl.ctx.CoordManager

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
				//fmt.Printf("boundary suppressed: rowVisIdx=%d colVisIdx=%d anchor=%+v\n", rowVisIdx, colVisIdx, anchorVisIdx)
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
				//fmt.Printf("boundary suppressed: rowVisIdx=%d colVisIdx=%d anchor=%+v\n", rowVisIdx, colVisIdx, anchorVisIdx)
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

func (gl *GridLineRenderer) isGridLineRequiredV(colVisIdx, rowVisIdx int, region GridRegion) bool {

	pr := gl.ctx.PrimitiveRenderer
	cm := gl.ctx.CoordManager

	mm := gl.ctx.MergeManager

	id := CellID{Row: rowVisIdx, Col: colVisIdx}
	if mergeVisRange, exists := mm.GetMergedRangeByVisId(id); exists {
		if colVisIdx == mergeVisRange.VisColEnd { //if rowVisIdx == mergeVisRange.VisRowEnd {
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
	colVisIdx++
	id = CellID{Row: rowVisIdx, Col: colVisIdx}
	if mergeVisRange, exists := mm.GetMergedRangeByVisId(id); exists {
		if colVisIdx == mergeVisRange.VisColStart { // if rowVisIdx == mergeVisRange.VisRowStart {
			anchorVisIdx := CellID{mergeVisRange.VisRowStart, mergeVisRange.VisColStart}
			if hasBackground := mm.hasMergeRangeBackgroundByVisAnchorId(anchorVisIdx); hasBackground {
				return false
			}
		} else {
			// in the merged range somewhere
			return false
		}
	} else {
		if colModIdx, safe := cm.GetColModIdxFromVisIdxSafe(colVisIdx); safe {
			rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
			cellID := CellID{Row: rowModIdx, Col: colModIdx}

			if _, hasBackground := pr.rectanglePrimitivesIndex[region][cellID]; hasBackground {
				return false
			}
		}
	}
	return true
}
