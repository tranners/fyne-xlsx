package pkg

type RegionRenderer struct {
	ctx               *RenderContext
	fontManager       *FontManager
	primitiveRenderer *PrimitiveRenderer
	borderRenderer    *BorderLineRenderer
	gridlineRenderer  *GridLineRenderer
	frameCounters     map[GridRegion]int
}

func NewRegionRenderer(ctx *RenderContext, fontMgr *FontManager) *RegionRenderer {
	primitiveRenderer := NewPrimitiveRenderer(ctx, fontMgr)
	ctx.PrimitiveRenderer = primitiveRenderer
	return &RegionRenderer{
		ctx:               ctx,
		fontManager:       fontMgr,
		borderRenderer:    NewBorderLineRenderer(ctx),
		primitiveRenderer: primitiveRenderer,
		gridlineRenderer:  NewGridLineRenderer(ctx),
		frameCounters:     make(map[GridRegion]int),
	}
}

func (r *GridRenderer) renderRegion(
	region GridRegion,
	containers RegionContainers,
	forceFullRender bool,
	scrollChange ScrollChange,
) {

	ctx := r.context
	rr := r.regionRenderer

	if !ctx.PaneHasRenderedOnce[region] {
		// This region hasn't had initial render yet
		if r.isValidViewport(region) {
			// Viewport is ready → Do FULL render
			rr.renderCellsRegionFull(containers, region)
			rr.renderGridlinesRegionFull(containers, region)

			ctx.PaneHasRenderedOnce[region] = true
		}
	} else {
		// Use DELTA render from now on
		shouldRender := forceFullRender

		// Check if this region needs update based on scroll
		switch region {
		case RegionMain:
			shouldRender = shouldRender || scrollChange.X || scrollChange.Y
		case RegionFrozenRows:
			shouldRender = shouldRender || scrollChange.X
		case RegionFrozenCols:
			shouldRender = shouldRender || scrollChange.Y
		case RegionFixedCorner:
			// Corner only updates on forceFullRender
		}

		if shouldRender {
			rr.renderCellsRegionDelta(containers, region)

			rr.renderGridlinesRegionDelta(containers, region)
		}
	}
}

func (rr *RegionRenderer) removeStandardCellsOutsideViewport(vpCurrent, vpPrevious Viewport, region GridRegion) {
	var remainingRowStart, remainingRowEnd int
	cm := rr.ctx.CoordManager
	pr := rr.primitiveRenderer

	for rowVisIdx := vpPrevious.FirstRowVisIdx; rowVisIdx < vpCurrent.FirstRowVisIdx; rowVisIdx++ {
		rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
		for colVisIdx := vpPrevious.FirstColVisIdx; colVisIdx <= vpPrevious.LastColVisIdx; colVisIdx++ {
			colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
			pr.removeCell(CellID{Row: rowModIdx, Col: colModIdx}, region)
		}
	}
	for rowVisIdx := vpCurrent.LastRowVisIdx + 1; rowVisIdx <= vpPrevious.LastRowVisIdx; rowVisIdx++ {
		rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
		for colVisIdx := vpPrevious.FirstColVisIdx; colVisIdx <= vpPrevious.LastColVisIdx; colVisIdx++ {
			colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
			pr.removeCell(CellID{Row: rowModIdx, Col: colModIdx}, region)
		}
	}

	remainingRowStart = max(vpPrevious.FirstRowVisIdx, vpCurrent.FirstRowVisIdx)
	remainingRowEnd = min(vpPrevious.LastRowVisIdx, vpCurrent.LastRowVisIdx)

	for colVisIdx := vpPrevious.FirstColVisIdx; colVisIdx < vpCurrent.FirstColVisIdx; colVisIdx++ {
		colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
		for rowVisIdx := remainingRowStart; rowVisIdx <= remainingRowEnd; rowVisIdx++ {
			rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
			pr.removeCell(CellID{Row: rowModIdx, Col: colModIdx}, region)
		}
	}
	// scroll up
	for colVisIdx := vpCurrent.LastColVisIdx + 1; colVisIdx <= vpPrevious.LastColVisIdx; colVisIdx++ {
		colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
		for rowVisIdx := remainingRowStart; rowVisIdx <= remainingRowEnd; rowVisIdx++ {
			rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
			pr.removeCell(CellID{Row: rowModIdx, Col: colModIdx}, region)
		}
	}

}

func (rr *RegionRenderer) addStandardCellsInViewport(regionContainers RegionContainers, vpCurrent, vpPrevious Viewport, region GridRegion) {
	var remainingRowStart, remainingRowEnd int
	cm := rr.ctx.CoordManager
	pr := rr.primitiveRenderer
	for rowVisIdx := vpCurrent.FirstRowVisIdx; rowVisIdx < vpPrevious.FirstRowVisIdx; rowVisIdx++ {
		rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
		for colVisIdx := vpCurrent.FirstColVisIdx; colVisIdx <= vpCurrent.LastColVisIdx; colVisIdx++ {
			colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
			pr.renderCell(regionContainers, rowModIdx, colModIdx, region)
		}
	}

	for rowVisIdx := vpPrevious.LastRowVisIdx + 1; rowVisIdx <= vpCurrent.LastRowVisIdx; rowVisIdx++ {
		rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
		for colVisIdx := vpCurrent.FirstColVisIdx; colVisIdx <= vpCurrent.LastColVisIdx; colVisIdx++ {
			colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
			pr.renderCell(regionContainers, rowModIdx, colModIdx, region)
		}
	}

	remainingRowStart = max(vpPrevious.FirstRowVisIdx, vpCurrent.FirstRowVisIdx)
	remainingRowEnd = min(vpPrevious.LastRowVisIdx, vpCurrent.LastRowVisIdx)

	for colVisIdx := vpCurrent.FirstColVisIdx; colVisIdx < vpPrevious.FirstColVisIdx; colVisIdx++ {
		colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
		for rowVisIdx := remainingRowStart; rowVisIdx <= remainingRowEnd; rowVisIdx++ {
			rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
			pr.renderCell(regionContainers, rowModIdx, colModIdx, region)
		}
	}

	for colVisIdx := vpPrevious.LastColVisIdx + 1; colVisIdx <= vpCurrent.LastColVisIdx; colVisIdx++ {
		colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
		for rowVisIdx := remainingRowStart; rowVisIdx <= remainingRowEnd; rowVisIdx++ {
			rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
			pr.renderCell(regionContainers, rowModIdx, colModIdx, region)
		}
	}

}
func (rr *RegionRenderer) renderCellsRegionDelta(regionContainers RegionContainers, region GridRegion) {

	pr := rr.primitiveRenderer
	cm := rr.ctx.CoordManager

	vpCurrent := rr.ctx.Viewports[region]
	vpPrevious := rr.ctx.LastViewports[region]

	rr.removeStandardCellsOutsideViewport(vpCurrent, vpPrevious, region)

	rr.addStandardCellsInViewport(regionContainers, vpCurrent, vpPrevious, region)

	pr.updateVisibleMerges(regionContainers, vpCurrent, region)

	// every thing but main region need manually placed
	if region != RegionMain {
		for cellid, idx := range pr.textPrimitivesIndex[region] {
			item := pr.textPrimitives[region][idx]
			item.Move(cm.GetPixelPos(region, cellid.Row, cellid.Col))
		}

		for cellid, idx := range pr.rectanglePrimitivesIndex[region] {
			item := pr.rectanglePrimitives[region][idx]
			item.Move(cm.GetPixelPos(region, cellid.Row, cellid.Col))
		}
	}

	pr.moveToCorner(region)
}

func (rr *RegionRenderer) renderCellsRegionFull(regionContainers RegionContainers, region GridRegion) {

	vpCurrent := rr.ctx.Viewports[region]

	pr := rr.primitiveRenderer
	cm := rr.ctx.CoordManager

	for rowVisIdx := vpCurrent.FirstRowVisIdx; rowVisIdx <= vpCurrent.LastRowVisIdx; rowVisIdx++ {
		rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
		for colVisIdx := vpCurrent.FirstColVisIdx; colVisIdx <= vpCurrent.LastColVisIdx; colVisIdx++ {
			colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
			pr.renderCell(regionContainers, rowModIdx, colModIdx, region)
		}
	}

	pr.updateVisibleMerges(regionContainers, vpCurrent, region)

	// every thing but main region need manually placed
	if region != RegionMain {
		for cellid, idx := range pr.textPrimitivesIndex[region] {
			item := pr.textPrimitives[region][idx]
			item.Move(cm.GetPixelPos(region, cellid.Row, cellid.Col))
		}

		for cellid, idx := range pr.rectanglePrimitivesIndex[region] {
			item := pr.rectanglePrimitives[region][idx]
			item.Move(cm.GetPixelPos(region, cellid.Row, cellid.Col))
		}
	}

	pr.moveToCorner(region)

}

func (rr *RegionRenderer) renderGridlinesRegionDelta(regionContainer RegionContainers, region GridRegion) {
	gl := rr.gridlineRenderer

	vpCurrent := rr.ctx.Viewports[region]
	vpPrevious := rr.ctx.LastViewports[region]

	gl.SetWaterMark(region, Horizontal)
	gl.SetWaterMark(region, Vertical)

	// remove top redundant rows
	for primaryVisIdx := vpPrevious.FirstRowVisIdx; primaryVisIdx <= vpCurrent.FirstRowVisIdx-1; primaryVisIdx++ {
		gl.Remove(primaryVisIdx, region, Horizontal)
	}
	// remove bottom redundant rows
	for primaryVisIdx := vpCurrent.LastRowVisIdx + 1; primaryVisIdx <= vpPrevious.LastRowVisIdx; primaryVisIdx++ {
		gl.Remove(primaryVisIdx, region, Horizontal)
	}

	remainingRowStart := max(vpPrevious.FirstRowVisIdx, vpCurrent.FirstRowVisIdx)
	remainingRowEnd := min(vpPrevious.LastRowVisIdx, vpCurrent.LastRowVisIdx)

	// remove left edge
	if vpCurrent.FirstColVisIdx > vpPrevious.FirstColVisIdx {
		for primaryVisIdx := remainingRowStart; primaryVisIdx <= remainingRowEnd; primaryVisIdx++ {
			gl.TrimStart(primaryVisIdx, vpCurrent.FirstColVisIdx, region, Horizontal)
		}
	}
	// remove right edge
	if vpCurrent.LastColVisIdx < vpPrevious.LastColVisIdx {
		for primaryVisIdx := remainingRowStart; primaryVisIdx <= remainingRowEnd; primaryVisIdx++ {
			gl.TrimEnd(primaryVisIdx, vpCurrent.LastColVisIdx, region, Horizontal)
		}
	}

	// add new top rows
	for rowVisIdx := vpCurrent.FirstRowVisIdx; rowVisIdx < vpPrevious.FirstRowVisIdx; rowVisIdx++ {
		gl.Add(rowVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx, region, Horizontal)
	}
	// add new bottom rows
	for rowVisIdx := vpCurrent.LastRowVisIdx; rowVisIdx > vpPrevious.LastRowVisIdx; rowVisIdx-- {
		gl.Add(rowVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx, region, Horizontal)
	}

	// extend items on the left
	if vpCurrent.FirstColVisIdx < vpPrevious.FirstColVisIdx {
		for primaryVisIdx := remainingRowStart; primaryVisIdx <= remainingRowEnd; primaryVisIdx++ {
			if _, exists := gl.LineItems[region][Horizontal].Edges[primaryVisIdx]; exists {
				gl.GrowStart(primaryVisIdx, vpCurrent.FirstColVisIdx, region, Horizontal)
			} else {
				gl.Add(primaryVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx, region, Horizontal)
			}
		}
	}
	// extend items on the right
	if vpCurrent.LastColVisIdx > vpPrevious.LastColVisIdx {
		for primaryVisIdx := remainingRowStart; primaryVisIdx <= remainingRowEnd; primaryVisIdx++ {
			if _, exists := gl.LineItems[region][Horizontal].Edges[primaryVisIdx]; exists {
				gl.GrowEnd(primaryVisIdx, vpCurrent.LastColVisIdx, region, Horizontal)
			} else {
				gl.Add(primaryVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx, region, Horizontal)
			}
		}
	}

	// remove left redundant cols (cols that have scrolled out of view on the left)
	for primaryVisIdx := vpPrevious.FirstColVisIdx; primaryVisIdx <= vpCurrent.FirstColVisIdx-1; primaryVisIdx++ {
		gl.Remove(primaryVisIdx, region, Vertical)
	}
	// remove right redundant cols (cols that have scrolled out of view on the right)
	for primaryVisIdx := vpCurrent.LastColVisIdx + 1; primaryVisIdx <= vpPrevious.LastColVisIdx; primaryVisIdx++ {
		gl.Remove(primaryVisIdx, region, Vertical)
	}

	remainingColStart := max(vpPrevious.FirstColVisIdx, vpCurrent.FirstColVisIdx)
	remainingColEnd := min(vpPrevious.LastColVisIdx, vpCurrent.LastColVisIdx)

	// remove top edge (vertical lines whose secondary/row-start is now above the viewport)
	if vpCurrent.FirstRowVisIdx > vpPrevious.FirstRowVisIdx {
		for primaryVisIdx := remainingColStart; primaryVisIdx <= remainingColEnd; primaryVisIdx++ {
			gl.TrimStart(primaryVisIdx, vpCurrent.FirstRowVisIdx, region, Vertical)
		}
	}
	// remove bottom edge (vertical lines whose secondary/row-end is now below the viewport)
	if vpCurrent.LastRowVisIdx < vpPrevious.LastRowVisIdx {
		for primaryVisIdx := remainingColStart; primaryVisIdx <= remainingColEnd; primaryVisIdx++ {
			gl.TrimEnd(primaryVisIdx, vpCurrent.LastRowVisIdx, region, Vertical)
		}
	}

	// add new left cols
	for colVisIdx := vpCurrent.FirstColVisIdx; colVisIdx < vpPrevious.FirstColVisIdx; colVisIdx++ {
		gl.Add(colVisIdx, vpCurrent.FirstRowVisIdx, vpCurrent.LastRowVisIdx, region, Vertical)
	}
	// add new right cols
	for colVisIdx := vpCurrent.LastColVisIdx; colVisIdx > vpPrevious.LastColVisIdx; colVisIdx-- {
		gl.Add(colVisIdx, vpCurrent.FirstRowVisIdx, vpCurrent.LastRowVisIdx, region, Vertical)
	}

	// extend items on the top (vertical lines that need to grow upward)
	if vpCurrent.FirstRowVisIdx < vpPrevious.FirstRowVisIdx {
		for primaryVisIdx := remainingColStart; primaryVisIdx <= remainingColEnd; primaryVisIdx++ {
			if _, exists := gl.LineItems[region][Vertical].Edges[primaryVisIdx]; exists {
				gl.GrowStart(primaryVisIdx, vpCurrent.FirstRowVisIdx, region, Vertical)
			} else {
				gl.Add(primaryVisIdx, vpCurrent.FirstRowVisIdx, vpCurrent.LastRowVisIdx, region, Vertical)
			}
		}
	}
	// extend items on the bottom (vertical lines that need to grow downward)
	if vpCurrent.LastRowVisIdx > vpPrevious.LastRowVisIdx {
		for primaryVisIdx := remainingColStart; primaryVisIdx <= remainingColEnd; primaryVisIdx++ {
			if _, exists := gl.LineItems[region][Vertical].Edges[primaryVisIdx]; exists {
				gl.GrowEnd(primaryVisIdx, vpCurrent.LastRowVisIdx, region, Vertical)
			} else {
				gl.Add(primaryVisIdx, vpCurrent.FirstRowVisIdx, vpCurrent.LastRowVisIdx, region, Vertical)
			}
		}
	}

	gl.fyneAddContent(regionContainer.Gridline, region, Horizontal)
	gl.fyneAddContent(regionContainer.Gridline, region, Vertical)

	gl.fyneMoveHotPoolItems(region, Horizontal)
	gl.fyneMoveHotPoolItems(region, Vertical)

	rr.frameCounters[region]++
	if rr.frameCounters[region] > 6 {
		gl.fynePositionGridlines(region, Horizontal, true)
		gl.fynePositionGridlines(region, Vertical, true)
		rr.frameCounters[region] = 0
	} else {
		gl.fynePositionGridlines(region, Horizontal, false)
		gl.fynePositionGridlines(region, Vertical, false)
	}
}

func (rr *RegionRenderer) renderGridlinesRegionFull(regionContainer RegionContainers,
	region GridRegion) {

	gl := rr.gridlineRenderer

	vpCurrent := rr.ctx.Viewports[region]

	gl.SetWaterMark(region, Horizontal)
	gl.SetWaterMark(region, Vertical)

	for rowVisIdx := vpCurrent.FirstRowVisIdx; rowVisIdx <= vpCurrent.LastRowVisIdx; rowVisIdx++ {
		gl.Add(rowVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx, region, Horizontal)
	}

	for colVisIdx := vpCurrent.FirstColVisIdx; colVisIdx <= vpCurrent.LastColVisIdx; colVisIdx++ {
		gl.Add(colVisIdx, vpCurrent.FirstRowVisIdx, vpCurrent.LastRowVisIdx, region, Vertical)
	}

	gl.fyneAddContent(regionContainer.Gridline, region, Horizontal)
	gl.fyneAddContent(regionContainer.Gridline, region, Vertical)

	gl.fynePositionGridlines(region, Horizontal, true)
	gl.fynePositionGridlines(region, Vertical, true)

}
