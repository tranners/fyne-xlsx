package pkg

import (
	"image/color"

	"fyne.io/fyne/v2/canvas"
)

type mode int

const (
	MODE_SEARCHING mode = iota
	MODE_STARTING
	MODE_COMPLETED
)

var borderGridlineColor = color.NRGBA{R: 212, G: 212, B: 212, A: 225}

type lineIndex struct {
	P1 map[int]int
	P2 map[int]int
}

type hLineConfig struct {
	Row          int
	ColStart     int
	ColEnd       int
	UnPositioned bool
}

type vLineConfig struct {
	Col          int
	RowStart     int
	RowEnd       int
	UnPositioned bool
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

	hFlaggedItems map[GridRegion]*PrimitiveGridlineFlagger
	hItemRecycler map[GridRegion]*PrimitiveGridlineRecycler

	vFlaggedItems map[GridRegion]*PrimitiveGridlineFlagger
	vItemRecycler map[GridRegion]*PrimitiveGridlineRecycler
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
		hFlaggedItems: map[GridRegion]*PrimitiveGridlineFlagger{
			RegionMain:        NewPrimitiveGridlineFlagger(),
			RegionFixedCorner: NewPrimitiveGridlineFlagger(),
			RegionFrozenRows:  NewPrimitiveGridlineFlagger(),
			RegionFrozenCols:  NewPrimitiveGridlineFlagger(),
		},
		hItemRecycler: map[GridRegion]*PrimitiveGridlineRecycler{
			RegionMain:        NewPrimitiveGridlineRecycler(),
			RegionFixedCorner: NewPrimitiveGridlineRecycler(),
			RegionFrozenRows:  NewPrimitiveGridlineRecycler(),
			RegionFrozenCols:  NewPrimitiveGridlineRecycler(),
		},
		vFlaggedItems: map[GridRegion]*PrimitiveGridlineFlagger{
			RegionMain:        NewPrimitiveGridlineFlagger(),
			RegionFixedCorner: NewPrimitiveGridlineFlagger(),
			RegionFrozenRows:  NewPrimitiveGridlineFlagger(),
			RegionFrozenCols:  NewPrimitiveGridlineFlagger(),
		},
		vItemRecycler: map[GridRegion]*PrimitiveGridlineRecycler{
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
	item := PrimitiveGridlineRecyclerItem{id: id}
	cyl.items = append(cyl.items, item)
}

func (cyl *PrimitiveGridlineRecycler) Size() int {

	return len(cyl.items)
}
