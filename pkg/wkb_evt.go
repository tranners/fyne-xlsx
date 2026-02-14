package pkg

import "fyne.io/fyne/v2"

// Implement fyne.Tappable interface
func (w *WorkbookWidget) Tapped(pe *fyne.PointEvent) {
	if w.currentRenderer == nil {
		return
	}
	w.handleTap(pe.Position)
}

func (w *WorkbookWidget) handleTap(pos fyne.Position) {
	renderer := w.currentRenderer
	cm := renderer.context.CoordManager

	// Adjust position to account for headers/groups if needed
	adjustedPos := w.adjustPositionForHeaders(pos)

	// Find which cell was tapped
	rowIdx, colIdx := cm.FindCellAtPosition(adjustedPos, RegionMain)

	if rowIdx < 0 || colIdx < 0 {
		// Tapped outside grid area (maybe on headers or empty space)
		return
	}

	// Clear previous selection
	if w.selection.HasSelection() {
		oldRow, oldCol, _ := w.selection.GetHighlightedRowCol()
		renderer.ClearSelectionHighlights(oldRow, oldCol)
	}

	// Set new selection
	w.selection.SetSelection(rowIdx, colIdx)

	// Apply highlights
	renderer.ApplySelectionHighlights(rowIdx, colIdx)
}

func (w *WorkbookWidget) adjustPositionForHeaders(pos fyne.Position) fyne.Position {
	// Account for header and group dimensions
	renderer := w.currentRenderer

	cm := renderer.context.CoordManager
	gm := renderer.context.GroupManager

	// Calculate offsets
	headerOffset := fyne.Position{}

	if renderer.context.Data.Settings.ShowHeadings {
		headerOffset.X = HeaderWidth
		headerOffset.Y = HeaderHeight
	}

	// Add group offsets if present
	if len(gm.rowGroups) > 0 {
		maxLevel := gm.GetMaxRowGroupLevel()
		headerOffset.X += float32(maxLevel)*GroupLevelSize + GroupHeaderPadding
	}

	if len(gm.colGroups) > 0 {
		maxLevel := gm.GetMaxColGroupLevel()
		headerOffset.Y += float32(maxLevel)*GroupLevelSize + GroupHeaderPadding
	}

	// Also account for frozen panes
	freezeColOffset, freezeRowOffset := cm.GetFreezeOffsets()

	// Create adjusted position
	adjusted := fyne.Position{
		X: pos.X - headerOffset.X - freezeColOffset,
		Y: pos.Y - headerOffset.Y - freezeRowOffset,
	}

	return adjusted
}
