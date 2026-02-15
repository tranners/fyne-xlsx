package pkg

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

func (pglr *PrimitiveGridLineRenderer) stashInCornerV(gridRegion GridRegion) {
	// if we have items in the flagged pool; put to recycle pool, for use later
	for _, itemId := range pglr.vFlaggedItems[gridRegion].items {

		obj := pglr.vLineItems[gridRegion].Lines[itemId]
		objConfig := pglr.vLineItems[gridRegion].ConfigLines[itemId]
		obj.Move(fyne.NewPos(-9999, -9999))

		pglr.vItemRecycler[gridRegion].Put(itemId)

		delete(pglr.vLineIndex[gridRegion][objConfig.Col].P1, objConfig.RowStart)
		delete(pglr.vLineIndex[gridRegion][objConfig.Col].P2, objConfig.RowEnd)

		pglr.vLineItems[gridRegion].ConfigLines[itemId] = vLineConfig{
			Col:          -1, // Invalid row
			RowStart:     -1,
			RowEnd:       -1,
			UnPositioned: true,
		}
	}
	pglr.vFlaggedItems[gridRegion].Reset()
}

func (pglr *PrimitiveGridLineRenderer) isGridLineRequiredV(isTransparent TransparencyCache, rowVisIdx, colVisIdx int) bool {
	mm := pglr.ctx.MergeManager

	id := CellID{Row: rowVisIdx, Col: colVisIdx}
	if info, exists := mm.visIdxMergeCache[id]; exists {
		if colVisIdx == info.VisColEnd {
			anchorVisIdx := CellID{info.VisRowStart, info.VisColStart}
			if hasBackground, _ := mm.anchorHasBackgroundCache[anchorVisIdx]; hasBackground {
				return false
			}
		} else {
			// in the merged range somewhere
			return false
		}
	} else {
		if needed, ok := isTransparent[id]; ok {
			if !needed {
				return false
			}
		}
	}

	// check the other cell, bordering the gridline
	colVisIdx++
	id = CellID{Row: rowVisIdx, Col: colVisIdx}
	if info, exists := mm.visIdxMergeCache[id]; exists {
		if colVisIdx == info.VisColStart {
			anchorVisIdx := CellID{info.VisRowStart, info.VisColStart}
			if hasBackground, _ := mm.anchorHasBackgroundCache[anchorVisIdx]; hasBackground {
				return false
			}
		} else {
			// in the merged range somewhere
			return false
		}
	} else {
		if needed, ok := isTransparent[id]; ok {
			if !needed {
				return false
			}
		}
	}
	return true
}

func (pglr *PrimitiveGridLineRenderer) RecycleBinItemsV(gridRegion GridRegion) int {
	itemRecycler := pglr.vItemRecycler[gridRegion]
	return itemRecycler.Size()
}

func (pglr *PrimitiveGridLineRenderer) removeColsLeftV(vpCurrent, vpPrevious Viewport, gridRegion GridRegion) {
	for colVisIdx := vpPrevious.FirstColVisIdx; colVisIdx < vpCurrent.FirstColVisIdx; colVisIdx++ {
		lineIdx := pglr.vLineIndex[gridRegion][colVisIdx]
		for _, itemId := range lineIdx.P1 {
			pglr.vFlaggedItems[gridRegion].Put(itemId)
		}
		delete(pglr.vLineIndex[gridRegion], colVisIdx)
	}
}

func (pglr *PrimitiveGridLineRenderer) removeColsRightV(vpCurrent, vpPrevious Viewport, gridRegion GridRegion) {
	for colVisIdx := vpCurrent.LastColVisIdx + 1; colVisIdx <= vpPrevious.LastRowVisIdx; colVisIdx++ {
		lineIdx := pglr.vLineIndex[gridRegion][colVisIdx]
		for _, itemId := range lineIdx.P1 {
			pglr.vFlaggedItems[gridRegion].Put(itemId)
		}
		delete(pglr.vLineIndex[gridRegion], colVisIdx)
	}
}

func (pglr *PrimitiveGridLineRenderer) cleanupTopEdgeV(vpCurrent, vpPrevious Viewport, gridRegion GridRegion) {
	for colVisIdx := vpCurrent.FirstColVisIdx; colVisIdx <= vpCurrent.LastColVisIdx; colVisIdx++ {
		lineIdx := pglr.vLineIndex[gridRegion][colVisIdx]

		for rowStart := vpPrevious.FirstRowVisIdx; rowStart < vpCurrent.FirstRowVisIdx; rowStart++ {
			if itemId, exists := lineIdx.P1[rowStart]; exists {
				config := pglr.vLineItems[gridRegion].ConfigLines[itemId]
				if config.RowEnd < vpCurrent.FirstRowVisIdx {
					pglr.vFlaggedItems[gridRegion].Put(itemId)
				} else {
					pglr.updatePosition1LineV(itemId, colVisIdx, vpCurrent.FirstRowVisIdx, gridRegion)
				}
			}
		}
	}
}

func (pglr *PrimitiveGridLineRenderer) cleanupBottomEdgeH(vpCurrent, vpPrevious Viewport, gridRegion GridRegion) {
	for colVisIdx := vpCurrent.FirstColVisIdx; colVisIdx <= vpCurrent.LastColVisIdx; colVisIdx++ {
		lineIdx := pglr.vLineIndex[gridRegion][colVisIdx]

		for rowEnd := vpPrevious.LastRowVisIdx; rowEnd > vpCurrent.LastColVisIdx; rowEnd-- {
			if itemId, exists := lineIdx.P2[rowEnd]; exists {
				config := pglr.vLineItems[gridRegion].ConfigLines[itemId]
				if config.RowStart > vpCurrent.LastRowVisIdx {
					pglr.vFlaggedItems[gridRegion].Put(itemId)
				} else {
					pglr.updatePosition2LineV(itemId, colVisIdx, vpCurrent.LastRowVisIdx, gridRegion)
				}
			}
		}
	}
}

func (pglr *PrimitiveGridLineRenderer) addNewLineV(container *fyne.Container, colVisIdx, rowStartVisIdx, rowEndVisIdx int, gridRegion GridRegion) {
	var lineItem *canvas.Line

	if flaggedItemId, exist := pglr.vFlaggedItems[gridRegion].Get(); exist {

		origItem := pglr.vLineItems[gridRegion].ConfigLines[flaggedItemId]

		newItem := vLineConfig{
			Col:          colVisIdx,
			RowStart:     rowStartVisIdx,
			RowEnd:       rowEndVisIdx,
			UnPositioned: true,
		}

		pglr.vLineItems[gridRegion].ConfigLines[flaggedItemId] = newItem

		lineItem = pglr.vLineItems[gridRegion].Lines[flaggedItemId]

		delete(pglr.vLineIndex[gridRegion][origItem.Col].P1, origItem.RowStart)

		delete(pglr.hLineIndex[gridRegion][origItem.Col].P2, origItem.RowEnd)

		pglr.hLineIndex[gridRegion][colVisIdx].P1[rowStartVisIdx] = flaggedItemId
		pglr.hLineIndex[gridRegion][colVisIdx].P2[rowEndVisIdx] = flaggedItemId

	} else {
		// No flagged item available; so goto the pool
		primitiveGridLineRecyleItem, recycledItem := pglr.vItemRecycler[gridRegion].Get()

		newConfigItem := vLineConfig{
			Col:          colVisIdx,
			RowStart:     rowStartVisIdx,
			RowEnd:       rowEndVisIdx,
			UnPositioned: true,
		}
		if recycledItem {
			lineItem = pglr.vLineItems[gridRegion].Lines[primitiveGridLineRecyleItem.id]

			pglr.vLineItems[gridRegion].ConfigLines[primitiveGridLineRecyleItem.id] = newConfigItem

			pglr.vLineIndex[gridRegion][colVisIdx].P1[rowStartVisIdx] = primitiveGridLineRecyleItem.id
			pglr.hLineIndex[gridRegion][colVisIdx].P2[rowEndVisIdx] = primitiveGridLineRecyleItem.id
		} else {
			lineItem = primitiveGridLineRecyleItem.obj

			item := pglr.vLineItems[gridRegion]
			item.ConfigLines = append(item.ConfigLines, newConfigItem)
			item.Lines = append(item.Lines, primitiveGridLineRecyleItem.obj)

			pglr.vLineItems[gridRegion] = item

			itemId := len(item.Lines) - 1

			pglr.vLineIndex[gridRegion][colVisIdx].P1[rowStartVisIdx] = itemId
			pglr.vLineIndex[gridRegion][colVisIdx].P2[rowEndVisIdx] = itemId

			// the only plavce where we add content
			container.Add(lineItem)
		}
	}
}

func (pglr *PrimitiveGridLineRenderer) updatePosition1LineV(itemId, colVisIdx, startVisRowIdx int, gridRegion GridRegion) {

	lineItem := pglr.vLineItems[gridRegion].ConfigLines[itemId]

	originalRowStart := lineItem.RowStart

	lineItem.RowStart = startVisRowIdx
	lineItem.UnPositioned = true

	pglr.vLineItems[gridRegion].ConfigLines[itemId] = lineItem
	pglr.vLineIndex[gridRegion][colVisIdx].P1[startVisRowIdx] = itemId

	delete(pglr.vLineIndex[gridRegion][colVisIdx].P1, originalRowStart)

}

func (pglr *PrimitiveGridLineRenderer) updatePosition2LineV(itemId, colVisIdx, endVisRowIdx int, gridRegion GridRegion) {

	lineItem := pglr.vLineItems[gridRegion].ConfigLines[itemId]

	originalRowEnd := lineItem.RowEnd

	lineItem.RowEnd = endVisRowIdx
	lineItem.UnPositioned = true

	pglr.vLineItems[gridRegion].ConfigLines[itemId] = lineItem
	pglr.vLineIndex[gridRegion][colVisIdx].P2[endVisRowIdx] = itemId

	delete(pglr.vLineIndex[gridRegion][colVisIdx].P2, originalRowEnd)

}

func (pglr *PrimitiveGridLineRenderer) renderTopEdgeV(container *fyne.Container,
	colStart, colEnd int,
	rowStart, rowEnd int,
	cache TransparencyCache,
	gridRegion GridRegion) {
	var mode mode
	var currentVisIdx int
	var startVisRowIdx int
	var endVisRowIdx int

	for colVisIdx := colStart; colVisIdx <= colEnd; colVisIdx++ {

		if _, exist := pglr.vLineIndex[gridRegion][colVisIdx]; !exist {
			pglr.vLineIndex[gridRegion][colVisIdx] = lineIndex{
				P1: make(map[int]int),
				P2: make(map[int]int),
			}
		}

		startVisRowIdx = -1
		endVisRowIdx = -1
		currentVisIdx = rowStart
		mode = MODE_SEARCHING

		for {
			if pglr.isGridLineRequiredV(cache, colVisIdx, currentVisIdx) {
				if mode == MODE_SEARCHING {
					startVisRowIdx = currentVisIdx
					mode = MODE_STARTING
				}
			} else if mode == MODE_STARTING {
				endVisRowIdx = currentVisIdx - 1
				pglr.addNewLineV(container, colVisIdx, startVisRowIdx, endVisRowIdx, gridRegion)

				mode = MODE_SEARCHING
			}
			if currentVisIdx == rowEnd {
				if mode == MODE_STARTING {
					endVisRowIdx = currentVisIdx
					if itemId, exist := pglr.vLineIndex[gridRegion][colVisIdx].P1[endVisRowIdx+1]; exist {
						pglr.updatePosition1LineV(itemId, colVisIdx, startVisRowIdx, gridRegion)
					} else {
						pglr.addNewLineV(container, colVisIdx, startVisRowIdx, endVisRowIdx, gridRegion)
					}
				}
				break
			}
			currentVisIdx++
		}
	}
}

func (pglr *PrimitiveGridLineRenderer) renderBottomEdgeV(container *fyne.Container,
	colStart, colEnd int,
	rowStart, rowEnd int,
	cache TransparencyCache,
	gridRegion GridRegion) {
	var mode mode
	var currentVisIdx int
	var startVisRowIdx int
	var endVisRowIdx int

	for colVisIdx := colStart; colVisIdx <= colEnd; colVisIdx++ {
		if _, exist := pglr.vLineIndex[gridRegion][colVisIdx]; !exist {
			pglr.vLineIndex[gridRegion][colVisIdx] = lineIndex{
				P1: make(map[int]int),
				P2: make(map[int]int),
			}
		}

		startVisRowIdx = -1
		endVisRowIdx = -1
		currentVisIdx = rowEnd
		mode = MODE_SEARCHING

		for {
			if pglr.isGridLineRequiredV(cache, colVisIdx, currentVisIdx) {
				if mode == MODE_SEARCHING {
					endVisRowIdx = currentVisIdx
					mode = MODE_STARTING
				}
			} else if mode == MODE_STARTING {
				startVisRowIdx = currentVisIdx + 1

				pglr.addNewLineV(container, colVisIdx, startVisRowIdx, endVisRowIdx, gridRegion)

				mode = MODE_SEARCHING
			}
			if currentVisIdx == colStart {
				if mode == MODE_STARTING {
					startVisRowIdx = currentVisIdx
					if itemId, exist := pglr.vLineIndex[gridRegion][colVisIdx].P2[startVisRowIdx-1]; exist {
						pglr.updatePosition2LineV(itemId, colVisIdx, endVisRowIdx, gridRegion)

					} else {
						pglr.addNewLineV(container, colVisIdx, startVisRowIdx, endVisRowIdx, gridRegion)
					}
				}
				break
			}
			currentVisIdx--
		}
	}
}
