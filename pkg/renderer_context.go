// pkg/render_context.go
package pkg

import (
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

type RenderState int

const (
	RenderStateNotStarted RenderState = iota
	RenderStateStarting
	RenderStateRunning
)

const RegionCount = 4

type RenderContext struct {
	CoordManager *CoordinateManager
	MergeManager *MergeManager
	GroupManager *GroupManager

	Data *WorkSheetData

	GroupPool    *sync.Pool
	GridLinePool *sync.Pool

	Viewports     [RegionCount]Viewport
	LastViewports [RegionCount]Viewport

	ShowGridlines bool
	ShowHeadings  bool

	SheetRenderState    RenderState
	PaneRenderState     [RegionCount]RenderState
	PaneHasRenderedOnce [RegionCount]bool
}

func NewRenderContext(data *WorkSheetData) *RenderContext {
	ctx := &RenderContext{
		Data:                data,
		CoordManager:        NewCoordinateManager(),
		GroupManager:        NewGroupManager(),
		ShowGridlines:       data.Settings.ShowGridlines,
		ShowHeadings:        data.Settings.ShowHeadings,
		SheetRenderState:    RenderStateNotStarted,
		PaneHasRenderedOnce: [4]bool{false, false, false, false},
		PaneRenderState: [4]RenderState{
			RenderStateNotStarted,
			RenderStateNotStarted,
			RenderStateNotStarted,
			RenderStateNotStarted},
	}

	ctx.MergeManager = NewMergeManager(ctx.CoordManager, ctx.Data)

	ctx.GroupPool = &sync.Pool{
		New: func() interface{} {
			extentLine := canvas.NewLine(color.NRGBA{R: 100, G: 100, B: 100, A: 255})
			extentLine.StrokeWidth = 1
			tailLine := canvas.NewLine(color.NRGBA{R: 100, G: 100, B: 100, A: 255})
			tailLine.StrokeWidth = 1
			boxBg := canvas.NewRectangle(color.White)
			boxBorder := canvas.NewRectangle(color.Transparent)
			boxBorder.StrokeColor = color.Black
			boxBorder.StrokeWidth = 1
			iconH := canvas.NewLine(color.Black)
			iconH.StrokeWidth = 2
			iconV := canvas.NewLine(color.Black)
			iconV.StrokeWidth = 2
			wrapper := container.NewWithoutLayout(extentLine, tailLine, boxBg, boxBorder, iconH, iconV)

			return &GroupCarcuss{
				Container:  wrapper,
				ExtentLine: extentLine,
				TailLine:   tailLine,
				BoxBg:      boxBg,
				BoxBorder:  boxBorder,
				IconHLine:  iconH,
				IconVLine:  iconV,
			}
		},
	}
	return ctx
}

func (ctx *RenderContext) UpdateViewports(scrollSize fyne.Size) {
	cm := ctx.CoordManager
	ctx.Viewports = cm.CalculateViewports(scrollSize)
}

func (ctx *RenderContext) FinalizeRenderCycle() {
	copy(ctx.LastViewports[:], ctx.Viewports[:])
}

func (ctx *RenderContext) SetScrollOffset(offset fyne.Position) {
	cm := ctx.CoordManager
	cm.SetScrollOffset(offset)
}

func (ctx *RenderContext) DataChanged() {
	cm := ctx.CoordManager
	gm := ctx.GroupManager
	mm := ctx.MergeManager

	cm.rebuildColLayout(ctx.Data)
	cm.rebuildRowLayout(ctx.Data)

	gm.buildColGroupsFromOutlineLevels(
		ctx.Data.ColGroupOutlineLevels,
		ctx.Data.HiddenColumns,
	)
	gm.buildRowGroupsFromOutlineLevels(
		ctx.Data.RowGroupOutlineLevels,
		ctx.Data.HiddenRows,
	)
	gm.buildColGroupsVisibleState(ctx.CoordManager)
	gm.buildRowGroupsVisibleState(ctx.CoordManager)

	mm.buildMergeLookup(ctx.Data.MergeCells)

}

type ScrollChange struct {
	X bool
	Y bool
}

func (ctx *RenderContext) ScrollOffsetChanged() ScrollChange {
	cm := ctx.CoordManager
	prevOffset := cm.prevScrollOffset
	currOffset := cm.scrollOffset
	return ScrollChange{
		X: prevOffset.X != currOffset.X,
		Y: prevOffset.Y != currOffset.Y,
	}
}
