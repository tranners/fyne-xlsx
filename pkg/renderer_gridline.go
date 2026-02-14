package pkg

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

type mode int

const (
	MODE_SEARCHING mode = iota
	MODE_STARTING
	MODE_COMPLETED
)

var borderGridlineColor = color.NRGBA{R: 212, G: 212, B: 212, A: 255}

type lineIndex struct {
	P1 map[int]int
	P2 map[int]int
}

type hLineConfig struct {
	Row      int
	ColStart int
	ColEnd   int
}

type vLineConfig struct {
	Col      int
	RowStart int
	rowEnd   int
}

type TransparencyCache map[CellID]bool

type hLineItems struct {
	Lines       []*canvas.Line
	ConfigLines []hLineConfig
}

type vLineItems struct {
	Lines       []*canvas.Line
	ConfigLines []vLineConfig
}

type PrimitiveGridLineRenderer struct {
	ctx *RenderContext

	hLineIndex map[GridRegion]map[int]lineIndex
	hLineItems map[GridRegion]hLineItems

	vLineIndex map[GridRegion]map[int]lineIndex
	vLineItems map[GridRegion]vLineItems

	flaggedItems map[GridRegion]*PrimitiveGridlineFlagger
	itemRecycler map[GridRegion]*PrimitiveGridlineRecycler
}

type PrimitiveGridlineRecycler struct {
	items []PrimitiveGridlineRecyclerItem
}

type PrimitiveGridlineRecyclerItem struct {
	id  int
	obj *canvas.Line
}

func NewPrimitiveGridlineRecycler() *PrimitiveGridlineRecycler {
	return &PrimitiveGridlineRecycler{
		items: []PrimitiveGridlineRecyclerItem{},
	}
}

type PrimitiveGridlineFlagger struct {
	// No mutex needed - each region has isolated instance and is processed sequentially
	items []int
}

func NewPrimitiveGridlineFlagger() *PrimitiveGridlineFlagger {
	return &PrimitiveGridlineFlagger{
		items: []int{},
	}
}

func NewPrimitiveGridLineRenderer(ctx *RenderContext) *PrimitiveGridLineRenderer {
	return &PrimitiveGridLineRenderer{
		ctx: ctx,

		hLineItems: map[GridRegion]hLineItems{
			RegionMain: {
				Lines:       []*canvas.Line{},
				ConfigLines: []hLineConfig{},
			},
			RegionFixedCorner: {
				Lines:       []*canvas.Line{},
				ConfigLines: []hLineConfig{},
			},
			RegionFrozenRows: {
				Lines:       []*canvas.Line{},
				ConfigLines: []hLineConfig{},
			},
			RegionFrozenCols: {
				Lines:       []*canvas.Line{},
				ConfigLines: []hLineConfig{},
			},
		},
		hLineIndex: map[GridRegion]map[int]lineIndex{
			RegionMain:        make(map[int]lineIndex),
			RegionFixedCorner: make(map[int]lineIndex),
			RegionFrozenRows:  make(map[int]lineIndex),
			RegionFrozenCols:  make(map[int]lineIndex),
		},
		vLineItems: map[GridRegion]vLineItems{
			RegionMain: {
				Lines:       []*canvas.Line{},
				ConfigLines: []vLineConfig{},
			},
			RegionFixedCorner: {
				Lines:       []*canvas.Line{},
				ConfigLines: []vLineConfig{},
			},
			RegionFrozenRows: {
				Lines:       []*canvas.Line{},
				ConfigLines: []vLineConfig{},
			},
			RegionFrozenCols: {
				Lines:       []*canvas.Line{},
				ConfigLines: []vLineConfig{},
			},
		},
		vLineIndex: map[GridRegion]map[int]lineIndex{
			RegionMain:        make(map[int]lineIndex),
			RegionFixedCorner: make(map[int]lineIndex),
			RegionFrozenRows:  make(map[int]lineIndex),
			RegionFrozenCols:  make(map[int]lineIndex),
		},
		flaggedItems: map[GridRegion]*PrimitiveGridlineFlagger{
			RegionMain:        NewPrimitiveGridlineFlagger(),
			RegionFixedCorner: NewPrimitiveGridlineFlagger(),
			RegionFrozenRows:  NewPrimitiveGridlineFlagger(),
			RegionFrozenCols:  NewPrimitiveGridlineFlagger(),
		},
		itemRecycler: map[GridRegion]*PrimitiveGridlineRecycler{
			RegionMain:        NewPrimitiveGridlineRecycler(),
			RegionFixedCorner: NewPrimitiveGridlineRecycler(),
			RegionFrozenRows:  NewPrimitiveGridlineRecycler(),
			RegionFrozenCols:  NewPrimitiveGridlineRecycler(),
		},
	}
}

func (cyl *PrimitiveGridlineFlagger) Get() (int, bool) {
	if len(cyl.items) > 0 {
		last := len(cyl.items) - 1
		c := cyl.items[last]
		cyl.items = cyl.items[:last]
		return c, true
	}
	return -1, false
}

func (pf *PrimitiveGridlineFlagger) Put(id int) {
	pf.items = append(pf.items, id)
}

func (pf *PrimitiveGridlineFlagger) Reset() {
	pf.items = pf.items[:0]
}

func (cyl *PrimitiveGridlineRecycler) Get() (PrimitiveGridlineRecyclerItem, bool) {
	if len(cyl.items) > 0 {
		last := len(cyl.items) - 1
		c := cyl.items[last]
		cyl.items = cyl.items[:last]
		return c, true
	}
	obj := canvas.NewLine(borderGridlineColor)
	return PrimitiveGridlineRecyclerItem{id: -1, obj: obj}, false
}

func (cyl *PrimitiveGridlineRecycler) Put(id int) {
	// tmp check for a duplicate
	for _, item := range cyl.items {
		if item.id == id {
			fmt.Printf("[ERR-DUPLICATE-ITEM] ID:%d\n", id)
		}
	}
	item := PrimitiveGridlineRecyclerItem{id: id}
	cyl.items = append(cyl.items, item)
}

func (cyl *PrimitiveGridlineRecycler) Size() int {

	return len(cyl.items)
}

func (pglr *PrimitiveGridLineRenderer) stashInCorner(gridRegion GridRegion) {
	// if we have items in the flagged pool; put to recycle pool, for use later
	for _, itemId := range pglr.flaggedItems[gridRegion].items {

		obj := pglr.hLineItems[gridRegion].Lines[itemId]

		objConfig := pglr.hLineItems[gridRegion].ConfigLines[itemId]

		obj.Move(fyne.NewPos(-9999, -9999))

		pglr.itemRecycler[gridRegion].Put(itemId)

		delete(pglr.hLineIndex[gridRegion][objConfig.Row].P1, objConfig.ColStart)

		delete(pglr.hLineIndex[gridRegion][objConfig.Row].P2, objConfig.ColEnd)

		pglr.hLineItems[gridRegion].ConfigLines[itemId] = hLineConfig{
			Row:      -1, // Invalid row
			ColStart: -1,
			ColEnd:   -1,
		}

	}
	pglr.flaggedItems[gridRegion].Reset()
}

func (pglr *PrimitiveGridLineRenderer) isHorisontalGridLineRequired(isTransparent TransparencyCache, rowVisIdx, colVisIdx int) bool {
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

func (pglr *PrimitiveGridLineRenderer) RecycleBinItems(gridRegion GridRegion) int {
	itemRecycler := pglr.itemRecycler[gridRegion]
	return itemRecycler.Size()
}

func (pglr *PrimitiveGridLineRenderer) removeRowsTop(vpCurrent, vpPrevious Viewport, gridRegion GridRegion) {
	for rowVisIdx := vpPrevious.FirstRowVisIdx; rowVisIdx < vpCurrent.FirstRowVisIdx; rowVisIdx++ {
		lineIdx := pglr.hLineIndex[gridRegion][rowVisIdx]

		for _, itemId := range lineIdx.P1 {
			pglr.flaggedItems[gridRegion].Put(itemId)
		}
		delete(pglr.hLineIndex[gridRegion], rowVisIdx)
	}
}

func (pglr *PrimitiveGridLineRenderer) removeRowsBottom(vpCurrent, vpPrevious Viewport, gridRegion GridRegion) {
	for rowVisIdx := vpCurrent.LastRowVisIdx + 1; rowVisIdx <= vpPrevious.LastRowVisIdx; rowVisIdx++ {
		lineIdx := pglr.hLineIndex[gridRegion][rowVisIdx]

		for _, itemId := range lineIdx.P1 {
			pglr.flaggedItems[gridRegion].Put(itemId)
		}
		delete(pglr.hLineIndex[gridRegion], rowVisIdx)
	}
}

func (pglr *PrimitiveGridLineRenderer) cleanupLeftEdge(vpCurrent, vpPrevious Viewport, gridRegion GridRegion) {
	for rowVisIdx := vpCurrent.FirstRowVisIdx; rowVisIdx <= vpCurrent.LastRowVisIdx; rowVisIdx++ {
		lineIdx := pglr.hLineIndex[gridRegion][rowVisIdx]

		for colStart := vpPrevious.FirstColVisIdx; colStart < vpCurrent.FirstColVisIdx; colStart++ {
			if itemId, exists := lineIdx.P1[colStart]; exists {
				config := pglr.hLineItems[gridRegion].ConfigLines[itemId]
				if config.ColEnd < vpCurrent.FirstColVisIdx {
					pglr.flaggedItems[gridRegion].Put(itemId)
				} else {
					pglr.updatePosition1HorizontalLine(itemId, rowVisIdx, vpCurrent.FirstColVisIdx, gridRegion)
				}
			}
		}
	}
}

func (pglr *PrimitiveGridLineRenderer) cleanupRightEdge(vpCurrent, vpPrevious Viewport, gridRegion GridRegion) {
	for rowVisIdx := vpCurrent.FirstRowVisIdx; rowVisIdx <= vpCurrent.LastRowVisIdx; rowVisIdx++ {
		lineIdx := pglr.hLineIndex[gridRegion][rowVisIdx]

		for colEnd := vpPrevious.LastColVisIdx; colEnd > vpCurrent.LastColVisIdx; colEnd-- {
			if itemId, exists := lineIdx.P2[colEnd]; exists {
				config := pglr.hLineItems[gridRegion].ConfigLines[itemId]
				if config.ColStart > vpCurrent.LastColVisIdx {
					pglr.flaggedItems[gridRegion].Put(itemId)
				} else {
					pglr.updatePosition2HorizontalLine(itemId, rowVisIdx, vpCurrent.LastColVisIdx, gridRegion)
				}
			}
		}
	}
}

func (pglr *PrimitiveGridLineRenderer) addNewHorizontalLine(container *fyne.Container, rowVisIdx, colStartVisIdx, colEndVisIdx int, gridRegion GridRegion) {
	cm := pglr.ctx.CoordManager
	var lineItem *canvas.Line

	if flaggedItemId, exist := pglr.flaggedItems[gridRegion].Get(); exist {

		origItem := pglr.hLineItems[gridRegion].ConfigLines[flaggedItemId]

		//fmt.Printf("[ORIG-FLAGGED-ITEM] OrigRow:%d, OrigCols:%d→%d, ItemID:%d, Flagged:%d\n",
		//	origItem.Row, origItem.ColStart, origItem.ColEnd, flaggedItemId, len(pglr.flaggedItems[gridRegion].items))

		newItem := hLineConfig{
			Row:      rowVisIdx,
			ColStart: colStartVisIdx,
			ColEnd:   colEndVisIdx,
		}
		//fmt.Printf("[FLAGGED-ITEM] OrigRow:%d, OrigCols:%d→%d, ItemID:%d, Flagged:%d\n",
		//	rowVisIdx, colStartVisIdx, colEndVisIdx, flaggedItemId, len(pglr.flaggedItems[gridRegion].items))

		pglr.hLineItems[gridRegion].ConfigLines[flaggedItemId] = newItem

		lineItem = pglr.hLineItems[gridRegion].Lines[flaggedItemId]

		delete(pglr.hLineIndex[gridRegion][origItem.Row].P1, origItem.ColStart)

		delete(pglr.hLineIndex[gridRegion][origItem.Row].P2, origItem.ColEnd)

		pglr.hLineIndex[gridRegion][rowVisIdx].P1[colStartVisIdx] = flaggedItemId
		pglr.hLineIndex[gridRegion][rowVisIdx].P2[colEndVisIdx] = flaggedItemId

	} else {
		// No flagged item available; so goto the pool
		primitiveGridLineRecyleItem, recycledItem := pglr.itemRecycler[gridRegion].Get()

		newConfigItem := hLineConfig{
			Row:      rowVisIdx,
			ColStart: colStartVisIdx,
			ColEnd:   colEndVisIdx,
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

	y := cm.GetRowPixelPosEndY(gridRegion, cm.GetRowModIdxFromVisIdx(rowVisIdx))

	lineItem.Position1.Y = y

	lineItem.Position1.X = cm.GetColPixelPosX(gridRegion, cm.GetColModIdxFromVisIdx(colStartVisIdx))

	lineItem.Position2.Y = y

	lineItem.Position2.X = cm.GetColPixelPosEndX(gridRegion, cm.GetColModIdxFromVisIdx(colEndVisIdx))

}

func (pglr *PrimitiveGridLineRenderer) updatePosition1HorizontalLine(itemId, rowVisIdx, startVisColIdx int, gridRegion GridRegion) {

	cm := pglr.ctx.CoordManager
	lineItem := pglr.hLineItems[gridRegion].ConfigLines[itemId]

	originalColStart := lineItem.ColStart

	lineItem.ColStart = startVisColIdx

	pglr.hLineItems[gridRegion].ConfigLines[itemId] = lineItem

	pglr.hLineIndex[gridRegion][rowVisIdx].P1[startVisColIdx] = itemId

	delete(pglr.hLineIndex[gridRegion][rowVisIdx].P1, originalColStart)

	line := pglr.hLineItems[gridRegion].Lines[itemId]

	line.Position1.X = cm.GetColPixelPosX(gridRegion, cm.GetColModIdxFromVisIdx(startVisColIdx))
}

func (pglr *PrimitiveGridLineRenderer) updatePosition2HorizontalLine(itemId, rowVisIdx, endVisColIdx int, gridRegion GridRegion) {

	cm := pglr.ctx.CoordManager
	lineItem := pglr.hLineItems[gridRegion].ConfigLines[itemId]

	originalColEnd := lineItem.ColEnd

	lineItem.ColEnd = endVisColIdx

	pglr.hLineItems[gridRegion].ConfigLines[itemId] = lineItem

	pglr.hLineIndex[gridRegion][rowVisIdx].P2[endVisColIdx] = itemId

	delete(pglr.hLineIndex[gridRegion][rowVisIdx].P2, originalColEnd)

	line := pglr.hLineItems[gridRegion].Lines[itemId]

	line.Position2.X = cm.GetColPixelPosEndX(gridRegion, cm.GetColModIdxFromVisIdx(endVisColIdx))
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

		//if rowVisIdx != 14 {
		//	continue
		//}

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
			if pglr.isHorisontalGridLineRequired(cache, rowVisIdx, currentVisIdx) {
				if mode == MODE_SEARCHING {
					startVisColIdx = currentVisIdx
					mode = MODE_STARTING
				}
			} else if mode == MODE_STARTING {
				endVisColIdx = currentVisIdx - 1

				if rowVisIdx == 9 {
					//fmt.Printf("TARGET-LIST-01] ThisRow:%d, ColStart:%d ColEnd:%d\n", rowVisIdx, startVisColIdx, endVisColIdx)
				}

				pglr.addNewHorizontalLine(container, rowVisIdx, startVisColIdx, endVisColIdx, gridRegion)

				mode = MODE_SEARCHING
			}
			if currentVisIdx == colEnd {
				if mode == MODE_STARTING {
					endVisColIdx = currentVisIdx

					if rowVisIdx == 9 {
						//fmt.Printf("TARGET-LIST-02] ThisRow:%d, ColStart:%d ColEnd:%d\n", rowVisIdx, startVisColIdx, endVisColIdx)
					}
					if itemId, exist := pglr.hLineIndex[gridRegion][rowVisIdx].P1[endVisColIdx+1]; exist {
						pglr.updatePosition1HorizontalLine(itemId, rowVisIdx, startVisColIdx, gridRegion)
					} else {
						pglr.addNewHorizontalLine(container, rowVisIdx, startVisColIdx, endVisColIdx, gridRegion)
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
		//if rowVisIdx != 14 {
		//	continue
		//}
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
				//fmt.Println("Here")
			}
			if pglr.isHorisontalGridLineRequired(cache, rowVisIdx, currentVisIdx) {
				if mode == MODE_SEARCHING {
					endVisColIdx = currentVisIdx
					mode = MODE_STARTING
				}
			} else if mode == MODE_STARTING {
				startVisColIdx = currentVisIdx + 1

				if rowVisIdx == 9 {
					//fmt.Printf("TARGET-LIST-03] ThisRow:%d, ColStart:%d ColEnd:%d\n", rowVisIdx, startVisColIdx, endVisColIdx)
				}

				pglr.addNewHorizontalLine(container, rowVisIdx, startVisColIdx, endVisColIdx, gridRegion)

				mode = MODE_SEARCHING
			}
			if currentVisIdx == colStart {
				if mode == MODE_STARTING {
					startVisColIdx = currentVisIdx
					if itemId, exist := pglr.hLineIndex[gridRegion][rowVisIdx].P2[startVisColIdx-1]; exist {

						//if rowVisIdx == 9 {
						//fmt.Printf("TARGET-LIST-04] ThisRow:%d, ColStart:%d ColEnd:%d\n", rowVisIdx, startVisColIdx, endVisColIdx)
						//}

						pglr.updatePosition2HorizontalLine(itemId, rowVisIdx, endVisColIdx, gridRegion)

					} else {
						//if rowVisIdx == 9 {
						//fmt.Printf("TARGET-LIST-05] ThisRow:%d, ColStart:%d ColEnd:%d\n", rowVisIdx, startVisColIdx, endVisColIdx)
						//}
						pglr.addNewHorizontalLine(container, rowVisIdx, startVisColIdx, endVisColIdx, gridRegion)
					}
				}
				break
			}
			currentVisIdx--
		}
	}
}
