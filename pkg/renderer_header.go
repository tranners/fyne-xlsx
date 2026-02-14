package pkg

import (
	"fmt"
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// corner
var headerBackgroundColor = color.NRGBA{R: 243, G: 243, B: 243, A: 255} // #D9D9D9 - Light gray
var headerBorderColor = color.NRGBA{R: 198, G: 198, B: 198, A: 255}

// headers
var headerBorderLightColor = color.NRGBA{R: 198, G: 198, B: 198, A: 120}
var headerBorderDarkColor = color.NRGBA{R: 198, G: 198, B: 198, A: 255}

type HeaderCarcuss struct {
	Container    *fyne.Container
	Background   *canvas.Rectangle
	Label        *canvas.Text
	RightBorder  *canvas.Line
	BottomBorder *canvas.Line
}

type HeaderRenderer struct {
	ctx              *RenderContext
	colHeadersScroll map[int]*HeaderCarcuss
	colHeadersFixed  map[int]*HeaderCarcuss
	rowHeadersScroll map[int]*HeaderCarcuss
	rowHeadersFixed  map[int]*HeaderCarcuss
	recyclerRows     *HeaderRecycler
	recyclerColumns  *HeaderRecycler
}

type HeaderRecycler struct {
	mu    sync.Mutex
	items []*HeaderCarcuss
}

func NewHeaderRecycler() *HeaderRecycler {
	return &HeaderRecycler{
		mu:    sync.Mutex{},
		items: []*HeaderCarcuss{},
	}
}

// Headers and groups get SAME context
func NewHeaderRenderer(ctx *RenderContext) *HeaderRenderer {
	return &HeaderRenderer{
		ctx:              ctx,
		colHeadersScroll: make(map[int]*HeaderCarcuss),
		colHeadersFixed:  make(map[int]*HeaderCarcuss),
		rowHeadersScroll: make(map[int]*HeaderCarcuss),
		rowHeadersFixed:  make(map[int]*HeaderCarcuss),
		recyclerRows:     NewHeaderRecycler(),
		recyclerColumns:  NewHeaderRecycler(),
	}
}

func (cyl *HeaderRecycler) Get() (*HeaderCarcuss, bool) {
	cyl.mu.Lock()
	if len(cyl.items) > 0 {

		last := len(cyl.items) - 1
		c := cyl.items[last]
		cyl.items[last] = nil
		cyl.items = cyl.items[:last]

		cyl.mu.Unlock()
		return c, true
	}
	cyl.mu.Unlock()

	bg := canvas.NewRectangle(headerBackgroundColor)
	text := canvas.NewText("", color.Black)
	text.TextSize = 11
	text.Alignment = fyne.TextAlignCenter
	rightBorder := canvas.NewLine(headerBorderLightColor)
	bottomBorder := canvas.NewLine(headerBorderDarkColor)
	wrapper := container.NewWithoutLayout(bg, text, rightBorder, bottomBorder)

	return &HeaderCarcuss{
		Container:    wrapper,
		Background:   bg,
		Label:        text,
		RightBorder:  rightBorder,
		BottomBorder: bottomBorder,
	}, false

}
func (cyl *HeaderRecycler) Put(obj *HeaderCarcuss) {
	cyl.mu.Lock()
	cyl.items = append(cyl.items, obj)
	cyl.mu.Unlock()
}

func (hr *HeaderRenderer) renderColumnHeaders(colHdrContainer *fyne.Container) {
	ctx := hr.ctx
	cm := ctx.CoordManager
	new := ctx.Viewports[RegionMain]
	old := ctx.LastViewports[RegionMain]
	//old := ctx.SnapshotViewports[RegionMain]

	for colVisIdx := old.FirstColVisIdx; colVisIdx <= old.LastColVisIdx; colVisIdx++ {
		if colVisIdx < new.FirstColVisIdx || colVisIdx > new.LastColVisIdx {
			colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
			hr.removeHeaderItem(hr.recyclerColumns, hr.colHeadersScroll, colModIdx)
		}
	}

	for colVisIdx := new.FirstColVisIdx; colVisIdx <= new.LastColVisIdx; colVisIdx++ {
		if colVisIdx < old.FirstColVisIdx || colVisIdx > old.LastColVisIdx {
			colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
			hr.createColumnHeaderItem(colHdrContainer, hr.colHeadersScroll, colModIdx)
		}
	}

	for id, header := range hr.colHeadersScroll {
		x := cm.GetColPixelPosX(RegionFrozenRows, id)
		header.Container.Move(fyne.NewPos(x, 0))
	}
}
func (hr *HeaderRenderer) renderFullColumnHeaders(colHdrContainer *fyne.Container) {
	ctx := hr.ctx
	cm := ctx.CoordManager
	vp := ctx.Viewports[RegionMain]

	for colVisIdx := vp.FirstColVisIdx; colVisIdx <= vp.LastColVisIdx; colVisIdx++ {
		colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
		hr.createColumnHeaderItem(colHdrContainer, hr.colHeadersScroll, colModIdx)
	}

	for id, header := range hr.colHeadersScroll {
		x := cm.GetColPixelPosX(RegionFrozenRows, id)
		header.Container.Move(fyne.NewPos(x, 0))
	}
}
func (hr *HeaderRenderer) renderRowHeaders(rowHdrContainer *fyne.Container) {
	ctx := hr.ctx
	cm := ctx.CoordManager
	new := ctx.Viewports[RegionMain]
	old := ctx.LastViewports[RegionMain]

	for rowVisIdx := old.FirstRowVisIdx; rowVisIdx <= old.LastRowVisIdx; rowVisIdx++ {
		if rowVisIdx < new.FirstRowVisIdx || rowVisIdx > new.LastRowVisIdx {
			rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
			hr.removeHeaderItem(hr.recyclerRows, hr.rowHeadersScroll, rowModIdx)
		}
	}

	for rowVisIdx := new.FirstRowVisIdx; rowVisIdx <= new.LastRowVisIdx; rowVisIdx++ {
		if rowVisIdx < old.FirstRowVisIdx || rowVisIdx > old.LastRowVisIdx {
			rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
			hr.createRowHeaderItem(rowHdrContainer, hr.rowHeadersScroll, rowModIdx)
		}
	}

	for id, header := range hr.rowHeadersScroll {
		y := cm.GetRowPixelPosY(RegionFrozenCols, id)
		header.Container.Move(fyne.NewPos(0, y))
	}
}
func (hr *HeaderRenderer) renderFullRowHeaders(rowHdrContainer *fyne.Container) {
	ctx := hr.ctx
	cm := ctx.CoordManager
	vp := ctx.Viewports[RegionMain]

	for rowVisIdx := vp.FirstRowVisIdx; rowVisIdx <= vp.LastRowVisIdx; rowVisIdx++ {
		rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
		hr.createRowHeaderItem(rowHdrContainer, hr.rowHeadersScroll, rowModIdx)
	}

	for id, header := range hr.rowHeadersScroll {
		y := cm.GetRowPixelPosY(RegionFrozenCols, id)
		header.Container.Move(fyne.NewPos(0, y))
	}
}

func (hr *HeaderRenderer) renderFixedColumnHeaders(colHdrFixedContainer *fyne.Container) {
	ctx := hr.ctx
	cm := ctx.CoordManager

	for colVisIdx := 1; colVisIdx <= cm.GetVisibleFrozenColumns(); colVisIdx++ {
		colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
		hr.createColumnHeaderItem(colHdrFixedContainer, hr.colHeadersFixed, colModIdx)
	}

	for colModIdx, header := range hr.colHeadersFixed {
		x := cm.GetColPixelPosX(RegionFixedCorner, colModIdx)
		header.Container.Move(fyne.NewPos(x, 0))
	}
}

func (hr *HeaderRenderer) renderFixedRowHeaders(rowHdrFixedContainer *fyne.Container) {
	ctx := hr.ctx
	cm := ctx.CoordManager

	for rowVisIdx := 1; rowVisIdx <= cm.GetVisibleFrozenRows(); rowVisIdx++ {
		rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
		hr.createRowHeaderItem(rowHdrFixedContainer, hr.rowHeadersFixed, rowModIdx)
	}

	for rowModIdx, header := range hr.rowHeadersFixed {
		y := cm.GetRowPixelPosY(RegionFixedCorner, rowModIdx)
		header.Container.Move(fyne.NewPos(0, y))
	}
}

func (hr *HeaderRenderer) createColumnHeaderItem(content *fyne.Container, headerMap map[int]*HeaderCarcuss, colModIdx int) {
	if _, exists := headerMap[colModIdx]; exists {
		return
	}
	ctx := hr.ctx
	cm := ctx.CoordManager

	width := cm.GetWidthByModIdx(colModIdx)

	//header := ctx.HeaderPool.Get().(*HeaderCarcuss)

	recycler := hr.recyclerColumns
	header, recycled := recycler.Get()

	header.setSize(width, HeaderHeight)
	header.Label.Text = columnIndexToName(colModIdx)
	header.Label.TextSize = 11
	labelHeight := header.Label.MinSize().Height
	header.Label.Move(fyne.NewPos(width/2, (HeaderHeight-labelHeight)/2))

	if !recycled {
		content.Add(header.Container)
	} else {
		header.Container.Show()
	}

	headerMap[colModIdx] = header
}

func (hr *HeaderRenderer) createRowHeaderItem(content *fyne.Container, headerMap map[int]*HeaderCarcuss, rowModIdx int) {
	if _, exists := headerMap[rowModIdx]; exists {
		return
	}
	ctx := hr.ctx
	cm := ctx.CoordManager

	height := cm.GetHeightByModIdx(rowModIdx)

	//header := ctx.HeaderPool.Get().(*HeaderCarcuss)
	recycler := hr.recyclerRows
	header, recycled := recycler.Get()

	header.setSize(HeaderWidth, height)
	header.Label.Text = fmt.Sprintf("%d", rowModIdx+1)
	header.Label.TextSize = 11
	labelHeight := header.Label.MinSize().Height
	header.Label.Move(fyne.NewPos(HeaderWidth/2, (height-labelHeight)/2))

	if !recycled {
		content.Add(header.Container)
	} else {
		header.Container.Show()
	}

	headerMap[rowModIdx] = header
}

func (r *HeaderRenderer) removeHeaderItem(recycler *HeaderRecycler, cellMap map[int]*HeaderCarcuss, modIdx int) {
	if cell, exists := cellMap[modIdx]; exists {

		cell.Container.Hide()

		//recycler := r.recyclerRows
		recycler.Put(cell)

		//content.Remove(cell.Container)
		//pool.Put(cell)
		delete(cellMap, modIdx)
	}
}

func (h *HeaderCarcuss) setSize(width, height float32) {
	h.Background.Resize(fyne.NewSize(width, height))

	h.RightBorder.Position1 = fyne.NewPos(width, 0)
	h.RightBorder.Position2 = fyne.NewPos(width, height)

	h.BottomBorder.Position1 = fyne.NewPos(0, height)
	h.BottomBorder.Position2 = fyne.NewPos(width, height)

	h.Container.Resize(fyne.NewSize(width, height))
}

func columnIndexToName(col int) string {
	name := ""
	for col >= 0 {
		name = string(rune('A'+col%26)) + name
		col = col/26 - 1
	}
	return name
}

func (r *HeaderRenderer) RenderCorner(cnrHdrContainer *fyne.Container) {
	bg := canvas.NewRectangle(headerBackgroundColor)
	bg.Resize(fyne.NewSize(HeaderWidth, HeaderHeight))
	bg.Move(fyne.NewPos(0, 0))

	bottomBorder := canvas.NewLine(headerBorderColor)
	bottomBorder.Position1 = fyne.NewPos(0, HeaderHeight)
	bottomBorder.Position2 = fyne.NewPos(HeaderWidth, HeaderHeight)

	rightBorder := canvas.NewLine(headerBorderColor)
	rightBorder.Position1 = fyne.NewPos(HeaderWidth, 0)
	rightBorder.Position2 = fyne.NewPos(HeaderWidth, HeaderHeight)

	cnrHdrContainer.Objects = []fyne.CanvasObject{bg, bottomBorder, rightBorder}
	cnrHdrContainer.Refresh()
}
