package pkg

import (
	"sort"

	"fyne.io/fyne/v2"
)

type Viewport struct {
	FirstRowVisIdx int
	LastRowVisIdx  int
	FirstColVisIdx int
	LastColVisIdx  int
}

type RowLayout struct {
	ModIdx     int
	Height     float32
	PixelStart float32
	PixelEnd   float32
}

type ColLayout struct {
	ModIdx     int
	Width      float32
	PixelStart float32
	PixelEnd   float32
}

type CoordinateManager struct {
	visRowMap      []RowLayout
	modToVisRowMap []int

	visColMap      []ColLayout
	modToVisColMap []int

	totalVisHeight float32
	totalVisWidth  float32

	scrollOffset     fyne.Position
	prevScrollOffset fyne.Position

	freezeRowSplit int // Model row index for freeze split
	freezeColSplit int // Model col index for freeze split

	freezeRowPixelEnd float32
	freezeColPixelEnd float32

	viewports []Viewport
}

func NewCoordinateManager() *CoordinateManager {
	return &CoordinateManager{
		visRowMap: []RowLayout{{ModIdx: -1, Height: 0, PixelStart: 0, PixelEnd: 0}},
		visColMap: []ColLayout{{ModIdx: -1, Width: 0, PixelStart: 0, PixelEnd: 0}},
	}
}

func (cm *CoordinateManager) rebuildRowLayout(grid *WorkSheetData) {

	cm.freezeRowSplit = grid.FreezePanes.YSplit

	cm.visRowMap = cm.visRowMap[:1]
	cm.visRowMap[0] = RowLayout{ModIdx: -1, Height: 0, PixelStart: 0}

	cm.modToVisRowMap = make([]int, grid.RowsNof)

	y := float32(0)
	for modIdx := 0; modIdx < grid.RowsNof; modIdx++ {
		if !grid.HiddenRows[modIdx] {
			height := grid.RowHeights[modIdx]
			cm.visRowMap = append(cm.visRowMap, RowLayout{
				ModIdx:     modIdx,
				Height:     height,
				PixelStart: y,
				PixelEnd:   y + height,
			})
			cm.modToVisRowMap[modIdx] = len(cm.visRowMap) - 1
			y += height
		} else {
			cm.modToVisRowMap[modIdx] = -1
		}
	}
	cm.totalVisHeight = y
	cm.freezeRowPixelEnd = cm.visRowMap[cm.freezeRowSplit].PixelEnd
}

func (cm *CoordinateManager) rebuildColLayout(grid *WorkSheetData) {

	cm.freezeColSplit = grid.FreezePanes.XSplit

	cm.visColMap = cm.visColMap[:1]
	cm.visColMap[0] = ColLayout{ModIdx: -1, Width: 0, PixelStart: 0}
	cm.modToVisColMap = make([]int, grid.ColsNof) // or resize to current used rows

	x := float32(0)
	for modIdx := 0; modIdx < grid.ColsNof; modIdx++ {
		if !grid.HiddenColumns[modIdx] {
			width := grid.ColWidths[modIdx]
			cm.visColMap = append(cm.visColMap, ColLayout{
				ModIdx:     modIdx,
				Width:      width,
				PixelStart: x,
				PixelEnd:   x + width,
			})
			cm.modToVisColMap[modIdx] = len(cm.visColMap) - 1
			x += width
		} else {
			cm.modToVisColMap[modIdx] = -1
		}
	}
	cm.totalVisWidth = x
	cm.freezeColPixelEnd = cm.visColMap[cm.freezeColSplit].PixelEnd
}
func (cm *CoordinateManager) GetColPixelPosEndX(region GridRegion, colModIdx int) float32 {
	colVisIdx := cm.modToVisColMap[colModIdx]
	x := cm.visColMap[colVisIdx].PixelEnd

	switch region {
	case RegionMain:
		x -= cm.freezeColPixelEnd
	case RegionFrozenRows:
		x -= cm.freezeColPixelEnd + cm.scrollOffset.X
	case RegionFixedCorner, RegionFrozenCols:
		// no adjustment
	}
	return x
}
func (cm *CoordinateManager) GetColPixelPosX(region GridRegion, colModIdx int) float32 {
	colVisIdx := cm.modToVisColMap[colModIdx]
	x := cm.visColMap[colVisIdx].PixelStart

	switch region {
	case RegionMain:
		x -= cm.freezeColPixelEnd
	case RegionFrozenRows:
		x -= cm.freezeColPixelEnd + cm.scrollOffset.X
	case RegionFixedCorner, RegionFrozenCols:
		// no adjustment
	}
	return x
}

func (cm *CoordinateManager) GetRowPixelPosEndY(region GridRegion, rowModIdx int) float32 {
	rowVisIdx := cm.modToVisRowMap[rowModIdx]
	y := cm.visRowMap[rowVisIdx].PixelEnd

	switch region {
	case RegionMain:
		y -= cm.freezeRowPixelEnd
	case RegionFrozenCols:
		y -= cm.freezeRowPixelEnd + cm.scrollOffset.Y
	case RegionFixedCorner, RegionFrozenRows:
		// no adjustment
	}
	return y
}
func (cm *CoordinateManager) GetRowPixelPosY(region GridRegion, rowModIdx int) float32 {
	rowVisIdx := cm.modToVisRowMap[rowModIdx]
	y := cm.visRowMap[rowVisIdx].PixelStart

	switch region {
	case RegionMain:
		y -= cm.freezeRowPixelEnd
	case RegionFrozenCols:
		y -= cm.freezeRowPixelEnd + cm.scrollOffset.Y
	case RegionFixedCorner, RegionFrozenRows:
		// no adjustment
	}
	return y
}

func (cm *CoordinateManager) GetPixelPos(region GridRegion, rowModIdx, colModIdx int) fyne.Position {
	colVisIdx := cm.modToVisColMap[colModIdx]
	rowVisIdx := cm.modToVisRowMap[rowModIdx]

	x := cm.visColMap[colVisIdx].PixelStart
	y := cm.visRowMap[rowVisIdx].PixelStart

	switch region {
	case RegionMain:
		x -= cm.freezeColPixelEnd
		y -= cm.freezeRowPixelEnd
	case RegionFrozenRows:
		x -= cm.freezeColPixelEnd + cm.scrollOffset.X
	case RegionFrozenCols:
		y -= cm.freezeRowPixelEnd + cm.scrollOffset.Y
	case RegionFixedCorner:
		// no adjustment
	}
	return fyne.NewPos(x, y)
}

func (cm *CoordinateManager) GetRowPixelEndByVisIdx(visIdx int) float32 {
	return cm.visRowMap[visIdx].PixelEnd
}

func (cm *CoordinateManager) GetRowPixelStartByVisIdx(visIdx int) float32 {
	return cm.visRowMap[visIdx].PixelStart
}

func (cm *CoordinateManager) GetColPixelEndByVisIdx(visIdx int) float32 {
	return cm.visColMap[visIdx].PixelEnd
}

func (cm *CoordinateManager) GetColPixelStartByVisIdx(visIdx int) float32 {
	return cm.visColMap[visIdx].PixelStart
}

func (cm *CoordinateManager) GetFreezeOffsets() (colOffset, rowOffset float32) {
	colOffset = cm.freezeColPixelEnd
	rowOffset = cm.freezeRowPixelEnd
	return
}

func (cm *CoordinateManager) SetScrollOffset(offset fyne.Position) {
	cm.prevScrollOffset = cm.scrollOffset
	cm.scrollOffset = offset
}

func (cm *CoordinateManager) GetScrollDeltaY() float32 {
	//return cm.prevScrollOffset.Y - cm.scrollOffset.Y
	return cm.scrollOffset.Y - cm.prevScrollOffset.Y
}

func (cm *CoordinateManager) GetScrollDeltaX() float32 {
	//return cm.prevScrollOffset.X - cm.scrollOffset.X
	return cm.scrollOffset.X - cm.prevScrollOffset.X
}

func (cm *CoordinateManager) GetScrollableSize() fyne.Size {
	freezeColOffset, freezeRowOffset := cm.GetFreezeOffsets()
	return fyne.NewSize(
		cm.totalVisWidth-freezeColOffset,
		cm.totalVisHeight-freezeRowOffset,
	)
}

func (cm *CoordinateManager) CalculateViewports(scrollSize fyne.Size) [4]Viewport {

	mainFirstRow := cm.findFirstVisRowIdx()
	mainLastRow := cm.findLastVisRowIdx(scrollSize.Height)
	mainFirstCol := cm.findFirstVisColIdx()
	mainLastCol := cm.findLastVisColIdx(scrollSize.Width)

	var viewports [4]Viewport

	viewports[RegionFixedCorner] = Viewport{
		FirstRowVisIdx: 1,
		LastRowVisIdx:  cm.freezeRowSplit,
		FirstColVisIdx: 1,
		LastColVisIdx:  cm.freezeColSplit,
	}

	viewports[RegionFrozenRows] = Viewport{
		FirstRowVisIdx: 1,
		LastRowVisIdx:  cm.freezeRowSplit,
		FirstColVisIdx: mainFirstCol,
		LastColVisIdx:  mainLastCol,
	}

	viewports[RegionFrozenCols] = Viewport{
		FirstRowVisIdx: mainFirstRow,
		LastRowVisIdx:  mainLastRow,
		FirstColVisIdx: 1,
		LastColVisIdx:  cm.freezeColSplit,
	}

	viewports[RegionMain] = Viewport{
		FirstRowVisIdx: mainFirstRow,
		LastRowVisIdx:  mainLastRow,
		FirstColVisIdx: mainFirstCol,
		LastColVisIdx:  mainLastCol,
	}

	cm.viewports = viewports[:]

	return viewports
}
func (cm *CoordinateManager) GetViewportForRegion(region GridRegion) Viewport {
	return cm.viewports[region]
}

func (cm *CoordinateManager) findFirstVisRowIdx() int {
	adjustedY := cm.scrollOffset.Y
	adjustedY += cm.freezeRowPixelEnd

	idx := sort.Search(len(cm.visRowMap), func(i int) bool {
		return cm.visRowMap[i].PixelEnd > adjustedY
	})

	return idx

}

func (cm *CoordinateManager) findLastVisRowIdx(height float32) int {
	adjustedY := cm.scrollOffset.Y + height
	adjustedY += cm.freezeRowPixelEnd

	idx := sort.Search(len(cm.visRowMap), func(i int) bool {
		return cm.visRowMap[i].PixelEnd >= adjustedY
	})

	if idx >= len(cm.visRowMap) {
		idx = len(cm.visRowMap) - 1
	}

	return idx
}

func (cm *CoordinateManager) findFirstVisColIdx() int {
	adjustedX := cm.scrollOffset.X
	adjustedX += cm.freezeColPixelEnd

	idx := sort.Search(len(cm.visColMap), func(i int) bool {
		return cm.visColMap[i].PixelEnd > adjustedX
	})

	return idx
}

func (cm *CoordinateManager) findLastVisColIdx(width float32) int {
	adjustedX := cm.scrollOffset.X + width
	adjustedX += cm.freezeColPixelEnd

	idx := sort.Search(len(cm.visColMap), func(i int) bool {
		return cm.visColMap[i].PixelEnd >= adjustedX
	})

	if idx >= len(cm.visColMap) {
		idx = len(cm.visColMap) - 1
	}

	return idx
}

func (cm *CoordinateManager) GetRowVisIdxFromModIdx(rowModIdx int) int {
	return cm.modToVisRowMap[rowModIdx]
}

func (cm *CoordinateManager) GetColVisIdxFromModIdx(colModIdx int) int {
	return cm.modToVisColMap[colModIdx]
}

func (cm *CoordinateManager) GetWidthByVisIdx(visIdx int) float32 {
	return cm.visColMap[visIdx].Width
}

func (cm *CoordinateManager) GetHeightByVisIdx(visIdx int) float32 {
	return cm.visRowMap[visIdx].Height
}

func (cm *CoordinateManager) GetColModIdxFromVisIdx(visIdx int) int {
	return cm.visColMap[visIdx].ModIdx
}

func (cm *CoordinateManager) GetRowModIdxFromVisIdx(visIdx int) int {
	return cm.visRowMap[visIdx].ModIdx
}

func (cm *CoordinateManager) GetCellSizeByModIdx(rowModIdx, colModIdx int) fyne.Size {

	width := cm.GetWidthByModIdx(colModIdx)
	height := cm.GetHeightByModIdx(rowModIdx)

	return fyne.NewSize(width, height)
}

func (cm *CoordinateManager) GetHeightByModIdx(rowModIdx int) float32 {
	rowVisIdx := cm.modToVisRowMap[rowModIdx]
	height := cm.visRowMap[rowVisIdx].Height

	return height
}

func (cm *CoordinateManager) GetWidthByModIdx(colModIdx int) float32 {
	colVisIdx := cm.modToVisColMap[colModIdx]
	width := cm.visColMap[colVisIdx].Width

	return width
}

func (cm *CoordinateManager) GetFrozenRows() int {
	return cm.freezeRowSplit
}
func (cm *CoordinateManager) HasFrozenRows() bool {
	return cm.freezeColSplit > 0
}

func (cm *CoordinateManager) GetFrozenColumns() int {
	return cm.freezeColSplit
}

func (cm *CoordinateManager) HasFrozenColumns() bool {
	return cm.freezeColSplit > 0
}

func (cm *CoordinateManager) FindFirstVisibleColInRange(startModIdx, endModIdx int) int {
	for modIdx := startModIdx; modIdx <= endModIdx; modIdx++ {
		if cm.GetColVisIdxFromModIdx(modIdx) != -1 {
			return modIdx
		}
	}
	return -1
}

func (cm *CoordinateManager) FindLastVisibleColInRange(startModIdx, endModIdx int) int {
	for modIdx := endModIdx; modIdx >= startModIdx; modIdx-- {
		if cm.GetColVisIdxFromModIdx(modIdx) != -1 {
			return modIdx
		}
	}
	return -1
}

func (cm *CoordinateManager) FindFirstVisibleRowInRange(startModIdx, endModIdx int) int {
	for modIdx := startModIdx; modIdx <= endModIdx; modIdx++ {
		if cm.GetRowVisIdxFromModIdx(modIdx) != -1 {
			return modIdx
		}
	}
	return -1
}

func (cm *CoordinateManager) FindLastVisibleRowInRange(startModIdx, endModIdx int) int {
	for modIdx := endModIdx; modIdx >= startModIdx; modIdx-- {
		if cm.GetRowVisIdxFromModIdx(modIdx) != -1 {
			return modIdx
		}
	}
	return -1
}

func (cm *CoordinateManager) FindCellAtPosition(pos fyne.Position, region GridRegion) (rowModIdx, colModIdx int) {
	adjustedPos := pos

	switch region {
	case RegionMain:
		adjustedPos.X += cm.scrollOffset.X + cm.freezeColPixelEnd
		adjustedPos.Y += cm.scrollOffset.Y + cm.freezeRowPixelEnd
	case RegionFrozenRows:
		adjustedPos.X += cm.scrollOffset.X + cm.freezeColPixelEnd
	case RegionFrozenCols:
		adjustedPos.Y += cm.scrollOffset.Y + cm.freezeRowPixelEnd
	case RegionFixedCorner:
		// No adjustment needed
	}

	// Find column
	colModIdx = cm.findColAtX(adjustedPos.X)

	// Find row
	rowModIdx = cm.findRowAtY(adjustedPos.Y)

	return rowModIdx, colModIdx
}

func (cm *CoordinateManager) findColAtX(x float32) int {
	idx := sort.Search(len(cm.visColMap), func(i int) bool {
		return cm.visColMap[i].PixelEnd > x
	})

	if idx < len(cm.visColMap) && idx > 0 {
		col := cm.visColMap[idx]
		if x >= col.PixelStart && x < col.PixelEnd {
			return col.ModIdx
		}
	}

	return -1
}

func (cm *CoordinateManager) findRowAtY(y float32) int {
	idx := sort.Search(len(cm.visRowMap), func(i int) bool {
		return cm.visRowMap[i].PixelEnd > y
	})

	if idx < len(cm.visRowMap) && idx > 0 {
		row := cm.visRowMap[idx]
		if y >= row.PixelStart && y < row.PixelEnd {
			return row.ModIdx
		}
	}

	return -1
}

func (cm *CoordinateManager) FindNextVisibleColModIdxByModIdx(colModIdx int) int {
	visIdx := cm.GetColVisIdxFromModIdx(colModIdx)
	if visIdx == -1 || visIdx+1 >= len(cm.visColMap) {
		return -1
	}
	return cm.GetColModIdxFromVisIdx(visIdx + 1)
}

func (cm *CoordinateManager) FindNextVisibleRowModIdxByModIdx(rowModIdx int) int {
	visIdx := cm.GetRowVisIdxFromModIdx(rowModIdx)
	if visIdx == -1 || visIdx+1 >= len(cm.visRowMap) {
		return -1
	}
	return cm.GetRowModIdxFromVisIdx(visIdx + 1)
}
