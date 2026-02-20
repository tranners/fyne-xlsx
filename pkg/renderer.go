package pkg

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/tranners/fyne-xlsx/pkg/layouts"
)

type GridRegion int

const (
	RegionMain GridRegion = iota
	RegionFixedCorner
	RegionFrozenRows
	RegionFrozenCols
)

// String returns the string representation of the GridRegion
func (gr GridRegion) String() string {
	switch gr {
	case RegionMain:
		return "RegionMain"
	case RegionFixedCorner:
		return "RegionFixedCorner"
	case RegionFrozenRows:
		return "RegionFrozenRows"
	case RegionFrozenCols:
		return "RegionFrozenCols"
	default:
		return fmt.Sprintf("GridRegion(%d)", gr)
	}
}

type RegionContainers struct {
	Background *fyne.Container
	Gridline   *fyne.Container
	Data       *fyne.Container
	Border     *fyne.Container
	Stack      *fyne.Container
}

func NewRegionContainers() RegionContainers {
	backgrounds := container.NewWithoutLayout()
	gridlines := container.NewWithoutLayout()
	data := container.NewWithoutLayout()
	borders := container.NewWithoutLayout()
	return RegionContainers{
		Background: backgrounds,
		Gridline:   gridlines,
		Data:       data,
		Border:     borders,
		Stack:      container.NewStack(backgrounds, gridlines, data, borders),
	}
}

// GridRenderer creates context ONCE
type GridRenderer struct {
	context *RenderContext // shared resource

	fontManager *FontManager

	regionRenderer  *RegionRenderer
	headerRenderer  *HeaderRenderer
	groupRenderer   *GroupRenderer
	dividerRenderer *DividerRenderer

	selection *SelectionState

	xlContainer *fyne.Container

	regionMain, regionFrozen         RegionContainers
	regionRowFrozen, regionColFrozen RegionContainers

	scroll *container.Scroll

	colHdrContainer       *fyne.Container
	colHdrFrozenContainer *fyne.Container
	rowHdrContainer       *fyne.Container
	rowHdrFrozenContainer *fyne.Container
	cnrHdrContainer       *fyne.Container

	colGroupContainer       *fyne.Container
	colGroupFrozenContainer *fyne.Container
	rowGroupContainer       *fyne.Container
	rowGroupFrozenContainer *fyne.Container
	cnrGroupContainer       *fyne.Container

	gdRowClippedContainer       *container.Clip
	gdColClippedContainer       *container.Clip
	colHdrClippedContainer      *container.Clip
	rowHdrClippedContainer      *container.Clip
	colGroupClippedContainer    *container.Clip
	rowGroupClippedContainer    *container.Clip
	colGroupFrozenClipContainer *container.Clip
	rowGroupFrozenClipContainer *container.Clip

	freezePaneDividerContainer *fyne.Container
}

func NewGridRenderer(data *WorkSheetData, fontmgr *FontManager) *GridRenderer {

	ctx := NewRenderContext(data)
	ctx.DataChanged() // Initial build

	r := &GridRenderer{
		context:         ctx,
		fontManager:     fontmgr,
		regionRenderer:  NewRegionRenderer(ctx, fontmgr),
		headerRenderer:  NewHeaderRenderer(ctx),
		groupRenderer:   NewGroupRenderer(ctx),
		dividerRenderer: NewDividerRenderer(ctx),
		selection:       NewSelectionState(),
	}

	r.initializeContainers()
	r.setupScrollCallback()
	r.buildMainContainer()

	return r
}

func (r *GridRenderer) isValidViewport(region GridRegion) bool {
	ctx := r.context

	if ctx.PaneRenderState[region] == RenderStateNotStarted {
		vp := ctx.Viewports[region]
		isValidCurrent := vp.FirstRowVisIdx <= vp.LastRowVisIdx &&
			vp.FirstColVisIdx <= vp.LastColVisIdx &&
			vp.FirstRowVisIdx >= 0

		if !isValidCurrent {
			return false
		}
		ctx.PaneRenderState[region] = RenderStateStarting
		return true
	}
	return true
}

func (r *GridRenderer) renderOrchestrator(forceFullRender bool) {
	ctx := r.context
	cm := ctx.CoordManager

	scrollChange := ctx.ScrollOffsetChanged()

	// ============================================================
	// PHASE 1: Render Each Region Independently
	// ============================================================
	r.renderRegion(RegionMain, r.regionMain, forceFullRender, scrollChange)

	hasFrozenCols := cm.HasFrozenColumns()
	hasFrozenRows := cm.HasFrozenRows()

	// Fixed Corner (only if both frozen rows/cols exist)
	if hasFrozenCols && hasFrozenRows {
		r.renderRegion(RegionFixedCorner, r.regionFrozen, forceFullRender, scrollChange)
	} else {
		// Mark as "completed" even though it doesn't exist
		ctx.PaneHasRenderedOnce[RegionFixedCorner] = true
	}

	// Frozen Rows
	if hasFrozenRows {
		r.renderRegion(RegionFrozenRows, r.regionRowFrozen, forceFullRender, scrollChange)
	} else {
		ctx.PaneHasRenderedOnce[RegionFrozenRows] = true
	}

	// Frozen Columns
	if hasFrozenCols {
		r.renderRegion(RegionFrozenCols, r.regionColFrozen, forceFullRender, scrollChange)
	} else {
		ctx.PaneHasRenderedOnce[RegionFrozenCols] = true
	}

	// ============================================================
	// PHASE 2: Activate Optimizations Once All Regions Ready
	// ============================================================
	if ctx.SheetRenderState != RenderStateRunning {
		allRendered := true
		for _, hasRendered := range ctx.PaneHasRenderedOnce {
			if !hasRendered {
				allRendered = false
				break
			}
		}

		if allRendered {
			ctx.SheetRenderState = RenderStateRunning
			// From this point forward, viewport validation checks are bypassed
			if ctx.Data.Settings.ShowHeadings {
				// If we have Headers; do a Full Render Now
				hr := r.headerRenderer

				hr.renderFullColumnHeaders(r.colHdrContainer)

				hr.renderFullRowHeaders(r.rowHdrContainer)

				if hasFrozenCols {
					hr.renderFixedColumnHeaders(r.colHdrFrozenContainer)
				}

				if hasFrozenRows {
					hr.renderFixedRowHeaders(r.rowHdrFrozenContainer)
				}

				hr.RenderCorner(r.cnrHdrContainer)
			}
		}

	} else {
		if ctx.Data.Settings.ShowHeadings {
			hr := r.headerRenderer

			if scrollChange.X || forceFullRender {
				hr.renderColumnHeaders(r.colHdrContainer)
			}
			if scrollChange.Y || forceFullRender {
				hr.renderRowHeaders(r.rowHdrContainer)
			}

		}
		// === GROUPS ===
		hasColGroups := len(ctx.GroupManager.colGroups) > 0
		hasRowGroups := len(ctx.GroupManager.rowGroups) > 0

		if hasColGroups || hasRowGroups {
			gr := r.groupRenderer

			if hasFrozenRows && hasColGroups {
				gr.renderColGroupIndicators(r.colGroupContainer)
			}
			if hasFrozenCols && hasRowGroups {
				gr.renderRowGroupIndicators(r.rowGroupContainer)
			}

			if forceFullRender && hasColGroups && hasFrozenCols {
				gr.RenderFixedColumnGroups(r.colGroupFrozenContainer)
			}

			if forceFullRender && hasRowGroups && hasFrozenRows {
				gr.RenderFixedRowGroups(r.rowGroupFrozenContainer)
			}
		}

		dr := r.dividerRenderer
		dr.updateDividers(r.freezePaneDividerContainer)

	}
	ctx.FinalizeRenderCycle()
}

func (r *GridRenderer) Render() {
	// sould not be used
	fmt.Printf("[-RENDER] Width:%f, Height:%f\n", r.scroll.Size().Width, r.scroll.Size().Height)
	r.renderOrchestrator(true)
}

func (r *GridRenderer) initializeContainers() {
	ctx := r.context
	cm := ctx.CoordManager
	scrollableSize := cm.GetScrollableSize()

	r.regionMain = NewRegionContainers()

	// Wrap the stacked content in fixed layout and scroll (unchanged)
	fixedLayout := layouts.NewFixedSizeLayout(fyne.NewSize(scrollableSize.Width, scrollableSize.Height))
	wrappedContent := container.New(fixedLayout, r.regionMain.Stack)
	r.scroll = container.NewScroll(wrappedContent)

	r.regionFrozen = NewRegionContainers()

	r.regionRowFrozen = NewRegionContainers()
	r.gdRowClippedContainer = container.NewClip(r.regionRowFrozen.Stack)

	r.regionColFrozen = NewRegionContainers()
	r.gdColClippedContainer = container.NewClip(r.regionColFrozen.Stack)

	// Headers
	r.colHdrContainer = container.NewWithoutLayout()
	r.colHdrClippedContainer = container.NewClip(r.colHdrContainer)

	r.rowHdrContainer = container.NewWithoutLayout()
	r.rowHdrClippedContainer = container.NewClip(r.rowHdrContainer)

	r.colHdrFrozenContainer = container.NewWithoutLayout()
	r.rowHdrFrozenContainer = container.NewWithoutLayout()

	r.cnrHdrContainer = container.NewWithoutLayout()

	// Groups
	r.colGroupContainer = container.NewWithoutLayout()
	r.colGroupClippedContainer = container.NewClip(r.colGroupContainer)

	r.rowGroupContainer = container.NewWithoutLayout()
	r.rowGroupClippedContainer = container.NewClip(r.rowGroupContainer)

	r.cnrGroupContainer = container.NewWithoutLayout()

	r.colGroupFrozenContainer = container.NewWithoutLayout()
	r.colGroupFrozenClipContainer = container.NewClip(r.colGroupFrozenContainer)

	r.rowGroupFrozenContainer = container.NewWithoutLayout()
	r.rowGroupFrozenClipContainer = container.NewClip(r.rowGroupFrozenContainer)

	r.freezePaneDividerContainer = container.NewWithoutLayout()

}

func (r *GridRenderer) setupScrollCallback() {

	r.scroll.OnScrolled = func(pos fyne.Position) {
		ctx := r.context
		ctx.SetScrollOffset(pos)
		ctx.UpdateViewports(r.scroll.Size())

		r.renderOrchestrator(false)

	}
}

func (r *GridRenderer) buildMainContainer() {
	ctx := r.context

	mainApp := container.New(r,
		r.scroll,
		r.colHdrClippedContainer,
		r.rowHdrClippedContainer,
		r.cnrHdrContainer,
		r.regionFrozen.Stack,
		r.gdRowClippedContainer,
		r.gdColClippedContainer,
		r.colHdrFrozenContainer,
		r.rowHdrFrozenContainer,
		r.colGroupClippedContainer,
		r.rowGroupClippedContainer,
		r.cnrGroupContainer,
		r.colGroupFrozenClipContainer,
		r.rowGroupFrozenClipContainer,
	)

	r.xlContainer = container.NewStack(mainApp, r.freezePaneDividerContainer)

	ctx.UpdateViewports(r.scroll.Size())
}

func (r *GridRenderer) GetContainer() *fyne.Container {
	return r.xlContainer
}

func (r *GridRenderer) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	cm := r.context.CoordManager
	gm := r.context.GroupManager

	// Determine freeze pane state
	//hasRowFreeze := cm.GetVisibleFrozenRows() > 0
	//hasColFreeze := cm.GetVisibleFrozenColumns() > 0
	hasRowFreeze := cm.GetFrozenRows() > 0
	hasColFreeze := cm.GetFrozenColumns() > 0
	// Determine group header state
	hasRowGroups := len(gm.rowGroups) > 0
	hasColGroups := len(gm.colGroups) > 0

	// Constants
	const (
		IDX_SCROLL           = iota // 0: Main scrollable data
		IDX_COL_HDR_SCROLL          // 1: Column headers (scrollable)
		IDX_ROW_HDR_SCROLL          // 2: Row headers (scrollable)
		IDX_CORNER                  // 3: Corner
		IDX_DATA_FIXED              // 4: Data fixed (top-left)
		IDX_DATA_ROW_FROZEN         // 5: Data row-frozen (top-right)
		IDX_DATA_COL_FROZEN         // 6: Data col-frozen (bottom-left)
		IDX_COL_HDR_FIXED           // 7: Column headers fixed
		IDX_ROW_HDR_FIXED           // 8: Row headers fixed
		IDX_COL_GROUP_SCROLL        // 9: Col Group headers
		IDX_ROW_GROUP_SCROLL        // 10: Row Group headers
		IDX_CORNER_GROUP            // 11: Corner Group header
		IDX_COL_GROUP_FROZEN        // 12: Col Group frozen
		IDX_ROW_GROUP_FROZEN        // 13: Row Group frozen

	)

	// === CALCULATE DIMENSIONS ===

	// Base dimensions
	colHdrHeight := float32(0)
	rowHdrWidth := float32(0)
	if r.context.Data.Settings.ShowHeadings {
		colHdrHeight = HeaderHeight
		rowHdrWidth = HeaderWidth
	}

	// Group header dimensions (variable based on nesting depth)
	colGroupHeight := float32(0)
	rowGroupWidth := float32(0)
	const levelSize = float32(GroupLevelSize) // Pixels per nesting level

	if hasColGroups {
		maxColLevel := gm.GetMaxColGroupLevel()
		colGroupHeight = float32(maxColLevel)*levelSize + GroupHeaderPadding
	}

	if hasRowGroups {
		maxRowLevel := gm.GetMaxRowGroupLevel()
		rowGroupWidth = float32(maxRowLevel)*levelSize + GroupHeaderPadding
	}

	// Freeze pane dimensions
	freezeWidth := float32(0)
	freezeHeight := float32(0)
	if hasColFreeze || hasRowFreeze {
		freezeWidth, freezeHeight = cm.GetFreezeOffsets()
	}

	// Calculate total offsets
	leftOffset := rowGroupWidth + rowHdrWidth
	topOffset := colGroupHeight + colHdrHeight

	// === POSITION GROUP HEADERS ===

	// Corner Group Header (top-left corner where row/col groups meet)
	if hasRowGroups && hasColGroups {
		objects[IDX_CORNER_GROUP].Move(fyne.NewPos(0, 0))
		objects[IDX_CORNER_GROUP].Resize(fyne.NewSize(rowGroupWidth, colGroupHeight))
		objects[IDX_CORNER_GROUP].Show()
	} else {
		objects[IDX_CORNER_GROUP].Hide()
	}

	// Column Group Headers - Main (scrollable)
	if hasColGroups {
		objects[IDX_COL_GROUP_SCROLL].Move(fyne.NewPos(leftOffset+freezeWidth, 0))
		objects[IDX_COL_GROUP_SCROLL].Resize(fyne.NewSize(
			size.Width-leftOffset-freezeWidth,
			colGroupHeight,
		))
		objects[IDX_COL_GROUP_SCROLL].Show()
	} else {
		objects[IDX_COL_GROUP_SCROLL].Hide()
	}

	if hasColGroups && hasColFreeze {
		objects[IDX_COL_GROUP_FROZEN].Move(fyne.NewPos(leftOffset, 0))
		objects[IDX_COL_GROUP_FROZEN].Resize(fyne.NewSize(freezeWidth, colGroupHeight))
		objects[IDX_COL_GROUP_FROZEN].Show()
	} else {
		objects[IDX_COL_GROUP_FROZEN].Hide()
	}

	// Row Group Headers - Main (scrollable)
	if hasRowGroups {
		objects[IDX_ROW_GROUP_SCROLL].Move(fyne.NewPos(0, topOffset+freezeHeight))
		objects[IDX_ROW_GROUP_SCROLL].Resize(fyne.NewSize(
			rowGroupWidth,
			size.Height-topOffset-freezeHeight,
		))
		objects[IDX_ROW_GROUP_SCROLL].Show()
	} else {
		objects[IDX_ROW_GROUP_SCROLL].Hide()
	}

	if hasRowGroups && hasRowFreeze {
		objects[IDX_ROW_GROUP_FROZEN].Move(fyne.NewPos(0, topOffset))
		objects[IDX_ROW_GROUP_FROZEN].Resize(fyne.NewSize(rowGroupWidth, freezeHeight))
		objects[IDX_ROW_GROUP_FROZEN].Show()
	} else {
		objects[IDX_ROW_GROUP_FROZEN].Hide()
	}

	if r.context.Data.Settings.ShowHeadings {
		// Corner Header
		objects[IDX_CORNER].Move(fyne.NewPos(rowGroupWidth, colGroupHeight))
		objects[IDX_CORNER].Resize(fyne.NewSize(rowHdrWidth, colHdrHeight))
		objects[IDX_CORNER].Show()

		// Column Headers - Main (scrollable)
		objects[IDX_COL_HDR_SCROLL].Move(fyne.NewPos(leftOffset+freezeWidth, colGroupHeight))
		objects[IDX_COL_HDR_SCROLL].Resize(fyne.NewSize(
			size.Width-leftOffset-freezeWidth,
			colHdrHeight,
		))
		objects[IDX_COL_HDR_SCROLL].Show()

		// Row Headers - Main (scrollable)
		objects[IDX_ROW_HDR_SCROLL].Move(fyne.NewPos(rowGroupWidth, topOffset+freezeHeight))
		objects[IDX_ROW_HDR_SCROLL].Resize(fyne.NewSize(
			rowHdrWidth,
			size.Height-topOffset-freezeHeight,
		))
		objects[IDX_ROW_HDR_SCROLL].Show()

		// Fixed Column Headers (if column freeze)
		if hasColFreeze {
			objects[IDX_COL_HDR_FIXED].Move(fyne.NewPos(leftOffset, colGroupHeight))
			objects[IDX_COL_HDR_FIXED].Resize(fyne.NewSize(freezeWidth, colHdrHeight))
			objects[IDX_COL_HDR_FIXED].Show()
		} else {
			objects[IDX_COL_HDR_FIXED].Hide()
		}

		// Fixed Row Headers (if row freeze)
		if hasRowFreeze {
			objects[IDX_ROW_HDR_FIXED].Move(fyne.NewPos(rowGroupWidth, topOffset))
			objects[IDX_ROW_HDR_FIXED].Resize(fyne.NewSize(rowHdrWidth, freezeHeight))
			objects[IDX_ROW_HDR_FIXED].Show()
		} else {
			objects[IDX_ROW_HDR_FIXED].Hide()
		}
	} else {
		// Hide all headers when not showing
		objects[IDX_CORNER].Hide()
		objects[IDX_COL_HDR_SCROLL].Hide()
		objects[IDX_ROW_HDR_SCROLL].Hide()
		objects[IDX_COL_HDR_FIXED].Hide()
		objects[IDX_ROW_HDR_FIXED].Hide()
	}

	// === POSITION DATA REGIONS (adjusted for group headers) ===

	// Main Scrollable Data
	objects[IDX_SCROLL].Move(fyne.NewPos(leftOffset+freezeWidth, topOffset+freezeHeight))
	objects[IDX_SCROLL].Resize(fyne.NewSize(
		size.Width-leftOffset-freezeWidth,
		size.Height-topOffset-freezeHeight,
	))
	objects[IDX_SCROLL].Show()

	// Fixed Corner Data (top-left) - only when BOTH row AND column are frozen
	if hasRowFreeze && hasColFreeze {
		objects[IDX_DATA_FIXED].Move(fyne.NewPos(leftOffset, topOffset))
		objects[IDX_DATA_FIXED].Resize(fyne.NewSize(freezeWidth, freezeHeight))
		objects[IDX_DATA_FIXED].Show()
	} else {
		objects[IDX_DATA_FIXED].Hide()
	}

	// Row-Frozen Data (top-right) - when rows are frozen
	if hasRowFreeze {
		objects[IDX_DATA_ROW_FROZEN].Move(fyne.NewPos(leftOffset+freezeWidth, topOffset))
		objects[IDX_DATA_ROW_FROZEN].Resize(fyne.NewSize(
			size.Width-leftOffset-freezeWidth,
			freezeHeight,
		))
		objects[IDX_DATA_ROW_FROZEN].Show()
	} else {
		objects[IDX_DATA_ROW_FROZEN].Hide()
	}

	// Col-Frozen Data (bottom-left) - when columns are frozen
	if hasColFreeze {
		objects[IDX_DATA_COL_FROZEN].Move(fyne.NewPos(leftOffset, topOffset+freezeHeight))
		objects[IDX_DATA_COL_FROZEN].Resize(fyne.NewSize(
			freezeWidth,
			size.Height-topOffset-freezeHeight,
		))
		objects[IDX_DATA_COL_FROZEN].Show()
	} else {
		objects[IDX_DATA_COL_FROZEN].Hide()
	}
}

func (r *GridRenderer) UpdateScrollContentSize() {
	r.updateScrollContentSize()
}

func (r *GridRenderer) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(250, 500)
}

func (r *GridRenderer) OnResize(newSize fyne.Size) {
	ctx := r.context

	ctx.UpdateViewports(r.scroll.Size())

	//fmt.Printf("[SCROLL-RESIZE] Width:%f, Height:%f\n", r.scroll.Size().Width, r.scroll.Size().Height)
	r.renderOrchestrator(true)
}

func (r *GridRenderer) updateScrollContentSize() {
	ctx := r.context
	cm := ctx.CoordManager

	scrollabelSize := cm.GetScrollableSize()

	viewportSize := r.scroll.Size()

	if viewportSize.Width > 0 && viewportSize.Height > 0 {
		maxScrollWidth := scrollabelSize.Width + viewportSize.Width
		maxScrollHeight := scrollabelSize.Height + viewportSize.Height

		wrappedContent := r.scroll.Content.(*fyne.Container)

		newLayout := layouts.NewFixedSizeLayout(fyne.NewSize(maxScrollWidth, maxScrollHeight))

		wrappedContent.Layout = newLayout
		wrappedContent.Refresh()
	}
}
