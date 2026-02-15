package pkg

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

func (pglr *PrimitiveGridLineRenderer) stashInCornerH(gridRegion GridRegion) {
	// if we have items in the flagged pool; put to recycle pool, for use later
	for _, itemId := range pglr.hFlaggedItems[gridRegion].items {

		obj := pglr.hLineItems[gridRegion].Lines[itemId]
		objConfig := pglr.hLineItems[gridRegion].ConfigLines[itemId]
		obj.Move(fyne.NewPos(-9999, -9999))

		pglr.hItemRecycler[gridRegion].Put(itemId)

		delete(pglr.hLineIndex[gridRegion][objConfig.Row].P1, objConfig.ColStart)
		delete(pglr.hLineIndex[gridRegion][objConfig.Row].P2, objConfig.ColEnd)

		pglr.hLineItems[gridRegion].ConfigLines[itemId] = hLineConfig{
			Row:          -1, // Invalid row
			ColStart:     -1,
			ColEnd:       -1,
			UnPositioned: true,
		}
	}
	pglr.hFlaggedItems[gridRegion].Reset()
}

func (pglr *PrimitiveGridLineRenderer) isGridLineRequiredH(isTransparent TransparencyCache, rowVisIdx, colVisIdx int) bool {
	mm := pglr.ctx.MergeManager

	id := CellID{Row: rowVisIdx, Col: colVisIdx}
	if info, exists := mm.visIdxMergeCache[id]; exists {
		if rowVisIdx == info.VisRowEnd {
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
	rowVisIdx++
	id = CellID{Row: rowVisIdx, Col: colVisIdx}
	if info, exists := mm.visIdxMergeCache[id]; exists {
		if rowVisIdx == info.VisRowStart {
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

func (pglr *PrimitiveGridLineRenderer) RecycleBinItemsH(gridRegion GridRegion) int {
	itemRecycler := pglr.hItemRecycler[gridRegion]
	return itemRecycler.Size()
}

func (pglr *PrimitiveGridLineRenderer) removeRowsTopH(vpCurrent, vpPrevious Viewport, gridRegion GridRegion) {
	for rowVisIdx := vpPrevious.FirstRowVisIdx; rowVisIdx < vpCurrent.FirstRowVisIdx; rowVisIdx++ {
		lineIdx := pglr.hLineIndex[gridRegion][rowVisIdx]
		for _, itemId := range lineIdx.P1 {
			pglr.hFlaggedItems[gridRegion].Put(itemId)
		}
		delete(pglr.hLineIndex[gridRegion], rowVisIdx)
	}
}

func (pglr *PrimitiveGridLineRenderer) removeRowsBottomH(vpCurrent, vpPrevious Viewport, gridRegion GridRegion) {
	for rowVisIdx := vpCurrent.LastRowVisIdx + 1; rowVisIdx <= vpPrevious.LastRowVisIdx; rowVisIdx++ {
		lineIdx := pglr.hLineIndex[gridRegion][rowVisIdx]
		for _, itemId := range lineIdx.P1 {
			pglr.hFlaggedItems[gridRegion].Put(itemId)
		}
		delete(pglr.hLineIndex[gridRegion], rowVisIdx)
	}
}

func (pglr *PrimitiveGridLineRenderer) cleanupLeftEdgeH(vpCurrent, vpPrevious Viewport, gridRegion GridRegion) {
	for rowVisIdx := vpCurrent.FirstRowVisIdx; rowVisIdx <= vpCurrent.LastRowVisIdx; rowVisIdx++ {
		lineIdx := pglr.hLineIndex[gridRegion][rowVisIdx]

		for colStart := vpPrevious.FirstColVisIdx; colStart < vpCurrent.FirstColVisIdx; colStart++ {
			if itemId, exists := lineIdx.P1[colStart]; exists {
				config := pglr.hLineItems[gridRegion].ConfigLines[itemId]
				if config.ColEnd < vpCurrent.FirstColVisIdx {
					pglr.hFlaggedItems[gridRegion].Put(itemId)
				} else {
					pglr.updatePosition1LineH(itemId, rowVisIdx, vpCurrent.FirstColVisIdx, gridRegion)
				}
			}
		}
	}
}

func (pglr *PrimitiveGridLineRenderer) cleanupRightEdgeH(vpCurrent, vpPrevious Viewport, gridRegion GridRegion) {
	for rowVisIdx := vpCurrent.FirstRowVisIdx; rowVisIdx <= vpCurrent.LastRowVisIdx; rowVisIdx++ {
		lineIdx := pglr.hLineIndex[gridRegion][rowVisIdx]

		for colEnd := vpPrevious.LastColVisIdx; colEnd > vpCurrent.LastColVisIdx; colEnd-- {
			if itemId, exists := lineIdx.P2[colEnd]; exists {
				config := pglr.hLineItems[gridRegion].ConfigLines[itemId]
				if config.ColStart > vpCurrent.LastColVisIdx {
					pglr.hFlaggedItems[gridRegion].Put(itemId)
				} else {
					pglr.updatePosition2LineH(itemId, rowVisIdx, vpCurrent.LastColVisIdx, gridRegion)
				}
			}
		}
	}
}

func (pglr *PrimitiveGridLineRenderer) addNewLineH(container *fyne.Container, rowVisIdx, colStartVisIdx, colEndVisIdx int, gridRegion GridRegion) {
	var lineItem *canvas.Line

	if flaggedItemId, exist := pglr.hFlaggedItems[gridRegion].Get(); exist {

		origItem := pglr.hLineItems[gridRegion].ConfigLines[flaggedItemId]

		newItem := hLineConfig{
			Row:          rowVisIdx,
			ColStart:     colStartVisIdx,
			ColEnd:       colEndVisIdx,
			UnPositioned: true,
		}

		pglr.hLineItems[gridRegion].ConfigLines[flaggedItemId] = newItem

		lineItem = pglr.hLineItems[gridRegion].Lines[flaggedItemId]

		delete(pglr.hLineIndex[gridRegion][origItem.Row].P1, origItem.ColStart)

		delete(pglr.hLineIndex[gridRegion][origItem.Row].P2, origItem.ColEnd)

		pglr.hLineIndex[gridRegion][rowVisIdx].P1[colStartVisIdx] = flaggedItemId
		pglr.hLineIndex[gridRegion][rowVisIdx].P2[colEndVisIdx] = flaggedItemId

	} else {
		// No flagged item available; so goto the pool
		primitiveGridLineRecyleItem, recycledItem := pglr.hItemRecycler[gridRegion].Get()

		newConfigItem := hLineConfig{
			Row:          rowVisIdx,
			ColStart:     colStartVisIdx,
			ColEnd:       colEndVisIdx,
			UnPositioned: true,
		}
		if recycledItem {
			lineItem = pglr.hLineItems[gridRegion].Lines[primitiveGridLineRecyleItem.id]

			pglr.hLineItems[gridRegion].ConfigLines[primitiveGridLineRecyleItem.id] = newConfigItem

			pglr.hLineIndex[gridRegion][rowVisIdx].P1[colStartVisIdx] = primitiveGridLineRecyleItem.id
			pglr.hLineIndex[gridRegion][rowVisIdx].P2[colEndVisIdx] = primitiveGridLineRecyleItem.id
		} else {
			lineItem = primitiveGridLineRecyleItem.obj

			item := pglr.hLineItems[gridRegion]
			item.ConfigLines = append(item.ConfigLines, newConfigItem)
			item.Lines = append(item.Lines, primitiveGridLineRecyleItem.obj)

			pglr.hLineItems[gridRegion] = item

			itemId := len(item.Lines) - 1

			pglr.hLineIndex[gridRegion][rowVisIdx].P1[colStartVisIdx] = itemId
			pglr.hLineIndex[gridRegion][rowVisIdx].P2[colEndVisIdx] = itemId

			// the only plavce where we add content
			container.Add(lineItem)
		}
	}
}

func (pglr *PrimitiveGridLineRenderer) updatePosition1LineH(itemId, rowVisIdx, startVisColIdx int, gridRegion GridRegion) {

	lineItem := pglr.hLineItems[gridRegion].ConfigLines[itemId]

	originalColStart := lineItem.ColStart

	lineItem.ColStart = startVisColIdx
	lineItem.UnPositioned = true

	pglr.hLineItems[gridRegion].ConfigLines[itemId] = lineItem

	pglr.hLineIndex[gridRegion][rowVisIdx].P1[startVisColIdx] = itemId

	delete(pglr.hLineIndex[gridRegion][rowVisIdx].P1, originalColStart)

}

func (pglr *PrimitiveGridLineRenderer) updatePosition2LineH(itemId, rowVisIdx, endVisColIdx int, gridRegion GridRegion) {

	lineItem := pglr.hLineItems[gridRegion].ConfigLines[itemId]

	originalColEnd := lineItem.ColEnd

	lineItem.ColEnd = endVisColIdx
	lineItem.UnPositioned = true

	pglr.hLineItems[gridRegion].ConfigLines[itemId] = lineItem

	pglr.hLineIndex[gridRegion][rowVisIdx].P2[endVisColIdx] = itemId

	delete(pglr.hLineIndex[gridRegion][rowVisIdx].P2, originalColEnd)

}

func (pglr *PrimitiveGridLineRenderer) renderLeftEdge(container *fyne.Container,
	rowStart, rowEnd int,
	colStart, colEnd int,
	cache TransparencyCache,
	gridRegion GridRegion) {
	var mode mode
	var currentVisIdx int
	var startVisColIdx int
	var endVisColIdx int

	for rowVisIdx := rowStart; rowVisIdx <= rowEnd; rowVisIdx++ {

		if _, exist := pglr.hLineIndex[gridRegion][rowVisIdx]; !exist {
			pglr.hLineIndex[gridRegion][rowVisIdx] = lineIndex{
				P1: make(map[int]int),
				P2: make(map[int]int),
			}
		}

		startVisColIdx = -1
		endVisColIdx = -1
		currentVisIdx = colStart
		mode = MODE_SEARCHING

		for {
			if pglr.isGridLineRequiredH(cache, rowVisIdx, currentVisIdx) {
				if mode == MODE_SEARCHING {
					startVisColIdx = currentVisIdx
					mode = MODE_STARTING
				}
			} else if mode == MODE_STARTING {
				endVisColIdx = currentVisIdx - 1
				pglr.addNewLineH(container, rowVisIdx, startVisColIdx, endVisColIdx, gridRegion)

				mode = MODE_SEARCHING
			}
			if currentVisIdx == colEnd {
				if mode == MODE_STARTING {
					endVisColIdx = currentVisIdx
					if itemId, exist := pglr.hLineIndex[gridRegion][rowVisIdx].P1[endVisColIdx+1]; exist {
						pglr.updatePosition1LineH(itemId, rowVisIdx, startVisColIdx, gridRegion)
					} else {
						pglr.addNewLineH(container, rowVisIdx, startVisColIdx, endVisColIdx, gridRegion)
					}
				}
				break
			}
			currentVisIdx++
		}
	}
}

func (pglr *PrimitiveGridLineRenderer) renderRightEdge(container *fyne.Container,
	rowStart, rowEnd int,
	colStart, colEnd int,
	cache TransparencyCache,
	gridRegion GridRegion) {
	var mode mode
	var currentVisIdx int
	var startVisColIdx int
	var endVisColIdx int

	for rowVisIdx := rowStart; rowVisIdx <= rowEnd; rowVisIdx++ {
		if _, exist := pglr.hLineIndex[gridRegion][rowVisIdx]; !exist {
			pglr.hLineIndex[gridRegion][rowVisIdx] = lineIndex{
				P1: make(map[int]int),
				P2: make(map[int]int),
			}
		}

		startVisColIdx = -1
		endVisColIdx = -1
		currentVisIdx = colEnd
		mode = MODE_SEARCHING

		for {

			if currentVisIdx == 24 {
			}
			if pglr.isGridLineRequiredH(cache, rowVisIdx, currentVisIdx) {
				if mode == MODE_SEARCHING {
					endVisColIdx = currentVisIdx
					mode = MODE_STARTING
				}
			} else if mode == MODE_STARTING {
				startVisColIdx = currentVisIdx + 1

				pglr.addNewLineH(container, rowVisIdx, startVisColIdx, endVisColIdx, gridRegion)

				mode = MODE_SEARCHING
			}
			if currentVisIdx == colStart {
				if mode == MODE_STARTING {
					startVisColIdx = currentVisIdx
					if itemId, exist := pglr.hLineIndex[gridRegion][rowVisIdx].P2[startVisColIdx-1]; exist {
						pglr.updatePosition2LineH(itemId, rowVisIdx, endVisColIdx, gridRegion)

					} else {
						pglr.addNewLineH(container, rowVisIdx, startVisColIdx, endVisColIdx, gridRegion)
					}
				}
				break
			}
			currentVisIdx--
		}
	}
}
