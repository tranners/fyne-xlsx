package pkg

import (
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

type BorderLineRenderer struct {
	ctx               *RenderContext
	regionBorderLines map[GridRegion]map[CellID]*BorderLineCarcuss
	recyclers         map[GridRegion]*BorderRecycler
}

type BorderRecycler struct {
	mu    sync.Mutex
	items []*BorderLineCarcuss
}

type BorderLineCarcuss struct {
	Container        *fyne.Container
	BorderLineRight  *canvas.Line
	BorderLineBottom *canvas.Line
	// Secondary lines for double borders (only shown for BorderStyleDouble)
	BorderLineRight2  *canvas.Line
	BorderLineBottom2 *canvas.Line
}

func NewBorderRecycler() *BorderRecycler {
	return &BorderRecycler{
		mu:    sync.Mutex{},
		items: []*BorderLineCarcuss{},
	}
}

func NewBorderLineRenderer(ctx *RenderContext) *BorderLineRenderer {
	return &BorderLineRenderer{
		ctx: ctx,
		regionBorderLines: map[GridRegion]map[CellID]*BorderLineCarcuss{
			RegionMain:        make(map[CellID]*BorderLineCarcuss),
			RegionFixedCorner: make(map[CellID]*BorderLineCarcuss),
			RegionFrozenRows:  make(map[CellID]*BorderLineCarcuss),
			RegionFrozenCols:  make(map[CellID]*BorderLineCarcuss),
		},
		recyclers: map[GridRegion]*BorderRecycler{
			RegionMain:        NewBorderRecycler(),
			RegionFixedCorner: NewBorderRecycler(),
			RegionFrozenRows:  NewBorderRecycler(),
			RegionFrozenCols:  NewBorderRecycler(),
		},
	}
}

func (cyl *BorderRecycler) Get() (*BorderLineCarcuss, bool) {
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

	borderRight := canvas.NewLine(color.Transparent)
	borderBottom := canvas.NewLine(color.Transparent)
	borderRight2 := canvas.NewLine(color.Transparent)
	borderBottom2 := canvas.NewLine(color.Transparent)

	borderContainer := container.NewWithoutLayout(
		borderRight,
		borderBottom,
		borderRight2,
		borderBottom2,
	)
	return &BorderLineCarcuss{
		Container:         borderContainer,
		BorderLineRight:   borderRight,
		BorderLineBottom:  borderBottom,
		BorderLineRight2:  borderRight2,
		BorderLineBottom2: borderBottom2,
	}, false

}
func (cyl *BorderRecycler) Put(obj *BorderLineCarcuss) {
	cyl.mu.Lock()
	cyl.items = append(cyl.items, obj)
	cyl.mu.Unlock()
}

func (gl *BorderLineRenderer) addBorderLine(content *fyne.Container, cellMap map[CellID]*BorderLineCarcuss, rowModIdx, colModIdx int, gridRegion GridRegion) *BorderLineCarcuss {
	id := CellID{Row: rowModIdx, Col: colModIdx}
	if _, exists := cellMap[id]; exists {
		return nil
	}

	ctx := gl.ctx
	cm := ctx.CoordManager

	recycler := gl.recyclers[gridRegion]
	carcuss, recycled := recycler.Get()

	cellWidth := cm.GetWidthByModIdx(id.Col)
	cellHeight := cm.GetHeightByModIdx(id.Row)

	carcuss.BorderLineBottom.Position1 = fyne.NewPos(0, cellHeight)
	carcuss.BorderLineBottom.Position2 = fyne.NewPos(cellWidth, cellHeight)

	carcuss.BorderLineRight.Position1 = fyne.NewPos(cellWidth, 0)
	carcuss.BorderLineRight.Position2 = fyne.NewPos(cellWidth, cellHeight)

	gl.setBorderlineVisibility(carcuss, id)

	if gridRegion == RegionMain {
		carcuss.Container.Move(cm.GetPixelPos(gridRegion, id.Row, id.Col))
	}
	if !recycled {
		content.Add(carcuss.Container)
	} else {
		carcuss.Container.Show()
	}
	//content.Add(carcuss.Container)
	cellMap[id] = carcuss
	return carcuss
}

func (gl *BorderLineRenderer) setBorderlineVisibility(carcuss *BorderLineCarcuss, id CellID) {
	mm := gl.ctx.MergeManager

	anchor, isMerged := mm.GetCellAnchor(id)
	if isMerged || mm.IsVisibleMergeAnchor(id) {
		gl.setMergeBorderlineVisibility(carcuss, id, anchor)
		return
	}
	// default
	gl.setRegularBorderlineVisibility(carcuss, id)
}

func (gl *BorderLineRenderer) setMergeBorderlineVisibility(carcuss *BorderLineCarcuss, id CellID, anchor CellID) {
	mm := gl.ctx.MergeManager
	mergeRange, exists := mm.anchorToModelRange[anchor]
	if !exists {
		carcuss.BorderLineRight.Hide()
		carcuss.BorderLineBottom.Hide()
		return
	}

	_, _, endRow, endCol, _ := mergeRange.GetMergeModBounds()

	// Show gridlines only on merge boundaries
	if id.Col == endCol {
		if gl.shouldShowRightBorderline(carcuss, id) {
			carcuss.BorderLineRight.Show()
		} else {
			carcuss.BorderLineRight.Hide()
		}
	} else {
		carcuss.BorderLineRight.Hide()
	}

	if id.Row == endRow {
		if gl.shouldShowBottomBorderline(carcuss, id) {
			carcuss.BorderLineBottom.Show()
		} else {
			carcuss.BorderLineBottom.Hide()
		}
	} else {
		carcuss.BorderLineBottom.Hide()
	}
}

func (gl *BorderLineRenderer) setRegularBorderlineVisibility(carcuss *BorderLineCarcuss, id CellID) {

	if gl.shouldShowRightBorderline(carcuss, id) {
		carcuss.BorderLineRight.Show()
	} else {
		carcuss.BorderLineRight.Hide()
	}

	if gl.shouldShowBottomBorderline(carcuss, id) {
		carcuss.BorderLineBottom.Show()
	} else {
		carcuss.BorderLineBottom.Hide()
	}
}

func (gl *BorderLineRenderer) shouldShowRightBorderline(carcuss *BorderLineCarcuss, id CellID) bool {
	ctx := gl.ctx
	cm := ctx.CoordManager

	cellData := ctx.Data.GridData[id]

	principalBorderStyle := cellData.Border.Right.Style
	principalBorderWidth := cellData.Border.Right.Style.GetWidth()
	principalBorderColor := cellData.Border.Right.Color

	nextVisColIdx := cm.GetColVisIdxFromModIdx(id.Col)
	if nextVisColIdx < len(cm.visColMap)-1 {
		nextModColIdx := cm.GetColModIdxFromVisIdx(nextVisColIdx + 1)
		altCellId := CellID{Row: id.Row, Col: nextModColIdx}
		altCellData := ctx.Data.GridData[altCellId]
		altCellWidth := altCellData.Border.Left.Style.GetWidth()

		if altCellWidth > principalBorderWidth {
			principalBorderWidth = altCellWidth
			principalBorderColor = altCellData.Border.Left.Color
			principalBorderStyle = altCellData.Border.Left.Style
		}
	}

	if principalBorderWidth == 0 {
		carcuss.BorderLineRight2.Hide()
		return false
	}
	// Handle double border style
	if principalBorderStyle == BorderStyleDouble {
		gl.setupDoubleBorder(carcuss, "right", principalBorderColor, id)
	} else {
		// Single line border
		carcuss.BorderLineRight.StrokeWidth = principalBorderWidth
		carcuss.BorderLineRight.StrokeColor = principalBorderColor
		carcuss.BorderLineRight2.Hide()
	}
	return true
}

func (gl *BorderLineRenderer) shouldShowBottomBorderline(carcuss *BorderLineCarcuss, id CellID) bool {
	ctx := gl.ctx
	cm := ctx.CoordManager

	cellData := ctx.Data.GridData[id]

	principalBorderStyle := cellData.Border.Bottom.Style
	principalBorderWidth := cellData.Border.Bottom.Style.GetWidth()
	principalBorderColor := cellData.Border.Bottom.Color

	nextVisRowIdx := cm.GetRowVisIdxFromModIdx(id.Row)
	if nextVisRowIdx < len(cm.visRowMap)-1 {
		nextModRowIdx := cm.GetRowModIdxFromVisIdx(nextVisRowIdx + 1)
		altCellId := CellID{Row: nextModRowIdx, Col: id.Col}
		altCellData := ctx.Data.GridData[altCellId]
		altCellWidth := altCellData.Border.Top.Style.GetWidth()
		if altCellWidth > principalBorderWidth {
			principalBorderWidth = altCellWidth
			principalBorderColor = altCellData.Border.Top.Color
			principalBorderStyle = altCellData.Border.Top.Style
		}
	}

	if principalBorderWidth == 0 {
		carcuss.BorderLineBottom2.Hide()
		return false
	}
	// Handle double border style
	if principalBorderStyle == BorderStyleDouble {
		gl.setupDoubleBorder(carcuss, "bottom", principalBorderColor, id)
	} else {
		// Single line border
		carcuss.BorderLineBottom.StrokeWidth = principalBorderWidth
		carcuss.BorderLineBottom.StrokeColor = principalBorderColor
		carcuss.BorderLineBottom2.Hide()
	}
	return true
}

func (gl *BorderLineRenderer) setupDoubleBorder(carcuss *BorderLineCarcuss, side string, borderColor color.Color, id CellID) {
	cm := gl.ctx.CoordManager
	cellWidth := cm.GetWidthByModIdx(id.Col)
	cellHeight := cm.GetHeightByModIdx(id.Row)

	const lineWidth = float32(1) // Each line is 1px
	const gap = float32(1)       // 1px gap between lines

	if side == "right" {
		// Position first line on the right edge
		carcuss.BorderLineRight.Position1 = fyne.NewPos(cellWidth+gap, 0)
		carcuss.BorderLineRight.Position2 = fyne.NewPos(cellWidth+gap, cellHeight)
		carcuss.BorderLineRight.StrokeWidth = lineWidth
		carcuss.BorderLineRight.StrokeColor = borderColor

		// Position second line with gap offset (inward)
		offset := lineWidth + gap
		carcuss.BorderLineRight2.Position1 = fyne.NewPos(cellWidth-offset+gap, 0)
		carcuss.BorderLineRight2.Position2 = fyne.NewPos(cellWidth-offset+gap, cellHeight)
		carcuss.BorderLineRight2.StrokeWidth = lineWidth
		carcuss.BorderLineRight2.StrokeColor = borderColor
		carcuss.BorderLineRight2.Show()

	} else if side == "bottom" {
		// Position first line on the bottom edge
		carcuss.BorderLineBottom.Position1 = fyne.NewPos(0, cellHeight)
		carcuss.BorderLineBottom.Position2 = fyne.NewPos(cellWidth, cellHeight)
		carcuss.BorderLineBottom.StrokeWidth = lineWidth
		carcuss.BorderLineBottom.StrokeColor = borderColor

		// Position second line with gap offset (inward)
		offset := lineWidth + gap
		carcuss.BorderLineBottom2.Position1 = fyne.NewPos(0, cellHeight-offset)
		carcuss.BorderLineBottom2.Position2 = fyne.NewPos(cellWidth, cellHeight-offset)
		carcuss.BorderLineBottom2.StrokeWidth = lineWidth
		carcuss.BorderLineBottom2.StrokeColor = borderColor
		carcuss.BorderLineBottom2.Show()
	}
}

func (gl *BorderLineRenderer) removeBorderLine(content *fyne.Container, cellMap map[CellID]*BorderLineCarcuss, cellModID CellID, gridRegion GridRegion) {

	if cell, exists := cellMap[cellModID]; exists {

		cell.Container.Hide()

		cell.BorderLineRight.StrokeColor = color.Transparent
		cell.BorderLineRight2.StrokeColor = color.Transparent
		cell.BorderLineBottom.StrokeColor = color.Transparent
		cell.BorderLineBottom2.StrokeColor = color.Transparent

		cell.Container.Hide()

		recycler := gl.recyclers[gridRegion]
		recycler.Put(cell)

		delete(cellMap, cellModID)
	}
}
