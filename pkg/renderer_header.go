package pkg

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

type HDRType int

const (
	ROW HDRType = iota
	COLUMN
)

const headerFontSize = 11
const offscreenCoord float32 = -9999
const estimatedCapacity = 50

// headers
var headerBorderLightColor = color.NRGBA{R: 217, G: 217, B: 217, A: 170}
var headerBorderDarkColor = color.NRGBA{R: 212, G: 212, B: 212, A: 225}

type HeaderRenderer struct {
	ctx *RenderContext

	backgroundFixedRows *canvas.Rectangle
	backgroundRows      *canvas.Rectangle
	backgroundFixedCols *canvas.Rectangle
	backgroundCols      *canvas.Rectangle
	edgeFixedRows       *canvas.Line
	edgeRows            *canvas.Line
	edgeFixedCols       *canvas.Line
	edgeCols            *canvas.Line

	fyneMountedCountCols int
	fyneMountedCountRows int

	hdrItemsRows HDRItems
	hdrItemsCols HDRItems

	hdrLabelColY float32
	hdrLabelRowY float32
}

type HDRSlot struct {
	Label       *canvas.Text
	Line        *canvas.Line
	Initialised bool
}

type HDRItems struct {
	Slots    []HDRSlot
	Index    map[int]int
	hotPool  []int
	coldPool []int
}

func (hr *HeaderRenderer) hdrItems(t HDRType) (*HDRItems, int) {
	if t == COLUMN {
		return &hr.hdrItemsCols, hr.fyneMountedCountCols
	}
	return &hr.hdrItemsRows, hr.fyneMountedCountRows
}

func NewHeaderRenderer(ctx *RenderContext) *HeaderRenderer {

	probe := canvas.NewText("A", color.Black)
	probe.TextSize = headerFontSize
	heightCol := (HeaderHeight / 2) - (probe.MinSize().Height / 2)
	heightRow := probe.MinSize().Height / 2
	return &HeaderRenderer{
		ctx: ctx,

		backgroundFixedRows: canvas.NewRectangle(headerBorderLightColor),
		backgroundRows:      canvas.NewRectangle(headerBorderLightColor),
		backgroundFixedCols: canvas.NewRectangle(headerBorderLightColor),
		backgroundCols:      canvas.NewRectangle(headerBorderLightColor),
		edgeFixedRows:       canvas.NewLine(headerBorderDarkColor),
		edgeRows:            canvas.NewLine(headerBorderDarkColor),
		edgeFixedCols:       canvas.NewLine(headerBorderDarkColor),
		edgeCols:            canvas.NewLine(headerBorderDarkColor),

		hdrItemsRows: HDRItems{
			Slots: make([]HDRSlot, 0, estimatedCapacity),
			Index: make(map[int]int),
		},
		hdrItemsCols: HDRItems{
			Slots: make([]HDRSlot, 0, estimatedCapacity),
			Index: make(map[int]int),
		},
		hdrLabelColY: heightCol,
		hdrLabelRowY: heightRow,
	}
}

func (hr *HeaderRenderer) setWaterMark(t HDRType) {
	if t == COLUMN {
		hr.fyneMountedCountCols = len(hr.hdrItemsCols.Slots)
		return
	}
	hr.fyneMountedCountRows = len(hr.hdrItemsRows.Slots)
}

func (hr *HeaderRenderer) mountHeaderBasics(
	colHdr, colHdrFrozen, rowHdr, rowHdrFrozen *fyne.Container,
) {
	cm := hr.ctx.CoordManager

	// Scrollable col + row header backgrounds are always required
	colHdr.Add(hr.backgroundCols)
	colHdr.Add(hr.edgeCols)
	rowHdr.Add(hr.backgroundRows)
	rowHdr.Add(hr.edgeRows)

	// Frozen column header strip — only mounted when columns are frozen
	if cm.HasFrozenColumns() {
		colHdrFrozen.Add(hr.backgroundFixedCols)
		colHdrFrozen.Add(hr.edgeFixedCols)
	}

	// Frozen row header strip — only mounted when rows are frozen
	if cm.HasFrozenRows() {
		rowHdrFrozen.Add(hr.backgroundFixedRows)
		rowHdrFrozen.Add(hr.edgeFixedRows)
	}
}

func (hr *HeaderRenderer) allocateHeaderSlots(t HDRType) {
	items, _ := hr.hdrItems(t)

	ctx := hr.ctx
	vp := ctx.Viewports[RegionMain]

	var firstIdx, lastIdx int

	if t == COLUMN {
		firstIdx = vp.FirstColVisIdx
		lastIdx = vp.LastColVisIdx
	} else {
		firstIdx = vp.FirstRowVisIdx
		lastIdx = vp.LastRowVisIdx
	}

	for visIdx := firstIdx; visIdx <= lastIdx; visIdx++ {
		items.Slots = append(items.Slots, HDRSlot{})
		items.Index[visIdx] = len(items.Slots) - 1
	}
}

func (hr *HeaderRenderer) recycleHeadersSlots(t HDRType) {
	items, _ := hr.hdrItems(t)

	ctx := hr.ctx
	newVP := ctx.Viewports[RegionMain]
	oldVP := ctx.LastViewports[RegionMain]

	var oldFirstIdx, oldLastIdx, newFirstIdx, newLastIdx int

	if t == COLUMN {
		oldFirstIdx = oldVP.FirstColVisIdx
		oldLastIdx = oldVP.LastColVisIdx
		newFirstIdx = newVP.FirstColVisIdx
		newLastIdx = newVP.LastColVisIdx
	} else {
		oldFirstIdx = oldVP.FirstRowVisIdx
		oldLastIdx = oldVP.LastRowVisIdx
		newFirstIdx = newVP.FirstRowVisIdx
		newLastIdx = newVP.LastRowVisIdx
	}

	// Stash only the delta ranges that have scrolled out of view.
	// The stable overlap region is not iterated.

	// Left stash: old entries now to the left/above the new viewport
	for visIdx := oldFirstIdx; visIdx < newFirstIdx && visIdx <= oldLastIdx; visIdx++ {
		idx := items.Index[visIdx]
		delete(items.Index, visIdx)
		items.hotPool = append(items.hotPool, idx)
	}

	// Right stash: old entries now to the right/below the new viewport
	for visIdx := max(oldFirstIdx, newLastIdx+1); visIdx <= oldLastIdx; visIdx++ {
		idx := items.Index[visIdx]
		delete(items.Index, visIdx)
		items.hotPool = append(items.hotPool, idx)
	}

	// Assign slots only to the delta ranges that have scrolled into view.

	// Left assign: new entries entering from the left/above
	for visIdx := newFirstIdx; visIdx < oldFirstIdx && visIdx <= newLastIdx; visIdx++ {
		hr.assignSlot(items, visIdx)
	}

	// Right assign: new entries entering from the right/below
	for visIdx := max(newFirstIdx, oldLastIdx+1); visIdx <= newLastIdx; visIdx++ {
		hr.assignSlot(items, visIdx)
	}
}

// assignSlot assigns a recycled or new HDRSlot to the given visible index.
// Pulls first from the hot stash, then the cold pool, then allocates fresh.
func (hr *HeaderRenderer) assignSlot(items *HDRItems, visIdx int) {
	if len(items.hotPool) > 0 {
		id := items.hotPool[len(items.hotPool)-1]
		items.hotPool = items.hotPool[:len(items.hotPool)-1]
		items.Slots[id].Initialised = false
		items.Index[visIdx] = id
	} else if len(items.coldPool) > 0 {
		id := items.coldPool[len(items.coldPool)-1]
		items.coldPool = items.coldPool[:len(items.coldPool)-1]
		items.Slots[id].Initialised = false
		items.Index[visIdx] = id
	} else {
		items.Slots = append(items.Slots, HDRSlot{})
		items.Index[visIdx] = len(items.Slots) - 1
	}
}

func (hr *HeaderRenderer) renderFixedColumnHDRs(c *fyne.Container) {
	ctx := hr.ctx
	cm := ctx.CoordManager

	for colVisIdx := 1; colVisIdx <= cm.GetFrozenColumns(); colVisIdx++ {
		colModIdx := cm.GetColModIdxFromVisIdx(colVisIdx)
		x := cm.GetColPixelPosEndX(RegionFixedCorner, colModIdx)

		lbl, ln := newHeaderItem()

		ln.Position1 = fyne.NewPos(x, HeaderHeight*0.2)
		ln.Position2 = fyne.NewPos(x, HeaderHeight)
		c.Add(ln)

		hdrWidth := cm.GetWidthByModIdx(colModIdx)

		lbl.Text = columnIndexToName(colModIdx)
		lbl.Move(fyne.NewPos(x-hdrWidth/2, hr.hdrLabelColY))
		c.Add(lbl)
	}
}

func (hr *HeaderRenderer) renderFixedRowHDRs(c *fyne.Container) {
	ctx := hr.ctx
	cm := ctx.CoordManager

	for rowVisIdx := 1; rowVisIdx <= cm.GetFrozenRows(); rowVisIdx++ {
		rowModIdx := cm.GetRowModIdxFromVisIdx(rowVisIdx)
		y := cm.GetRowPixelPosEndY(RegionFixedCorner, rowModIdx)

		lbl, ln := newHeaderItem()

		ln.Position1 = fyne.NewPos(HeaderWidth*0.2, y)
		ln.Position2 = fyne.NewPos(HeaderWidth, y)
		c.Add(ln)

		hdrHeight := cm.GetHeightByModIdx(rowModIdx)

		lbl.Text = fmt.Sprintf("%d", rowModIdx+1)
		lbl.Move(fyne.NewPos(HeaderWidth/2, y-hdrHeight/2-hr.hdrLabelRowY))
		c.Add(lbl)
	}
}
func (hr *HeaderRenderer) UpdateScrollableHeaders(
	c *fyne.Container, t HDRType, renderAbsolute bool,
) {
	items, _ := hr.hdrItems(t)

	hr.setWaterMark(t)
	if len(items.Index) == 0 {
		hr.allocateHeaderSlots(t)
	} else {
		hr.recycleHeadersSlots(t)
	}
	hr.fyneAddContent(c, t)

	if t == COLUMN {
		hr.fynePositionColumnHeaders(renderAbsolute)
	} else {
		hr.fynePositionRowHeaders(renderAbsolute)
	}
	hr.fyneMoveStashedItems(items)
}

func (hr *HeaderRenderer) fyneAddContent(c *fyne.Container, t HDRType) {
	items, from := hr.hdrItems(t)

	for i := from; i < len(items.Slots); i++ {
		lbl, ln := newHeaderItem()

		items.Slots[i].Label = lbl
		items.Slots[i].Line = ln

		c.Add(lbl)
		c.Add(ln)
	}
}

func newHeaderItem() (*canvas.Text, *canvas.Line) {
	lbl := canvas.NewText("", color.Black)
	lbl.TextSize = headerFontSize
	lbl.Alignment = fyne.TextAlignCenter
	ln := canvas.NewLine(headerBorderDarkColor)
	return lbl, ln
}

func (hr *HeaderRenderer) fynePositionColumnHeaders(renderAbsolute bool) {
	cm := hr.ctx.CoordManager

	dx := cm.GetScrollDeltaX()

	for visIdx, slotIdx := range hr.hdrItemsCols.Index {
		slot := &hr.hdrItemsCols.Slots[slotIdx]
		lbl := slot.Label
		line := slot.Line

		colModIdx := cm.GetColModIdxFromVisIdx(visIdx)

		isNew := !slot.Initialised
		if isNew {
			lbl.Text = columnIndexToName(colModIdx)
			slot.Initialised = true
		}

		if isNew || renderAbsolute {
			w := cm.GetWidthByModIdx(colModIdx)
			x := cm.GetColPixelPosEndX(RegionFrozenRows, colModIdx)
			lbl.Move(fyne.NewPos(x-w/2, hr.hdrLabelColY))
			line.Position1 = fyne.NewPos(x-w, HeaderHeight*0.2)
			line.Position2 = fyne.NewPos(x-w, HeaderHeight)
		} else {
			pos := lbl.Position()
			lbl.Move(fyne.NewPos(pos.X-dx, pos.Y))
			line.Position1.X -= dx
			line.Position2.X -= dx
		}
	}
}
func (hr *HeaderRenderer) fynePositionRowHeaders(renderAbsolute bool) {
	cm := hr.ctx.CoordManager

	dy := cm.GetScrollDeltaY()

	for visIdx, slotIdx := range hr.hdrItemsRows.Index {
		slot := &hr.hdrItemsRows.Slots[slotIdx]
		lbl := slot.Label
		line := slot.Line

		rowModIdx := cm.GetRowModIdxFromVisIdx(visIdx)

		isNew := !slot.Initialised
		if isNew {
			lbl.Text = fmt.Sprintf("%d", rowModIdx+1)
			slot.Initialised = true
		}

		if isNew || renderAbsolute {
			h := cm.GetHeightByModIdx(rowModIdx)
			y := cm.GetRowPixelPosY(RegionFrozenCols, rowModIdx)
			lbl.Move(fyne.NewPos(HeaderWidth/2, y+h/2-hr.hdrLabelRowY))

			line.Position1 = fyne.NewPos(HeaderWidth*0.2, y+h)
			line.Position2 = fyne.NewPos(HeaderWidth, y+h)
		} else {
			pos := lbl.Position()
			lbl.Move(fyne.NewPos(pos.X, pos.Y-dy))
			line.Position1.Y -= dy
			line.Position2.Y -= dy
		}
	}
}

func (hr *HeaderRenderer) fyneMoveStashedItems(items *HDRItems) {
	for _, id := range items.hotPool {
		items.Slots[id].Label.Move(fyne.NewPos(offscreenCoord, offscreenCoord))
		items.Slots[id].Line.Position1 = fyne.NewPos(offscreenCoord, offscreenCoord)
		items.Slots[id].Line.Position2 = fyne.NewPos(offscreenCoord, offscreenCoord)
		items.coldPool = append(items.coldPool, id)
	}
	items.hotPool = items.hotPool[:0]
}

func columnIndexToName(col int) string {
	name := ""
	for col >= 0 {
		name = string(rune('A'+col%26)) + name
		col = col/26 - 1
	}
	return name
}

func (hr *HeaderRenderer) renderCornerHDR(c *fyne.Container) {
	bg := canvas.NewRectangle(headerBorderLightColor)
	bg.Resize(fyne.NewSize(HeaderWidth, HeaderHeight))
	bg.Move(fyne.NewPos(0, 0))

	c.Add(bg)

	bottomBorder := canvas.NewLine(headerBorderDarkColor)
	bottomBorder.Position1 = fyne.NewPos(0, HeaderHeight)
	bottomBorder.Position2 = fyne.NewPos(HeaderWidth, HeaderHeight)

	c.Add(bottomBorder)

	rightBorder := canvas.NewLine(headerBorderDarkColor)
	rightBorder.Position1 = fyne.NewPos(HeaderWidth, 0)
	rightBorder.Position2 = fyne.NewPos(HeaderWidth, HeaderHeight)

	c.Add(rightBorder)
}
