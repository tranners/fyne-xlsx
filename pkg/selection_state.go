package pkg

import "image/color"

type SelectionState struct {
	SelectedCell *CellID
	highlightRow int
	highlightCol int
	hasSelection bool
}

func NewSelectionState() *SelectionState {
	return &SelectionState{}
}

var (
	SelectedBorderColor  = color.NRGBA{R: 0, G: 120, B: 215, A: 255} // Excel-like blue
	SelectedBorderWidth  = float32(2.0)
	HeaderHighlightColor = color.NRGBA{R: 220, G: 220, B: 220, A: 255} // Slightly darker
)

func (ss *SelectionState) SetSelection(rowIdx, colIdx int) {
	ss.SelectedCell = &CellID{Row: rowIdx, Col: colIdx}
	ss.highlightRow = rowIdx
	ss.highlightCol = colIdx
	ss.hasSelection = true
}

func (ss *SelectionState) ClearSelection() {
	ss.SelectedCell = nil
	ss.hasSelection = false
}

func (ss *SelectionState) HasSelection() bool {
	return ss.hasSelection
}

func (ss *SelectionState) GetHighlightedRowCol() (int, int, bool) {
	if !ss.hasSelection {
		return -1, -1, false
	}
	return ss.highlightRow, ss.highlightCol, true
}
