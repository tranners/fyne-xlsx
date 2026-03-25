package pkg

import "image/color"

var headerHighlightBackgroundColor = color.NRGBA{R: 220, G: 220, B: 220, A: 255} // Slightly darker
var headerHighLightBorderColor = color.NRGBA{R: 0, G: 0, B: 0, A: 255}           // Slightly darker

func (hr *HeaderRenderer) HighlightHeaders(rowIdx, colIdx int) {
	//hr.setColHeaderBackground(hr.colHeadersScroll, colIdx, headerHighlightBackgroundColor, headerHighLightBorderColor, 2)
	//hr.setColHeaderBackground(hr.colHeadersFixed, colIdx, headerHighlightBackgroundColor, headerHighLightBorderColor, 2)

	//hr.setRowHeaderBackground(hr.rowHeadersScroll, rowIdx, headerHighlightBackgroundColor, headerHighLightBorderColor, 2)
	//hr.setRowHeaderBackground(hr.rowHeadersFixed, rowIdx, headerHighlightBackgroundColor, headerHighLightBorderColor, 2)
}

func (hr *HeaderRenderer) ResetHeaderBackground(rowIdx, colIdx int) {
	//hr.setColHeaderBackground(hr.colHeadersScroll, colIdx, headerBackgroundColor, headerBorderLightColor, 1)
	//hr.setColHeaderBackground(hr.colHeadersFixed, colIdx, headerBackgroundColor, headerBorderLightColor, 1)

	//hr.setRowHeaderBackground(hr.rowHeadersScroll, rowIdx, headerBackgroundColor, headerBorderLightColor, 1)
	//hr.setRowHeaderBackground(hr.rowHeadersFixed, rowIdx, headerBackgroundColor, headerBorderLightColor, 1)
}

/*
func (hr *HeaderRenderer) setColHeaderBackground(headerMap map[int]*HeaderCarcuss, idx int, bgColor, bdrColor color.Color, borderWidth float32) {
	if header, exists := headerMap[idx]; exists {
		header.Background.FillColor = bgColor
		header.BottomBorder.StrokeColor = bdrColor
		header.BottomBorder.StrokeWidth = borderWidth
		header.Background.Refresh()
	}
}
func (hr *HeaderRenderer) setRowHeaderBackground(headerMap map[int]*HeaderCarcuss, idx int, bgColor, bdrColor color.Color, borderWidth float32) {
	if header, exists := headerMap[idx]; exists {
		header.Background.FillColor = bgColor
		header.RightBorder.StrokeColor = bdrColor
		header.RightBorder.StrokeWidth = borderWidth
		header.Background.Refresh()
	}
}
*/
