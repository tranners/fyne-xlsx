package pkg

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

// Freeze pane divider styling (similar to Excel)
var freezeDividerColor = color.NRGBA{R: 135, G: 135, B: 135, A: 255} // Dark gray
const freezeDividerWidth = float32(1)                                // Thicker than cell borders

type DividerCarcuss struct {
	HDivider *canvas.Line
	VDivider *canvas.Line
}

type DividerRenderer struct {
	ctx     *RenderContext
	divider DividerCarcuss
}

func NewDividerRenderer(ctx *RenderContext) *DividerRenderer {
	vDivider := canvas.NewLine(freezeDividerColor)
	vDivider.StrokeWidth = freezeDividerWidth
	hDivider := canvas.NewLine(freezeDividerColor)
	hDivider.StrokeWidth = freezeDividerWidth
	return &DividerRenderer{
		ctx: ctx,
		divider: DividerCarcuss{
			HDivider: vDivider,
			VDivider: hDivider,
		},
	}
}

func (dr *DividerRenderer) updateDividers(dividerContainer *fyne.Container) {
	ctx := dr.ctx
	cm := ctx.CoordManager
	gm := ctx.GroupManager

	dividerContainer.Objects = nil

	freezeWidth, freezeHeight := cm.GetFreezeOffsets()

	if freezeWidth == 0 && freezeHeight == 0 {
		return
	}

	colHdrHeight := float32(0)
	rowHdrWidth := float32(0)
	if ctx.Data.Settings.ShowHeadings {
		colHdrHeight = HeaderHeight
		rowHdrWidth = HeaderWidth
	}

	colGroupHeight := float32(0)
	rowGroupWidth := float32(0)
	if len(gm.colGroups) > 0 {
		maxColLevel := gm.GetMaxColGroupLevel()
		colGroupHeight = float32(maxColLevel)*GroupLevelSize + GroupHeaderPadding
	}
	if len(gm.rowGroups) > 0 {
		maxRowLevel := gm.GetMaxRowGroupLevel()
		rowGroupWidth = float32(maxRowLevel)*GroupLevelSize + GroupHeaderPadding
	}

	leftOffset := rowGroupWidth + rowHdrWidth
	topOffset := colGroupHeight + colHdrHeight

	// Vertical divider (for column freeze)
	if freezeWidth > 0 {
		x := leftOffset + freezeWidth
		dr.divider.VDivider.Position1 = fyne.NewPos(x, topOffset)
		dr.divider.VDivider.Position2 = fyne.NewPos(x, dividerContainer.Size().Height)
		dividerContainer.Add(dr.divider.VDivider)
	}

	// Horizontal divider (for row freeze)
	if freezeHeight > 0 {
		y := topOffset + freezeHeight
		dr.divider.HDivider.Position1 = fyne.NewPos(leftOffset, y)
		dr.divider.HDivider.Position2 = fyne.NewPos(dividerContainer.Size().Width, y)
		dividerContainer.Add(dr.divider.HDivider)
	}
}
