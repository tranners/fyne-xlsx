package pkg

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

type PrimitiveRenderer struct {
	ctx *RenderContext

	fontManager *FontManager

	rectanglePrimitives      map[GridRegion][]*canvas.Rectangle
	rectanglePrimitivesIndex map[GridRegion]map[CellID]int

	textPrimitives      map[GridRegion][]*canvas.Text
	textPrimitivesIndex map[GridRegion]map[CellID]int

	flagRectanglePrimitives map[GridRegion]*PrimitiveFlagger
	flagTextPrimitives      map[GridRegion]*PrimitiveFlagger

	rectangleRecycler map[GridRegion]*PrimitiveRectangleRecycler
	textRecycler      map[GridRegion]*PrimitiveTextRecycler
}

type PrimitiveTextRecycler struct {
	items []PrimitiveTextRecyclerItem
}

type PrimitiveTextRecyclerItem struct {
	id  int
	obj *canvas.Text
}

type PrimitiveRectangleRecycler struct {
	items []PrimitiveRectangleRecyclerItem
}
type PrimitiveRectangleRecyclerItem struct {
	id  int
	obj *canvas.Rectangle
}

type PrimitiveFlagger struct {
	items []CellID
}

func NewPrimitiveFlagger() *PrimitiveFlagger {
	return &PrimitiveFlagger{
		items: []CellID{},
	}
}

func NewPrimitiveTextRecycler() *PrimitiveTextRecycler {
	return &PrimitiveTextRecycler{
		items: []PrimitiveTextRecyclerItem{},
	}
}

func NewPrimitiveRectangleRecycler() *PrimitiveRectangleRecycler {
	return &PrimitiveRectangleRecycler{
		items: []PrimitiveRectangleRecyclerItem{},
	}
}

func NewPrimitiveRenderer(ctx *RenderContext, fontMgr *FontManager) *PrimitiveRenderer {
	return &PrimitiveRenderer{
		ctx:                 ctx,
		fontManager:         fontMgr,
		rectanglePrimitives: make(map[GridRegion][]*canvas.Rectangle),
		rectanglePrimitivesIndex: map[GridRegion]map[CellID]int{
			RegionMain:        make(map[CellID]int),
			RegionFixedCorner: make(map[CellID]int),
			RegionFrozenRows:  make(map[CellID]int),
			RegionFrozenCols:  make(map[CellID]int),
		},
		textPrimitives: make(map[GridRegion][]*canvas.Text),
		textPrimitivesIndex: map[GridRegion]map[CellID]int{
			RegionMain:        make(map[CellID]int),
			RegionFixedCorner: make(map[CellID]int),
			RegionFrozenRows:  make(map[CellID]int),
			RegionFrozenCols:  make(map[CellID]int),
		},
		flagRectanglePrimitives: map[GridRegion]*PrimitiveFlagger{
			RegionMain:        NewPrimitiveFlagger(),
			RegionFixedCorner: NewPrimitiveFlagger(),
			RegionFrozenRows:  NewPrimitiveFlagger(),
			RegionFrozenCols:  NewPrimitiveFlagger(),
		},
		flagTextPrimitives: map[GridRegion]*PrimitiveFlagger{
			RegionMain:        NewPrimitiveFlagger(),
			RegionFixedCorner: NewPrimitiveFlagger(),
			RegionFrozenRows:  NewPrimitiveFlagger(),
			RegionFrozenCols:  NewPrimitiveFlagger(),
		},
		textRecycler: map[GridRegion]*PrimitiveTextRecycler{
			RegionMain:        NewPrimitiveTextRecycler(),
			RegionFixedCorner: NewPrimitiveTextRecycler(),
			RegionFrozenRows:  NewPrimitiveTextRecycler(),
			RegionFrozenCols:  NewPrimitiveTextRecycler(),
		},
		rectangleRecycler: map[GridRegion]*PrimitiveRectangleRecycler{
			RegionMain:        NewPrimitiveRectangleRecycler(),
			RegionFixedCorner: NewPrimitiveRectangleRecycler(),
			RegionFrozenRows:  NewPrimitiveRectangleRecycler(),
			RegionFrozenCols:  NewPrimitiveRectangleRecycler(),
		},
	}
}

func (pglr *PrimitiveRenderer) RecycleBinItems(gridRegion GridRegion) int {
	itemRecycler := pglr.textRecycler[gridRegion]
	return itemRecycler.Size()
}

func (cyl *PrimitiveTextRecycler) Size() int {

	return len(cyl.items)
}

func (cyl *PrimitiveTextRecycler) Get() (PrimitiveTextRecyclerItem, bool) {
	if len(cyl.items) > 0 {
		last := len(cyl.items) - 1
		c := cyl.items[last]
		cyl.items = cyl.items[:last]
		return c, true
	}
	text := canvas.NewText("", color.Transparent)
	text.Alignment = fyne.TextAlignLeading
	text.TextSize = 11
	return PrimitiveTextRecyclerItem{id: -1, obj: text}, false
}

func (cyl *PrimitiveTextRecycler) Put(id int, obj *canvas.Text) {
	item := PrimitiveTextRecyclerItem{id: id, obj: obj}
	cyl.items = append(cyl.items, item)
}

func (cyl *PrimitiveRectangleRecycler) Get() (PrimitiveRectangleRecyclerItem, bool) {
	if len(cyl.items) > 0 {
		last := len(cyl.items) - 1
		c := cyl.items[last]
		cyl.items = cyl.items[:last]
		return c, true
	}
	rect := canvas.NewRectangle(color.Transparent)

	return PrimitiveRectangleRecyclerItem{id: -1, obj: rect}, false
}

func (cyl *PrimitiveRectangleRecycler) Put(id int, obj *canvas.Rectangle) {
	item := PrimitiveRectangleRecyclerItem{id: id, obj: obj}
	cyl.items = append(cyl.items, item)
}

func (cyl *PrimitiveFlagger) Get() (CellID, bool) {
	if len(cyl.items) > 0 {
		last := len(cyl.items) - 1
		c := cyl.items[last]
		cyl.items = cyl.items[:last]
		return c, true
	}

	return CellID{}, false
}

func (pf *PrimitiveFlagger) Put(obj CellID) {
	pf.items = append(pf.items, obj)
}

func (pf *PrimitiveFlagger) Reset() {
	pf.items = pf.items[:0] // Keep capacity for next cycle
}

func (pcr *PrimitiveRenderer) removeCell(cellModId CellID, gridRegion GridRegion) {

	mm := pcr.ctx.MergeManager

	if mm.isCellInMergedRange(cellModId) {
		return
	}

	if _, exists := pcr.rectanglePrimitivesIndex[gridRegion][cellModId]; exists {
		pcr.flagRectanglePrimitives[gridRegion].Put(cellModId)
	}

	if _, exists := pcr.textPrimitivesIndex[gridRegion][cellModId]; exists {
		pcr.flagTextPrimitives[gridRegion].Put(cellModId)

	}
}

func (pcr *PrimitiveRenderer) updateVisibleMerges(containers RegionContainers, viewport Viewport, region GridRegion) {
	mm := pcr.ctx.MergeManager

	mm.ForEachVisibleMerge(func(merge *VisibleMerge) {
		modelAnchor := mm.modelAnchors[merge.MergeIdx]
		if merge.VisRowStart > viewport.LastRowVisIdx ||
			merge.VisRowEnd < viewport.FirstRowVisIdx ||
			merge.VisColStart > viewport.LastColVisIdx ||
			merge.VisColEnd < viewport.FirstColVisIdx {

			// Merge left viewport - flag for removal
			if _, exists := pcr.rectanglePrimitivesIndex[region][modelAnchor]; exists {
				pcr.flagRectanglePrimitives[region].Put(modelAnchor)
			}
			if _, exists := pcr.textPrimitivesIndex[region][modelAnchor]; exists {
				pcr.flagTextPrimitives[region].Put(modelAnchor)
			}
		} else {

			anchorCellModId := merge.VisAnchor
			anchorCellModId = modelAnchor
			_, rectExists := pcr.rectanglePrimitivesIndex[region][anchorCellModId]
			_, textExists := pcr.textPrimitivesIndex[region][anchorCellModId]

			cellData := pcr.ctx.Data.GridData[modelAnchor]

			needRect := cellData != nil && cellData.Style != nil &&
				cellData.Style.Fill.BgColor != color.Transparent && !rectExists

			needText := cellData != nil && cellData.Value != "" && !textExists

			size := merge.PixelSize

			pcr.addPrimitivesToCell(containers,
				anchorCellModId,
				region, needRect, needText, size, cellData)
		}
	})
}

func (pcr *PrimitiveRenderer) addPrimitivesToCell(containers RegionContainers, id CellID, gridRegion GridRegion,
	isRectRequired bool,
	isTextRequired bool,
	size fyne.Size,
	cellData *CellData) {

	cm := pcr.ctx.CoordManager

	var recycledItem bool
	if isRectRequired {
		var rectanglePrimiticeItem *canvas.Rectangle
		var primitiveRectangleRecyleItem PrimitiveRectangleRecyclerItem
		// attempt to grab from the cell flagged becuase it left the viewport
		if flagCellId, ok := pcr.flagRectanglePrimitives[gridRegion].Get(); ok {
			idx := pcr.rectanglePrimitivesIndex[gridRegion][flagCellId]

			rectanglePrimiticeItem = pcr.rectanglePrimitives[gridRegion][idx]
			primitiveRectangleRecyleItem = PrimitiveRectangleRecyclerItem{id: idx, obj: rectanglePrimiticeItem}
			delete(pcr.rectanglePrimitivesIndex[gridRegion], flagCellId)
			pcr.rectanglePrimitivesIndex[gridRegion][id] = idx
		} else {
			// default to grab from the pool
			primitiveRectangleRecyleItem, recycledItem = pcr.rectangleRecycler[gridRegion].Get()
			if recycledItem {
				pcr.rectanglePrimitivesIndex[gridRegion][id] = primitiveRectangleRecyleItem.id
			} else {
				pcr.rectanglePrimitives[gridRegion] = append(pcr.rectanglePrimitives[gridRegion], primitiveRectangleRecyleItem.obj)
				containers.Background.Add(primitiveRectangleRecyleItem.obj)
				pcr.rectanglePrimitivesIndex[gridRegion][id] = len(pcr.rectanglePrimitives[gridRegion]) - 1
			}
		}

		primitiveRectangleRecyleItem.obj.FillColor = cellData.Style.Fill.BgColor
		primitiveRectangleRecyleItem.obj.Resize(size)
		if gridRegion == RegionMain {
			primitiveRectangleRecyleItem.obj.Move(cm.GetPixelPos(gridRegion, id.Row, id.Col))
		}
	}
	if isTextRequired {
		var textPrimiticeItem *canvas.Text
		var primitiveTextRecyleItem PrimitiveTextRecyclerItem
		// attempt to grab from the cell flagged becuase it left the viewport
		if flagCellId, ok := pcr.flagTextPrimitives[gridRegion].Get(); ok {
			idx := pcr.textPrimitivesIndex[gridRegion][flagCellId]

			textPrimiticeItem = pcr.textPrimitives[gridRegion][idx]
			primitiveTextRecyleItem = PrimitiveTextRecyclerItem{id: idx, obj: textPrimiticeItem}
			delete(pcr.textPrimitivesIndex[gridRegion], flagCellId)
			pcr.textPrimitivesIndex[gridRegion][id] = idx
		} else {
			// default to grab from the pool
			primitiveTextRecyleItem, recycledItem = pcr.textRecycler[gridRegion].Get()

			if recycledItem {
				pcr.textPrimitivesIndex[gridRegion][id] = primitiveTextRecyleItem.id
			} else {
				pcr.textPrimitives[gridRegion] = append(pcr.textPrimitives[gridRegion], primitiveTextRecyleItem.obj)
				containers.Data.Add(primitiveTextRecyleItem.obj)
				pcr.textPrimitivesIndex[gridRegion][id] = len(pcr.textPrimitives[gridRegion]) - 1
			}
		}
		pcr.setTextValue(primitiveTextRecyleItem.obj, cellData.Value, &cellData.Style.Font)
		if gridRegion == RegionMain {
			primitiveTextRecyleItem.obj.Move(cm.GetPixelPos(gridRegion, id.Row, id.Col))
		}
	}
}

func (pcr *PrimitiveRenderer) renderCell(containers RegionContainers, rowModIdx, colModIdx int, gridRegion GridRegion) {
	mm := pcr.ctx.MergeManager

	modCellId := CellID{Row: rowModIdx, Col: colModIdx}

	_, rectExists := pcr.rectanglePrimitivesIndex[gridRegion][modCellId]

	_, textExists := pcr.textPrimitivesIndex[gridRegion][modCellId]

	if rectExists && textExists {
		return
	}

	cellData := pcr.ctx.Data.GridData[modCellId]

	needRect := cellData != nil && cellData.Style != nil &&
		cellData.Style.Fill.BgColor != color.Transparent && !rectExists

	needText := cellData != nil && cellData.Value != "" && !textExists

	if !needRect && !needText {
		return
	}

	if mm.isCellInMergedRange(modCellId) {
		return
	}

	size := pcr.ctx.CoordManager.GetCellSizeByModIdx(rowModIdx, colModIdx)

	pcr.addPrimitivesToCell(containers,
		modCellId,
		gridRegion, needRect, needText, size, cellData)
}

func (pcr *PrimitiveRenderer) moveToCorner(gridRegion GridRegion) {
	// if we have items in the flagged pool; put to recycle pool, for use later
	for _, cellId := range pcr.flagRectanglePrimitives[gridRegion].items {

		idx := pcr.rectanglePrimitivesIndex[gridRegion][cellId]

		obj := pcr.rectanglePrimitives[gridRegion][idx]

		obj.Move(fyne.NewPos(-9999, -9999))

		pcr.rectangleRecycler[gridRegion].Put(idx, obj)
		delete(pcr.rectanglePrimitivesIndex[gridRegion], cellId)
	}

	pcr.flagRectanglePrimitives[gridRegion].Reset()

	for _, cellId := range pcr.flagTextPrimitives[gridRegion].items {

		idx := pcr.textPrimitivesIndex[gridRegion][cellId]

		obj := pcr.textPrimitives[gridRegion][idx]

		obj.Move(fyne.NewPos(-9999, -9999))

		pcr.textRecycler[gridRegion].Put(idx, obj)
		delete(pcr.textPrimitivesIndex[gridRegion], cellId)
	}

	pcr.flagTextPrimitives[gridRegion].Reset()
}

func (pcr *PrimitiveRenderer) setTextValue(text *canvas.Text, val string, font *FontStyle) {

	fontMgr := pcr.fontManager

	text.Text = val

	if font != nil {
		// Set font size
		if font.Size > 0 {
			text.TextSize = font.Size
		}

		// Set color
		if font.Color != nil {
			text.Color = font.Color
		} else {
			text.Color = color.Black
		}

		// Select appropriate font variant
		if fontMgr != nil {
			text.FontSource = fontMgr.SelectFont(font.Bold, font.Italic)
		}

		// Leave TextStyle empty when using real font variants
		text.TextStyle = fyne.TextStyle{}
	} else {
		text.TextSize = 11
		text.TextStyle = fyne.TextStyle{}
		text.Color = color.Black
		if fontMgr != nil {
			text.FontSource = fontMgr.Regular
		}
	}
}
