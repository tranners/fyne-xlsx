package pkg

// pkg/renderer.go

func (r *GridRenderer) ClearSelectionHighlights(rowIdx, colIdx int) {
	r.headerRenderer.ResetHeaderBackground(rowIdx, colIdx)
}

func (r *GridRenderer) ApplySelectionHighlights(rowIdx, colIdx int) {
	r.headerRenderer.HighlightHeaders(rowIdx, colIdx)
}
