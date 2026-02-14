package pkg

import "fmt"

type RegionRenderer struct {
	ctx               *RenderContext
	fontManager       *FontManager
	primitiveRenderer *PrimitiveRenderer
	borderRenderer    *BorderLineRenderer
	gridlineRenderer  *PrimitiveGridLineRenderer
}

func NewRegionRenderer(ctx *RenderContext, fontMgr *FontManager) *RegionRenderer {
	return &RegionRenderer{
		ctx:               ctx,
		fontManager:       fontMgr,
		borderRenderer:    NewBorderLineRenderer(ctx),
		primitiveRenderer: NewPrimitiveRenderer(ctx, fontMgr),
		gridlineRenderer:  NewPrimitiveGridLineRenderer(ctx),
	}
}

func (rr *RegionRenderer) renderCellsRegionDelta(regionContainers RegionContainers, region GridRegion) {

	var remainingRowStart, remainingRowEnd int
	vpCurrent := rr.ctx.Viewports[region]
	vpPrevious := rr.ctx.LastViewports[region]
	if 1 == 0 {
		if region == RegionMain {
			fmt.Printf("[VP-CURRENT-MAIN] FirstRow:%d, LastRow:%d, FirstCol:%d, LastCol:%d\n",
				vpCurrent.FirstRowVisIdx, vpCurrent.LastRowVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx)
			fmt.Printf("[VP-PREVIOUS-MAIN] FirstRow:%d, LastRow:%d, FirstCol:%d, LastCol:%d\n",
				vpPrevious.FirstRowVisIdx, vpPrevious.LastRowVisIdx, vpPrevious.FirstColVisIdx, vpPrevious.LastColVisIdx)
		}
		if region == RegionFixedCorner {

			fmt.Printf("[VP-CURRENT-FIXED-CORNER] FirstRow:%d, LastRow:%d, FirstCol:%d, LastCol:%d\n",
				vpCurrent.FirstRowVisIdx, vpCurrent.LastRowVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx)
			fmt.Printf("[VP-PREVIOUS-FIXED-CORNER] FirstRow:%d, LastRow:%d, FirstCol:%d, LastCol:%d\n",
				vpPrevious.FirstRowVisIdx, vpPrevious.LastRowVisIdx, vpPrevious.FirstColVisIdx, vpPrevious.LastColVisIdx)
		}
		if region == RegionFrozenCols {
			fmt.Printf("[VP-CURRENT-FIXED-COLS] FirstRow:%d, LastRow:%d, FirstCol:%d, LastCol:%d\n",
				vpCurrent.FirstRowVisIdx, vpCurrent.LastRowVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx)
			fmt.Printf("[VP-PREVIOUS-FIXED-COLS] FirstRow:%d, LastRow:%d, FirstCol:%d, LastCol:%d\n",
				vpPrevious.FirstRowVisIdx, vpPrevious.LastRowVisIdx, vpPrevious.FirstColVisIdx, vpPrevious.LastColVisIdx)
		}
		if region == RegionFrozenRows {

			fmt.Printf("[VP-CURRENT-FIXED-ROWS] FirstRow:%d, LastRow:%d, FirstCol:%d, LastCol:%d\n",
				vpCurrent.FirstRowVisIdx, vpCurrent.LastRowVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx)
			fmt.Printf("[VP-PREVIOUS-FIXED-ROWS] FirstRow:%d, LastRow:%d, FirstCol:%d, LastCol:%d\n",
				vpPrevious.FirstRowVisIdx, vpPrevious.LastRowVisIdx, vpPrevious.FirstColVisIdx, vpPrevious.LastColVisIdx)
		}
	}
	pr := rr.primitiveRenderer
	cm := rr.ctx.CoordManager

	//if region == RegionMain {
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

	pr.removeMergesOutsideViewport(vpCurrent, region)

	for rowVisIdx := vpCurrent.FirstRowVisIdx; rowVisIdx < vpPrevious.FirstRowVisIdx; rowVisIdx++ {
		rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
		for colVisIdx := vpCurrent.FirstColVisIdx; colVisIdx <= vpCurrent.LastColVisIdx; colVisIdx++ {
			colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
			pr.addCell(regionContainers.Background, regionContainers.Data, rowModIdx, colModIdx, region)
		}
	}

	for rowVisIdx := vpPrevious.LastRowVisIdx + 1; rowVisIdx <= vpCurrent.LastRowVisIdx; rowVisIdx++ {
		rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
		for colVisIdx := vpCurrent.FirstColVisIdx; colVisIdx <= vpCurrent.LastColVisIdx; colVisIdx++ {
			colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
			pr.addCell(regionContainers.Background, regionContainers.Data, rowModIdx, colModIdx, region)
		}
	}

	remainingRowStart = max(vpPrevious.FirstRowVisIdx, vpCurrent.FirstRowVisIdx)
	remainingRowEnd = min(vpPrevious.LastRowVisIdx, vpCurrent.LastRowVisIdx)

	for colVisIdx := vpCurrent.FirstColVisIdx; colVisIdx < vpPrevious.FirstColVisIdx; colVisIdx++ {
		colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
		for rowVisIdx := remainingRowStart; rowVisIdx <= remainingRowEnd; rowVisIdx++ {
			rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
			pr.addCell(regionContainers.Background, regionContainers.Data, rowModIdx, colModIdx, region)
		}
	}

	for colVisIdx := vpPrevious.LastColVisIdx + 1; colVisIdx <= vpCurrent.LastColVisIdx; colVisIdx++ {
		colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
		for rowVisIdx := remainingRowStart; rowVisIdx <= remainingRowEnd; rowVisIdx++ {
			rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
			pr.addCell(regionContainers.Background, regionContainers.Data, rowModIdx, colModIdx, region)
		}
	}

	pr.renderVisibleMerges(regionContainers.Background, regionContainers.Data, vpCurrent, region)

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
	glr := rr.gridlineRenderer
	pr := rr.primitiveRenderer
	cm := rr.ctx.CoordManager

	var remainingRowStart, remainingRowEnd int
	vpCurrent := rr.ctx.Viewports[region]
	vpPrevious := rr.ctx.LastViewports[region]

	cache := pr.BackgroundTransparencyStates(region)
	if vpCurrent.FirstRowVisIdx > vpPrevious.FirstRowVisIdx {
		glr.removeRowsTop(vpCurrent, vpPrevious, region)
	}

	if vpCurrent.LastRowVisIdx < vpPrevious.LastRowVisIdx {
		glr.removeRowsBottom(vpCurrent, vpPrevious, region)
	}

	if vpCurrent.FirstColVisIdx > vpPrevious.FirstColVisIdx {
		glr.cleanupLeftEdge(vpCurrent, vpPrevious, region)
	}

	if vpCurrent.LastColVisIdx < vpPrevious.LastColVisIdx {
		glr.cleanupRightEdge(vpCurrent, vpPrevious, region)
	}

	if vpCurrent.FirstRowVisIdx < vpPrevious.FirstRowVisIdx {
		glr.renderLeftEdge(regionContainer.Gridline, vpCurrent.FirstRowVisIdx, vpPrevious.FirstRowVisIdx-1, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx, cache, region)
	}

	if vpCurrent.LastRowVisIdx+1 > vpPrevious.LastRowVisIdx && vpCurrent.FirstColVisIdx <= vpCurrent.LastColVisIdx {
		glr.renderLeftEdge(regionContainer.Gridline, vpPrevious.LastRowVisIdx+1, vpCurrent.LastRowVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx, cache, region)
	}

	remainingRowStart = max(vpPrevious.FirstRowVisIdx, vpCurrent.FirstRowVisIdx)
	remainingRowEnd = min(vpPrevious.LastRowVisIdx, vpCurrent.LastRowVisIdx)

	if vpCurrent.FirstColVisIdx < vpPrevious.FirstColVisIdx {
		glr.renderLeftEdge(regionContainer.Gridline, remainingRowStart, remainingRowEnd, vpCurrent.FirstColVisIdx, vpPrevious.FirstColVisIdx-1, cache, region)
	}
	if vpCurrent.LastColVisIdx > vpPrevious.LastColVisIdx {
		glr.renderRightEdge(regionContainer.Gridline, remainingRowStart, remainingRowEnd, vpPrevious.LastColVisIdx+1, vpCurrent.LastColVisIdx, cache, region)
	}

	glr.stashInCorner(region)
	if 1 == 2 {
		rowId := 6

		found := true
		fmt.Printf("[ITEM-LIST-STARTING]\n")
		for found {
			found = false
			for itemId, configItem := range glr.hLineItems[region].ConfigLines {
				if configItem.Row == rowId {
					fmt.Printf("[ITEM-LIST] Row:%d, ColStart:%d, ColEnd:%d, ItemCount:%d, ItemId:%d, RecycleBinItems:%d\n",
						configItem.Row, configItem.ColStart, configItem.ColEnd, len(glr.hLineItems[region].ConfigLines), itemId+1, glr.RecycleBinItems(region))
					found = true
				}
			}
			rowId++
		}
		fmt.Printf("[ITEM-LIST-ENDING]\n")
	}

	if region != RegionMain {
		for itemId, config := range glr.hLineItems[region].ConfigLines {
			lineItem := glr.hLineItems[region].Lines[itemId]
			if config.Row != -1 {
				y := cm.GetRowPixelPosEndY(region, cm.GetRowModIdxFromVisIdx(config.Row))

				lineItem.Position1.Y = y

				lineItem.Position1.X = cm.GetColPixelPosX(region, cm.GetColModIdxFromVisIdx(config.ColStart))

				lineItem.Position2.Y = y

				lineItem.Position2.X = cm.GetColPixelPosEndX(region, cm.GetColModIdxFromVisIdx(config.ColEnd))
			}
		}
	}
}

func (rr *RegionRenderer) renderCellsRegionFull(regionContainers RegionContainers, region GridRegion) {

	vp := rr.ctx.Viewports[region]
	if 1 == 2 {
		if region == RegionMain {
			fmt.Printf("[VP-CURRENT-MAIN] FirstRow:%d, LastRow:%d, FirstCol:%d, LastCol:%d\n",
				vp.FirstRowVisIdx, vp.LastRowVisIdx, vp.FirstColVisIdx, vp.LastColVisIdx)
		}
		if region == RegionFixedCorner {

			fmt.Printf("[VP-CURRENT-FIXED-CORNER] FirstRow:%d, LastRow:%d, FirstCol:%d, LastCol:%d\n",
				vp.FirstRowVisIdx, vp.LastRowVisIdx, vp.FirstColVisIdx, vp.LastColVisIdx)
		}
		if region == RegionFrozenCols {
			fmt.Printf("[VP-CURRENT-FIXED-COLS] FirstRow:%d, LastRow:%d, FirstCol:%d, LastCol:%d\n",
				vp.FirstRowVisIdx, vp.LastRowVisIdx, vp.FirstColVisIdx, vp.LastColVisIdx)
		}
		if region == RegionFrozenRows {
			fmt.Printf("[VP-CURRENT-FIXED-ROWS] FirstRow:%d, LastRow:%d, FirstCol:%d, LastCol:%d\n",
				vp.FirstRowVisIdx, vp.LastRowVisIdx, vp.FirstColVisIdx, vp.LastColVisIdx)
		}
	}

	pr := rr.primitiveRenderer
	cm := rr.ctx.CoordManager

	for rowVisIdx := vp.FirstRowVisIdx; rowVisIdx <= vp.LastRowVisIdx; rowVisIdx++ {
		rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
		for colVisIdx := vp.FirstColVisIdx; colVisIdx <= vp.LastColVisIdx; colVisIdx++ {
			colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
			pr.addCell(regionContainers.Background, regionContainers.Data, rowModIdx, colModIdx, region)
		}
	}

	pr.renderVisibleMerges(regionContainers.Background, regionContainers.Data, vp, region)

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

func (rr *RegionRenderer) renderGridlinesRegionFull(regionContainer RegionContainers, region GridRegion) {
	glr := rr.gridlineRenderer
	pr := rr.primitiveRenderer

	//	var remainingRowStart, remainingRowEnd int
	vpCurrent := rr.ctx.Viewports[region]
	//	vpPrevious := rr.ctx.LastViewports[region]

	cache := pr.BackgroundTransparencyStates(region)

	//if region == RegionMain {
	glr.renderLeftEdge(regionContainer.Gridline, vpCurrent.FirstRowVisIdx, vpCurrent.LastRowVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx, cache, region)

	if region == RegionFixedCorner {
		fmt.Printf("[ITEM-LIST-CORNER] ElementNo:%d\n", glr.hLineItems[region].ConfigLines)
	}
	//}
	glr.stashInCorner(region)
	if 1 == 2 {
		rowId := 6

		found := true
		fmt.Printf("[ITEM-LIST-STARTING]\n")
		for found {
			found = false
			for itemId, configItem := range glr.hLineItems[region].ConfigLines {
				if configItem.Row == rowId {
					fmt.Printf("[ITEM-LIST] Row:%d, ColStart:%d, ColEnd:%d, ItemCount:%d, ItemId:%d, RecycleBinItems:%d\n",
						configItem.Row, configItem.ColStart, configItem.ColEnd, len(glr.hLineItems[region].ConfigLines), itemId+1, glr.RecycleBinItems(region))
					found = true
				}
			}
			rowId++
		}
		fmt.Printf("[ITEM-LIST-ENDING]\n")
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

	// Decision: FULL or DELTA render?
	if !ctx.PaneHasRenderedOnce[region] {
		// This region hasn't had initial render yet
		if r.isValidViewport(region) {
			// Viewport is ready → Do FULL render
			rr.renderCellsRegionFull(containers, region)

			rr.renderGridlinesRegionFull(containers, region)

			ctx.PaneHasRenderedOnce[region] = true
		}
	} else {
		// Region already had initial render → Use DELTA render from now on
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
			//if region == RegionMain {
			rr.renderGridlinesRegionDelta(containers, region)
			//}

		}
	}
}
