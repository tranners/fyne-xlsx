package pkg

import (
	"image/color"

	"fyne.io/fyne/v2"
)

/*
type WorkbookSettings struct {
	DefaultColWidth  float32
	DefaultRowHeight float32
}
*/

type WorkBookData struct {
	Name           string
	WorkSheetsData map[string]*WorkSheetData
	WorkSheetList  []string
}

type WorkSheetData struct {
	Name          string
	Rows          [][]string
	Settings      SheetSettings
	GridData      map[CellID]*CellData
	Formulas      map[CellID]string
	Hyperlinks    map[CellID]string
	Comments      map[CellID]string
	MergeCells    []MergeRange
	ColWidths     map[int]float32
	RowHeights    map[int]float32
	HiddenColumns map[int]bool
	HiddenRows    map[int]bool

	FreezePanes FreezePanes
	TabColor    color.Color
	offset      fyne.Position

	RowGroupOutlineLevels map[int]int // map[rowIndex]outlineLevel
	ColGroupOutlineLevels map[int]int // map[colIndex]outlineLevel

	RowsNof int
	ColsNof int
}

type CellData struct {
	Value  string
	Style  *CellStyle
	Border *BorderSet
}

type SheetDisplaySettings struct {
	ShowHeaders   bool
	ShowGridlines bool
}
