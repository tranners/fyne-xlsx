package pkg

import (
	"image/color"
)

type HorizontalAlign int

const (
	AlignLeft HorizontalAlign = iota
	AlignCenter
	AlignRight
	AlignGeneral
)

type VerticalAlign int

const (
	VAlignTop VerticalAlign = iota
	VAlignCenter
	VAlignBottom
)

type BorderStyle int

const (
	BorderStyleNone BorderStyle = iota
	BorderStyleThin
	BorderStyleMedium
	BorderStyleDashed
	BorderStyleDotted
	BorderStyleThick
	BorderStyleDouble
	BorderStyleHair
	BorderStyleMeduimDashed
	BorderStyleDashDot
	BorderStyleMeduimDashDot
	BorderStyleDashDotDot
	BorderStyleSlantDashDot
)

// Model Load
type WorkBook struct {
	Name       string
	WorkSheets map[string]*WorkSheet // Key: sheet name
	SheetList  []string              // Preserve order for tabs
}

// Pure Excel data model (no Fyne dependencies)
// Model Load
type WorkSheet struct {
	Name             string
	Rows             [][]string
	Settings         SheetSettings
	Styles           map[string]CellStyle // "A1" -> style
	MergeCells       []MergeRange         // "A1:C3"
	ColWidths        map[int]float32
	RowHeights       map[int]float32
	DefaultColWidth  float32
	DefaultRowHeight float32
	HiddenRows       map[int]bool
	HiddenColumns    map[int]bool
	Formulas         map[string]string // "B5" -> "=SUM(A1:A4)"
	Hyperlinks       map[string]string
	Comments         map[string]string
	FreezePanes      FreezePanes
	TabColor         color.Color // Own color type or use standard
}

// SheetSettings holds the display settings for a sheet, from excelize
// Model Load
type SheetSettings struct {
	ShowGridlines bool
	ShowHeadings  bool
	ShowFormulas  bool
	ShowZeros     bool
	TopLeftCell   string
	IsSet         bool
}

// Model Load
type FreezePanes struct {
	XSplit      int    // Vertical split position (columns)
	YSplit      int    // Horizontal split position (rows)
	TopLeftCell string // Top-left cell of bottom-right pane
	ActivePane  string // Which pane is active
}

type CellStyle struct {
	Font      FontStyle
	Fill      FillStyle
	Alignment AlignmentStyle
	//Borders    BorderSet
	NumFmt     NumberFormat
	Protection Protection
}

type FontStyle struct {
	Family    string
	Size      float32
	Bold      bool
	Italic    bool
	Underline bool
	Strike    bool
	Color     color.Color
}

type FillStyle struct {
	Type    string // "pattern", "gradient"
	Pattern int
	FgColor color.Color // Foreground
	BgColor color.Color // Background
}

type AlignmentStyle struct {
	Horizontal   HorizontalAlign
	Vertical     VerticalAlign
	Indent       int
	WrapText     bool
	TextRotation int
	ShrinkToFit  bool
}

type BorderSet struct {
	Top    Border
	Bottom Border
	Left   Border
	Right  Border
}

type Border struct {
	Style      BorderStyle
	Color      color.Color
	Width      float32
	IsGridline bool
}

type NumberFormat struct {
	Code     string
	IsCustom bool
}

type Protection struct {
	Locked bool
	Hidden bool
}

// Model Load
type MergeRange struct {
	Range string // "A1:C3"
	Data  string
}

type CellID struct {
	Row int
	Col int
}
