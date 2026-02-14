package pkg

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

type GroupCarcuss struct {
	Container *fyne.Container

	ExtentLine *canvas.Line // Long horizontal/vertical span line
	TailLine   *canvas.Line // Short perpendicular connector

	BoxBg     *canvas.Rectangle // Box background
	BoxBorder *canvas.Rectangle // Box border (can use StrokeColor)

	// +/- symbol inside box
	IconHLine *canvas.Line
	IconVLine *canvas.Line

	InitialRow int
	InitialCol int
}

type GroupRenderer struct {
	ctx             *RenderContext
	colGroupsScroll map[int]*GroupCarcuss
	colGroupsFixed  map[int]*GroupCarcuss
	rowGroupsScroll map[int]*GroupCarcuss
	rowGroupsFixed  map[int]*GroupCarcuss
}

func NewGroupRenderer(ctx *RenderContext) *GroupRenderer {
	return &GroupRenderer{
		ctx:             ctx,
		colGroupsScroll: make(map[int]*GroupCarcuss),
		colGroupsFixed:  make(map[int]*GroupCarcuss),
		rowGroupsScroll: make(map[int]*GroupCarcuss),
		rowGroupsFixed:  make(map[int]*GroupCarcuss),
	}
}

//func (gr *GroupRenderer) RenderScrollable(colGroupContainer, rowGroupContainer *fyne.Container) {
//	gr.renderColGroupIndicators(colGroupContainer)
//gr.renderRowGroupIndicators(rowGroupContainer)
//}

func (gr *GroupRenderer) RenderFixedColumnGroups(colGroupFrozenContainer *fyne.Container) {
	gr.duplicateToFrozenColumn(colGroupFrozenContainer)
}

func (gr *GroupRenderer) RenderFixedRowGroups(rowGroupFrozenContainer *fyne.Container) {
	gr.duplicateToFrozenRow(rowGroupFrozenContainer)
}

func (gr *GroupRenderer) Render(colGroupContainer, rowGroupContainer *fyne.Container) {
	gr.renderColGroupIndicators(colGroupContainer)
	gr.renderRowGroupIndicators(rowGroupContainer)
}
func (gr *GroupRenderer) RenderFixed(colGroupFrozenContainer, rowGroupFrozenContainer *fyne.Container) {
	gr.duplicateToFrozenColumn(colGroupFrozenContainer)
	gr.duplicateToFrozenColumn(rowGroupFrozenContainer)
}

func (gr *GroupRenderer) renderColGroupIndicators(colGroupContainer *fyne.Container) {
	ctx := gr.ctx
	cm := ctx.CoordManager
	gm := ctx.GroupManager

	for id, state := range gm.colGroupState {
		group := gm.colGroupIndex[id]
		if _, exists := gr.colGroupsScroll[id]; !exists {
			carcuss := ctx.GroupPool.Get().(*GroupCarcuss)
			gr.configIndicatorForColGroup(carcuss, group, state)
			intialCol := 0
			switch state {
			case IND_NORMAL, IND_VISIBLE_RANGE:
				intialCol = cm.FindFirstVisibleColInRange(
					group.ModIdxStart,
					group.ModIdxEnd,
				)
			case IND_VISIBLE_CONTROL:
				intialCol = gm.colGroupIndex[id].ControlCol
			}
			carcuss.InitialCol = intialCol
			colGroupContainer.Add(carcuss.Container)
			gr.colGroupsScroll[id] = carcuss
		}
	}

	for id, carcuss := range gr.colGroupsScroll {
		group := gm.colGroupIndex[id]
		y := float32(group.Level*GroupLevelSize) + GroupLevelOffset
		switch gm.colGroupState[id] {
		case IND_NORMAL, IND_VISIBLE_RANGE:
			x := cm.GetColPixelPosX(RegionFrozenRows, carcuss.InitialCol)
			carcuss.Container.Move(fyne.NewPos(x, y))
		case IND_VISIBLE_CONTROL:
			x := cm.GetColPixelPosX(RegionFrozenRows, carcuss.InitialCol)
			carcuss.Container.Move(fyne.NewPos(x, y))
			controlWidth := cm.GetWidthByModIdx(carcuss.InitialCol)
			carcuss.Container.Resize(fyne.NewSize(controlWidth, 16))
		default:
			carcuss.Container.Hide()
		}
	}
}

func (gr *GroupRenderer) configIndicatorForColGroup(carcess *GroupCarcuss, group *ColGroup, indicatorState GroupIndicatorType) {

	ctx := gr.ctx
	cm := ctx.CoordManager

	const boxSize = float32(GroupBoxSize)
	const tailLength = float32(GroupTailLength)
	const iconSize = float32(GroupIconSize)
	const centreY = boxSize / 2

	var endVisIdx int
	var startX float32
	var endX float32

	if indicatorState == IND_NORMAL || indicatorState == IND_VISIBLE_RANGE {
		firstVisibleCol := cm.FindFirstVisibleColInRange(
			group.ModIdxStart,
			group.ModIdxEnd,
		)
		startVisIdx := cm.GetColVisIdxFromModIdx(firstVisibleCol)
		startX = cm.GetColPixelStartByVisIdx(startVisIdx)

		lastVisibleCol := cm.FindLastVisibleColInRange(
			group.ModIdxStart,
			group.ModIdxEnd,
		)
		endVisIdx = cm.GetColVisIdxFromModIdx(lastVisibleCol)
		startX = cm.GetColPixelStartByVisIdx(startVisIdx)
	}

	switch indicatorState {
	case IND_NORMAL:
		carcess.ExtentLine.Position1 = fyne.NewPos(0, 7)
		endX = cm.GetColPixelEndByVisIdx(endVisIdx) +
			(cm.GetWidthByVisIdx(endVisIdx+1) / 2) + 8
		carcess.ExtentLine.Position2 = fyne.NewPos(endX-startX-16, 7)

		boxX := endX - startX - 16
		carcess.BoxBg.Move(fyne.NewPos(boxX, 0))
		carcess.BoxBg.Resize(fyne.NewSize(boxSize, boxSize))
		carcess.BoxBorder.Move(fyne.NewPos(boxX, 0))
		carcess.BoxBorder.Resize(fyne.NewSize(boxSize, boxSize))

		centreX := boxX + boxSize/2
		carcess.IconHLine.Position1 = fyne.NewPos(centreX-iconSize/2, centreY)
		carcess.IconHLine.Position2 = fyne.NewPos(centreX+iconSize/2, centreY)
		carcess.IconVLine.Position1 = fyne.NewPos(centreX, centreY-iconSize/2)
		carcess.IconVLine.Position2 = fyne.NewPos(centreX, centreY+iconSize/2)

		carcess.TailLine.Position1 = fyne.NewPos(0, 7)
		carcess.TailLine.Position2 = fyne.NewPos(0, 10)
		carcess.TailLine.Show()

		carcess.IconHLine.Show()
		carcess.IconHLine.Show()
		carcess.IconVLine.Hide()
	case IND_VISIBLE_RANGE:
		carcess.ExtentLine.Position1 = fyne.NewPos(0, 7)
		endX = cm.GetColPixelEndByVisIdx(endVisIdx)
		carcess.ExtentLine.Position2 = fyne.NewPos(endX-startX, 7)
		carcess.ExtentLine.Show()

		carcess.BoxBg.Hide()
		carcess.BoxBorder.Hide()
		carcess.IconHLine.Hide()
		carcess.IconVLine.Hide()
	case IND_VISIBLE_CONTROL:
		idx := cm.GetColVisIdxFromModIdx(group.ControlCol)
		controlWidth := cm.GetWidthByVisIdx(idx)

		boxX := (controlWidth / 2) - (boxSize / 2)
		carcess.BoxBg.Move(fyne.NewPos(boxX, 0))
		carcess.BoxBg.Resize(fyne.NewSize(boxSize, boxSize))
		carcess.BoxBorder.Move(fyne.NewPos(boxX, 0))
		carcess.BoxBorder.Resize(fyne.NewSize(boxSize, boxSize))
		carcess.BoxBg.Show()
		carcess.BoxBorder.Show()

		centreX := controlWidth / 2
		carcess.IconHLine.Position1 = fyne.NewPos(centreX-iconSize/2, centreY)
		carcess.IconHLine.Position2 = fyne.NewPos(centreX+iconSize/2, centreY)
		carcess.IconVLine.Position1 = fyne.NewPos(centreX, centreY-iconSize/2)
		carcess.IconVLine.Position2 = fyne.NewPos(centreX, centreY+iconSize/2)
		carcess.IconHLine.Show()
		carcess.IconVLine.Show()

		carcess.ExtentLine.Hide()
		carcess.TailLine.Hide()
	}
}

func (gr *GroupRenderer) renderRowGroupIndicators(rowGroupContainer *fyne.Container) {
	ctx := gr.ctx
	cm := ctx.CoordManager
	gm := ctx.GroupManager

	for id, state := range gm.rowGroupState {
		indicator := gm.rowGroupIndex[id]
		if _, exists := gr.rowGroupsScroll[id]; !exists {
			carcuss := ctx.GroupPool.Get().(*GroupCarcuss)
			gr.configIndicatorForRowGroup(carcuss, indicator, state)
			intialRow := 0
			switch state {
			case IND_NORMAL, IND_VISIBLE_RANGE:
				intialRow = cm.FindFirstVisibleRowInRange(
					indicator.ModIdxStart,
					indicator.ModIdxEnd,
				)
			case IND_VISIBLE_CONTROL:
				intialRow = gm.rowGroupIndex[id].ControlRow
			}
			carcuss.InitialRow = intialRow

			rowGroupContainer.Add(carcuss.Container)
			gr.rowGroupsScroll[id] = carcuss
		}
	}

	for id, carcuss := range gr.rowGroupsScroll {
		group := gm.rowGroupIndex[id]
		level := group.Level
		x := float32(level*GroupLevelSize) + GroupLevelOffset
		switch gm.rowGroupState[id] {
		case IND_NORMAL, IND_VISIBLE_RANGE:
			y := cm.GetRowPixelPosY(RegionFrozenCols, carcuss.InitialRow)
			carcuss.Container.Move(fyne.NewPos(x, y))
		case IND_VISIBLE_CONTROL:
			y := cm.GetRowPixelPosY(RegionFrozenCols, carcuss.InitialRow)
			carcuss.Container.Move(fyne.NewPos(x, y))
			controlHeight := cm.GetHeightByModIdx(carcuss.InitialRow)
			carcuss.Container.Resize(fyne.NewSize(16, controlHeight))
		default:
			carcuss.Container.Hide()
		}
	}
}

func (gr *GroupRenderer) configIndicatorForRowGroup(carcess *GroupCarcuss, indicator *RowGroup, indicatorState GroupIndicatorType) {

	ctx := gr.ctx
	cm := ctx.CoordManager

	const boxSize = float32(16)
	const tailLength = float32(4)
	const iconSize = float32(8)
	const centreX = boxSize / 2

	var endVisIdx int
	var startY float32
	var endY float32

	if indicatorState == IND_NORMAL || indicatorState == IND_VISIBLE_RANGE {
		firstVisibleRow := cm.FindFirstVisibleRowInRange(
			indicator.ModIdxStart,
			indicator.ModIdxEnd,
		)
		startVisIdx := cm.GetRowVisIdxFromModIdx(firstVisibleRow)
		startY = cm.GetRowPixelStartByVisIdx(startVisIdx)

		lastVisibleRow := cm.FindLastVisibleRowInRange(
			indicator.ModIdxStart,
			indicator.ModIdxEnd,
		)
		endVisIdx = cm.GetRowVisIdxFromModIdx(lastVisibleRow)
	}

	switch indicatorState {
	case IND_NORMAL:
		carcess.ExtentLine.Position1 = fyne.NewPos(7, 0)

		endY = cm.GetRowPixelEndByVisIdx(endVisIdx) +
			(cm.GetHeightByVisIdx(endVisIdx+1) / 2) + 8
		carcess.ExtentLine.Position2 = fyne.NewPos(7, endY-startY-16)

		boxY := endY - startY - 16
		carcess.BoxBg.Move(fyne.NewPos(0, boxY))
		carcess.BoxBg.Resize(fyne.NewSize(boxSize, boxSize))
		carcess.BoxBorder.Move(fyne.NewPos(0, boxY))
		carcess.BoxBorder.Resize(fyne.NewSize(boxSize, boxSize))

		centreY := endY - startY - boxSize/2
		carcess.IconHLine.Position1 = fyne.NewPos(centreX-iconSize/2, centreY)
		carcess.IconHLine.Position2 = fyne.NewPos(centreX+iconSize/2, centreY)
		carcess.IconVLine.Position1 = fyne.NewPos(centreX, centreY-iconSize/2)
		carcess.IconVLine.Position2 = fyne.NewPos(centreX, centreY+iconSize/2)

		carcess.TailLine.Position1 = fyne.NewPos(7, 0)
		carcess.TailLine.Position2 = fyne.NewPos(10, 0)
		carcess.TailLine.Show()

		carcess.IconHLine.Show()
		carcess.IconHLine.Show()
		carcess.IconVLine.Hide()
	case IND_VISIBLE_RANGE:
		carcess.ExtentLine.Position1 = fyne.NewPos(7, 0)
		//endY = cm.visibleRowMap[endVisIdx].PixelEnd
		endY = cm.GetRowPixelEndByVisIdx(endVisIdx)
		carcess.ExtentLine.Position2 = fyne.NewPos(7, endY-startY)
		carcess.ExtentLine.Show()

		carcess.BoxBg.Hide()
		carcess.BoxBorder.Hide()
		carcess.IconHLine.Hide()
		carcess.IconVLine.Hide()
	case IND_VISIBLE_CONTROL:
		idx := cm.GetRowVisIdxFromModIdx(indicator.ControlRow)
		controlHeight := cm.GetHeightByVisIdx(idx)

		boxY := (controlHeight / 2) - (boxSize / 2)
		carcess.BoxBg.Move(fyne.NewPos(0, boxY))
		carcess.BoxBg.Resize(fyne.NewSize(boxSize, boxSize))
		carcess.BoxBorder.Move(fyne.NewPos(0, boxY))
		carcess.BoxBorder.Resize(fyne.NewSize(boxSize, boxSize))
		carcess.BoxBg.Show()
		carcess.BoxBorder.Show()

		centreY := controlHeight / 2
		carcess.IconHLine.Position1 = fyne.NewPos(centreX-iconSize/2, centreY)
		carcess.IconHLine.Position2 = fyne.NewPos(centreX+iconSize/2, centreY)
		carcess.IconVLine.Position1 = fyne.NewPos(centreX, centreY-iconSize/2)
		carcess.IconVLine.Position2 = fyne.NewPos(centreX, centreY+iconSize/2)
		carcess.IconHLine.Show()
		carcess.IconVLine.Show()

		carcess.ExtentLine.Hide()
		carcess.TailLine.Hide()

	}
}

func (gr *GroupRenderer) duplicateToFrozenColumn(colGroupFrozenContainer *fyne.Container) {
	ctx := gr.ctx
	cm := ctx.CoordManager
	gm := ctx.GroupManager

	freezeColSplit := cm.freezeColSplit

	for id, scrollIndicator := range gr.colGroupsScroll {
		group := gm.colGroupIndex[id]
		if group.ModIdxStart >= freezeColSplit {
			continue
		}

		var grpIndicator *GroupCarcuss
		if existing, exists := gr.colGroupsFixed[id]; exists {
			grpIndicator = existing
		} else {
			grpIndicator = ctx.GroupPool.Get().(*GroupCarcuss)
			colGroupFrozenContainer.Add(grpIndicator.Container)
			gr.colGroupsFixed[id] = grpIndicator
		}
		gr.copyIndicatorConfiguration(grpIndicator, scrollIndicator)

		y := float32(group.Level*GroupLevelSize) + GroupLevelOffset
		switch gm.colGroupState[id] {
		case IND_NORMAL, IND_VISIBLE_RANGE:
			firstVisibleCol := cm.FindFirstVisibleColInRange(
				group.ModIdxStart,
				group.ModIdxEnd,
			)
			x := cm.GetColPixelPosX(RegionFixedCorner, firstVisibleCol)
			grpIndicator.Container.Move(fyne.NewPos(x, y))
		case IND_VISIBLE_CONTROL:
			controlCol := gm.colGroupIndex[id].ControlCol
			x := cm.GetColPixelPosX(RegionFixedCorner, controlCol)
			grpIndicator.Container.Move(fyne.NewPos(x, y))
			controlWidth := cm.GetWidthByModIdx(controlCol)
			grpIndicator.Container.Resize(fyne.NewSize(controlWidth, 16))
		default:
			grpIndicator.Container.Hide()
		}
	}
}

func (gr *GroupRenderer) duplicateToFrozenRow(rowGroupFrozenContainer *fyne.Container) {
	ctx := gr.ctx
	cm := ctx.CoordManager
	gm := ctx.GroupManager

	freezeRowSplit := cm.freezeRowSplit

	for id, scrollIndicator := range gr.rowGroupsScroll {
		group := gm.rowGroupIndex[id]
		if group.ModIdxStart >= freezeRowSplit {
			continue
		}

		var frozenIndicator *GroupCarcuss
		if existing, exists := gr.rowGroupsFixed[id]; exists {
			frozenIndicator = existing
		} else {
			frozenIndicator = ctx.GroupPool.Get().(*GroupCarcuss)
			rowGroupFrozenContainer.Add(frozenIndicator.Container)
			gr.rowGroupsFixed[id] = frozenIndicator
		}
		gr.copyIndicatorConfiguration(frozenIndicator, scrollIndicator)

		x := float32(group.Level*GroupLevelSize) + GroupLevelOffset
		switch gm.rowGroupState[id] {
		case IND_NORMAL, IND_VISIBLE_RANGE:
			firstVisibleRow := cm.FindFirstVisibleRowInRange(
				group.ModIdxStart,
				group.ModIdxEnd,
			)
			y := cm.GetRowPixelPosY(RegionFixedCorner, firstVisibleRow)
			frozenIndicator.Container.Move(fyne.NewPos(x, y))
			frozenIndicator.Container.Show()
		case IND_VISIBLE_CONTROL:
			controlRow := gm.rowGroupIndex[id].ControlRow
			y := cm.GetRowPixelPosY(RegionFixedCorner, controlRow)
			frozenIndicator.Container.Move(fyne.NewPos(x, y))
			controlHeight := cm.GetHeightByModIdx(controlRow)
			frozenIndicator.Container.Resize(fyne.NewSize(16, controlHeight))
			frozenIndicator.Container.Show()
		default:
			frozenIndicator.Container.Hide()
		}
	}
}

func (gr *GroupRenderer) copyIndicatorConfiguration(dest, src *GroupCarcuss) {
	dest.ExtentLine.Position1 = src.ExtentLine.Position1
	dest.ExtentLine.Position2 = src.ExtentLine.Position2
	dest.ExtentLine.StrokeWidth = src.ExtentLine.StrokeWidth
	dest.ExtentLine.StrokeColor = src.ExtentLine.StrokeColor
	if src.ExtentLine.Visible() {
		dest.ExtentLine.Show()
	} else {
		dest.ExtentLine.Hide()
	}

	dest.TailLine.Position1 = src.TailLine.Position1
	dest.TailLine.Position2 = src.TailLine.Position2
	dest.TailLine.StrokeWidth = src.TailLine.StrokeWidth
	dest.TailLine.StrokeColor = src.TailLine.StrokeColor
	if src.TailLine.Visible() {
		dest.TailLine.Show()
	} else {
		dest.TailLine.Hide()
	}

	dest.BoxBg.Move(src.BoxBg.Position())
	dest.BoxBg.Resize(src.BoxBg.Size())
	dest.BoxBg.FillColor = src.BoxBg.FillColor
	if src.BoxBg.Visible() {
		dest.BoxBg.Show()
	} else {
		dest.BoxBg.Hide()
	}

	dest.BoxBorder.Move(src.BoxBorder.Position())
	dest.BoxBorder.Resize(src.BoxBorder.Size())
	dest.BoxBorder.StrokeColor = src.BoxBorder.StrokeColor
	dest.BoxBorder.StrokeWidth = src.BoxBorder.StrokeWidth
	if src.BoxBorder.Visible() {
		dest.BoxBorder.Show()
	} else {
		dest.BoxBorder.Hide()
	}

	dest.IconHLine.Position1 = src.IconHLine.Position1
	dest.IconHLine.Position2 = src.IconHLine.Position2
	dest.IconHLine.StrokeWidth = src.IconHLine.StrokeWidth
	dest.IconHLine.StrokeColor = src.IconHLine.StrokeColor
	if src.IconHLine.Visible() {
		dest.IconHLine.Show()
	} else {
		dest.IconHLine.Hide()
	}

	dest.IconVLine.Position1 = src.IconVLine.Position1
	dest.IconVLine.Position2 = src.IconVLine.Position2
	dest.IconVLine.StrokeWidth = src.IconVLine.StrokeWidth
	dest.IconVLine.StrokeColor = src.IconVLine.StrokeColor
	if src.IconVLine.Visible() {
		dest.IconVLine.Show()
	} else {
		dest.IconVLine.Hide()
	}

	//dest.Container.Refresh()
}
