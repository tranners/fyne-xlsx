package pkg

import (
	"fmt"
)

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

			rr.renderGridlinesRegionFull(containers, region, Horizontal)

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

			rr.renderGridlinesRegionDelta(containers, region, Horizontal)
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

func (rr *RegionRenderer) renderGridlinesRegionDelta(regionContainer RegionContainers, region GridRegion, orientation LineOrientation) {

	var remainingRowStart, remainingRowEnd int

	gl := rr.gridlineRenderer

	vpCurrent := rr.ctx.Viewports[region]
	vpPrevious := rr.ctx.LastViewports[region]

	// remove top redundant rows
	for primaryVisIdx := vpPrevious.FirstRowVisIdx; primaryVisIdx <= vpCurrent.FirstRowVisIdx-1; primaryVisIdx++ {
		//if primaryVisIdx != 21 {
		//	continue
		//}
		gl.Remove(primaryVisIdx, region, orientation)
	}

	for primaryVisIdx := vpCurrent.LastRowVisIdx + 1; primaryVisIdx <= vpPrevious.LastRowVisIdx; primaryVisIdx++ {
		//if primaryVisIdx != 21 {
		//	continue
		//}
		gl.Remove(primaryVisIdx, region, orientation)
	}

	remainingRowStart = max(vpPrevious.FirstRowVisIdx, vpCurrent.FirstRowVisIdx)
	remainingRowEnd = min(vpPrevious.LastRowVisIdx, vpCurrent.LastRowVisIdx)
	// remove left edge
	if vpCurrent.FirstColVisIdx > vpPrevious.FirstColVisIdx {
		for primaryVisIdx := remainingRowStart; primaryVisIdx <= remainingRowEnd; primaryVisIdx++ {
			//if primaryVisIdx != 21 {
			//	continue
			//}
			gl.TrimStart(primaryVisIdx, vpCurrent.FirstColVisIdx, region, orientation)
		}
	}
	// remove right edge
	if vpCurrent.LastColVisIdx < vpPrevious.LastColVisIdx {
		for primaryVisIdx := remainingRowStart; primaryVisIdx <= remainingRowEnd; primaryVisIdx++ {
			//if primaryVisIdx != 21 {
			//	continue
			//}
			gl.TrimEnd(primaryVisIdx, vpCurrent.LastColVisIdx, region, orientation)
		}
	}

	// add new top rows
	for rowVisIdx := vpCurrent.FirstRowVisIdx; rowVisIdx < vpPrevious.FirstRowVisIdx; rowVisIdx++ {
		//if rowVisIdx != 21 {
		//	continue
		//}
		gl.Add(regionContainer.Gridline, rowVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx, region, orientation)
	}

	// add new bottom rows
	for rowVisIdx := vpCurrent.LastRowVisIdx; rowVisIdx > vpPrevious.LastRowVisIdx; rowVisIdx-- {
		//if rowVisIdx != 21 {
		//	continue
		//}
		gl.Add(regionContainer.Gridline, rowVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx, region, orientation)
	}

	// extend items on the left
	if vpCurrent.FirstColVisIdx < vpPrevious.FirstColVisIdx {
		for primaryVisIdx := remainingRowStart; primaryVisIdx <= remainingRowEnd; primaryVisIdx++ {
			//if primaryVisIdx != 21 {
			//	continue
			//}
			if _, exists := gl.LineItems[region][orientation].Edges[primaryVisIdx]; exists {
				gl.GrowStart(regionContainer.Gridline, primaryVisIdx, vpCurrent.FirstColVisIdx, region, orientation)
			} else {
				gl.Add(regionContainer.Gridline, primaryVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx, region, orientation)
			}
		}
	}

	// extend items on the right
	if vpCurrent.LastColVisIdx > vpPrevious.LastColVisIdx {
		for primaryVisIdx := remainingRowStart; primaryVisIdx <= remainingRowEnd; primaryVisIdx++ {
			//if primaryVisIdx != 21 {
			//	continue
			//}
			if _, exists := gl.LineItems[region][orientation].Edges[primaryVisIdx]; exists {
				gl.GrowEnd(regionContainer.Gridline, primaryVisIdx, vpCurrent.LastColVisIdx, region, orientation)
			} else {
				gl.Add(regionContainer.Gridline, primaryVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx, region, orientation)
			}
		}
	}

	//glr.stashFlaggedItems(region, orientation)
	gl.stashFlaggedItems(region, orientation)
	if region == RegionMain {
		if 1 == 1 {
			for i, configItem := range gl.LineItems[region][orientation].LinesConfig {
				fmt.Printf("[ITEM-LIST] Row:%d, ColStart:%d, ColEnd:%d, ItemCount:%d, ItemId:%d, PrevId:%d, NextId:%d\n",
					configItem.PrimaryAxis, configItem.SecondaryStart, configItem.SecondaryEnd, len(gl.LineItems[region][orientation].LinesConfig), i, configItem.PrevLineId, configItem.NextLineId)
			}

			for i, configItem := range gl.LineItems[region][orientation].Edges {
				fmt.Printf("[EDGE-LIST] EdgeRow:%d, LeftMost:%d, RightMost:%d\n", i, configItem.Startmost, configItem.Endmost)
			}
		}
	}

	rr.frameCounters[region]++
	if rr.frameCounters[region] > 6 {
		gl.positionGridlines(region, orientation, true)
		rr.frameCounters[region] = 0
	} else {
		gl.positionGridlines(region, orientation, false)

	}

	gl.fyneMoveStashedItems(region, orientation)
}

func (rr *RegionRenderer) renderGridlinesRegionFull(regionContainer RegionContainers,
	region GridRegion,
	orientation LineOrientation) {

	gl := rr.gridlineRenderer

	vpCurrent := rr.ctx.Viewports[region]

	for rowVisIdx := vpCurrent.FirstRowVisIdx; rowVisIdx <= vpCurrent.LastRowVisIdx; rowVisIdx++ {
		//if rowVisIdx != 21 {
		//	continue
		//}
		gl.Add(regionContainer.Gridline, rowVisIdx, vpCurrent.FirstColVisIdx, vpCurrent.LastColVisIdx, region, orientation)
	}

	gl.positionGridlines(region, orientation, true)
}
