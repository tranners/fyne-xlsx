package internal

import (
	"fmt"
	"image/color"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/tranners/fyne-xlsx/pkg"
	"github.com/xuri/excelize/v2"
)

// internal/parse.go

type InternalLoader struct{}

func NewLoader() pkg.WorkbookLoader {
	return &InternalLoader{}
}

func (l *InternalLoader) Load(r io.Reader, sheetName string) (*pkg.WorkBookData, error) {
	wb, _ := LoadExcelWorkbook(r, sheetName)

	return &wb, nil
}

func LoadExcelWorkbook(r io.Reader, sheetName string) (pkg.WorkBookData, error) {
	start := time.Now()
	defer func() {
		log.Printf("LoadExcelWorkbook took %v", time.Since(start))
	}()
	f, err := excelize.OpenReader(r)
	if err != nil {
		return pkg.WorkBookData{}, err
	}
	defer f.Close()

	wbData := &pkg.WorkBookData{
		WorkSheetsData: make(map[string]*pkg.WorkSheetData),
	}

	docProps, err := f.GetDocProps()
	if err != nil {
		return pkg.WorkBookData{}, err
	}

	wbData.Name = docProps.Title

	var sheets []string
	if sheetName != "" {
		// Verify the sheet exists
		sheetExists := false
		for _, name := range f.GetSheetList() {
			if name == sheetName {
				sheetExists = true
				sheets = []string{sheetName}
				break
			}
		}
		if !sheetExists {
			return pkg.WorkBookData{}, fmt.Errorf("sheet %q not found", sheetName)
		}
	} else {
		sheets = f.GetSheetList()
	}

	for _, sheetName := range sheets {

		// Initialize SheetData with maps
		sheetdata := &pkg.WorkSheetData{
			Name: sheetName,
			//CellStyles: make(map[pkg.CellID]*pkg.CellStyle),
			ColWidths:  make(map[int]float32),
			RowHeights: make(map[int]float32),
			GridData:   make(map[pkg.CellID]*pkg.CellData),

			// OK, BUT NOT POPULATED
			//DefaultColWidth: 64.0, // Excel default
			// OK, BUT NOT POPULATED
			//DefaultRowHeight: 15.0, // Excel default
			HiddenRows:    make(map[int]bool),
			HiddenColumns: make(map[int]bool),
			Formulas:      make(map[pkg.CellID]string),
			Hyperlinks:    make(map[pkg.CellID]string),
			Comments:      make(map[pkg.CellID]string),

			RowGroupOutlineLevels: make(map[int]int),
			ColGroupOutlineLevels: make(map[int]int),
			// MISSING
			// Settings SheetSettings
			// FreeezePanes FreezePanes
			// TabColor color.Color
			// DataValidation, NOT IMPLEMENTED
			// ConditionalFormat []ConditionalFormat, Implement Later Perhaps
		}

		rowsData, err := f.GetRows(sheetName)
		if err != nil {
			return pkg.WorkBookData{}, err
		}
		sheetdata.Rows = rowsData

		sheetdata.RowsNof = len(rowsData)
		if sheetdata.RowsNof == 0 {
			sheetdata.ColsNof = 0
		} else {
			sheetdata.ColsNof = len(rowsData[0])
		}

		// Load sheet settings
		sheetdata.Settings = loadSheetSettings(f, sheetName)

		// Load column widths and hidden columns
		cols, err := f.GetCols(sheetName)
		if err == nil {
			for colIdx := range cols {
				cellRef := columnIndexToName(colIdx)
				width, err := f.GetColWidth(sheetName, cellRef)
				if err == nil {
					sheetdata.ColWidths[colIdx] = float32(width * 7) // Convert to pixels (~7px per char)
				}
				visible, err := f.GetColVisible(sheetName, cellRef)
				if err == nil && !visible {
					sheetdata.HiddenColumns[colIdx] = !visible
				}
				level, err := f.GetColOutlineLevel(sheetName, cellRef)
				if err == nil && level > 0 {
					sheetdata.ColGroupOutlineLevels[colIdx] = int(level) // Convert uint8 to int
				}
			}
		}

		// Load row heights and hidden rows
		for rowIdx := range rowsData {
			height, err := f.GetRowHeight(sheetName, rowIdx+1)
			if err == nil {
				sheetdata.RowHeights[rowIdx] = float32(height * 1.33)
			}

			visible, err := f.GetRowVisible(sheetName, rowIdx+1)
			if err == nil && !visible {
				sheetdata.HiddenRows[rowIdx] = !visible
			}
			level, err := f.GetRowOutlineLevel(sheetName, rowIdx+1)
			if err == nil && level > 0 {
				sheetdata.RowGroupOutlineLevels[rowIdx] = int(level)
			}
		}

		var cellIndexRow int
		var cellIndexCol int
		// Load cell styles, formulas, and values
		for rowIdx, row := range rowsData {
			for colIdx := range row {
				cellIndexRow = rowIdx
				cellIndexCol = colIdx

				cellRef := cellName(rowIdx+1, colIdx+1)
				//cellStyle := pkg.CellStyle{}
				//var cellStyle *pkg.CellStyle
				// Get cell style
				styleID, err := f.GetCellStyle(sheetName, cellRef)

				style, _ := f.GetStyle(styleID)

				cellStyle, borderSet := convertExcelizeStyle(style, row[colIdx], f)

				sheetdata.GridData[pkg.CellID{Row: cellIndexRow, Col: cellIndexCol}] = &pkg.CellData{
					Value:  rowsData[rowIdx][colIdx],
					Style:  cellStyle,
					Border: borderSet,
				}

				// Get formula if exists
				formula, err := f.GetCellFormula(sheetName, string(cellRef))
				if err == nil && formula != "" {
					sheetdata.Formulas[pkg.CellID{Row: cellIndexRow, Col: cellIndexCol}] = formula
				}

				// Get hyperlink if exists
				_, link, err := f.GetCellHyperLink(sheetName, string(cellRef))
				if err == nil && link != "" {
					sheetdata.Hyperlinks[pkg.CellID{Row: cellIndexRow, Col: cellIndexCol}] = link
				}

				// Get comment if exists
				comments, err := f.GetComments(sheetName)
				if err == nil {
					for _, comment := range comments {
						if comment.Cell == string(cellRef) {
							sheetdata.Comments[pkg.CellID{Row: cellIndexRow, Col: cellIndexCol}] = comment.Text
						}
					}
				}
			}
		}

		// Load merged cells
		mergeCells, _ := f.GetMergeCells(sheetName)
		merges := []pkg.MergeRange{}
		for _, m := range mergeCells {
			merges = append(merges, pkg.MergeRange{
				Range: m[0],
				Data:  m[1],
			})
		}

		sheetdata.MergeCells = merges

		// Extract the panes info
		panes, err := f.GetPanes(sheetName)
		if err == nil {
			sheetdata.FreezePanes = pkg.FreezePanes{
				XSplit: panes.XSplit,
				YSplit: panes.YSplit,
			}
			sheetdata.FreezePanes.TopLeftCell = panes.TopLeftCell
			sheetdata.FreezePanes.ActivePane = panes.ActivePane

		}

		// Load tab color
		sheetProps, err := f.GetSheetProps(sheetName)
		if err == nil {
			sheetdata.TabColor = parseExcelColor(stringValue(sheetProps.TabColorRGB))
		}

		wbData.WorkSheetsData[sheetName] = sheetdata

		wbData.WorkSheetList = append(wbData.WorkSheetList, sheetName)

	}

	return *wbData, nil

}

// loadSheetSettings extracts view settings
func loadSheetSettings(f *excelize.File, sheetName string) pkg.SheetSettings {
	settings := pkg.SheetSettings{
		ShowGridlines: true,
		ShowHeadings:  true,
		ShowZeros:     true,
	}

	view, err := f.GetSheetView(sheetName, 0)
	if err == nil {
		if view.ShowGridLines != nil {
			settings.ShowGridlines = *view.ShowGridLines
		}
		if view.ShowRowColHeaders != nil {
			settings.ShowHeadings = *view.ShowRowColHeaders
		}
		if view.ShowFormulas != nil {
			settings.ShowFormulas = *view.ShowFormulas
		}
		if view.ShowZeros != nil {
			settings.ShowZeros = *view.ShowZeros
		}
		if view.TopLeftCell != nil {
			settings.TopLeftCell = *view.TopLeftCell
		}

		settings.IsSet = true
	}

	return settings
}

func stringValue(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

// Helper functions
func convertHAlign(align string) pkg.HorizontalAlign {

	switch strings.ToLower(align) {
	case "left":
		return pkg.AlignLeft
		//return pkg.HORIZONTAL_ALIGNMENT_LEEDING
	case "center", "centre":
		return pkg.AlignCenter
		//return pkg.HORIZONTAL_ALIGNMENT_CENTER
	case "right":
		return pkg.AlignRight
		//return pkg.HORIZONTAL_ALIGNMENT_TRAILING
	default:
		return pkg.AlignGeneral
		//return pkg.HORIZONTAL_ALIGNMENT_GENERAL
	}
}

func convertVAlign(align string) pkg.VerticalAlign {
	switch strings.ToLower(align) {
	case "top":
		return pkg.VAlignTop
	case "center", "centre":
		return pkg.VAlignCenter
	case "bottom":
		return pkg.VAlignBottom
	default:
		return pkg.VAlignCenter
	}
}

func convertBorder(borders []excelize.Border, side string) pkg.Border {
	for _, b := range borders {
		if strings.ToLower(b.Type) == side {
			return pkg.Border{
				Style: pkg.BorderStyleFromExcel(b.Style),
				Color: parseExcelColor(b.Color),
				Width: 1.0,
			}
		}
	}
	return pkg.Border{Style: pkg.BorderStyleNone}
}

func parseExcelColor(colorStr string) color.Color {
	colorStr = strings.TrimPrefix(colorStr, "#")

	if len(colorStr) == 6 {
		r, _ := strconv.ParseUint(colorStr[0:2], 16, 8)
		g, _ := strconv.ParseUint(colorStr[2:4], 16, 8)
		b, _ := strconv.ParseUint(colorStr[4:6], 16, 8)
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
	} else if len(colorStr) == 8 {
		a, _ := strconv.ParseUint(colorStr[0:2], 16, 8)
		r, _ := strconv.ParseUint(colorStr[2:4], 16, 8)
		g, _ := strconv.ParseUint(colorStr[4:6], 16, 8)
		b, _ := strconv.ParseUint(colorStr[6:8], 16, 8)
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
	}

	return nil
}

func cellName(row, col int) string {
	return columnIndexToName(col-1) + strconv.Itoa(row)
}

func columnIndexToName(col int) string {
	name := ""
	for col >= 0 {
		name = string(rune('A'+col%26)) + name
		col = col/26 - 1
	}
	return name
}

func convertExcelizeStyle(style *excelize.Style, text string, f *excelize.File) (*pkg.CellStyle, *pkg.BorderSet) {

	fill := pkg.FillStyle{}
	if len(style.Fill.Color) == 0 {
		fill = pkg.FillStyle{
			Type:    style.Fill.Type,
			Pattern: style.Fill.Pattern,
			BgColor: color.Transparent,
		}
	} else {
		fill = pkg.FillStyle{
			Type:    style.Fill.Type,
			Pattern: style.Fill.Pattern,
			BgColor: parseExcelColor(style.Fill.Color[0]),
		}
	}

	var alignment pkg.AlignmentStyle

	if style.Alignment != nil {
		alignment = pkg.AlignmentStyle{
			Horizontal: convertHAlign(style.Alignment.Horizontal),
			Vertical:   convertVAlign(style.Alignment.Vertical),
			Indent:     style.Alignment.Indent,
			WrapText:   style.Alignment.WrapText,
		}
	} else {
		// Excel defaults when no alignment specified
		alignment = pkg.AlignmentStyle{
			Horizontal: pkg.AlignGeneral, // Smart alignment
			Vertical:   pkg.VAlignBottom, // Bottom align
			Indent:     0,
			WrapText:   false,
		}
	}

	return &pkg.CellStyle{
			// Font group
			Font: pkg.FontStyle{
				Family:    style.Font.Family,
				Size:      float32(style.Font.Size),
				Bold:      style.Font.Bold,
				Italic:    style.Font.Italic,
				Underline: style.Font.Underline != "",
				//Color:     parseExcelColor(style.Font.Color),
				Color: resolveFontColor(style.Font, f),
			},

			Fill: fill,

			// Alignment group
			Alignment: alignment,

			// Number format
			NumFmt: pkg.NumberFormat{
				Code: fmt.Sprintf("%d", style.NumFmt),
			},
		}, &pkg.BorderSet{
			Top:    convertBorder(style.Border, "top"),
			Bottom: convertBorder(style.Border, "bottom"),
			Left:   convertBorder(style.Border, "left"),
			Right:  convertBorder(style.Border, "right"),
		}
}

// Get the theme color scheme
func getThemeColor(f *excelize.File, themeIndex int) (r, g, b uint8, err error) {
	// Excelize doesn't directly expose theme colors, so you may need to:
	// 1. Use GetTheme() if available in your excelize version
	// 2. Or parse the theme XML manually

	// Default Office theme colors (RGB values)
	defaultTheme := map[int][3]uint8{
		0:  {255, 255, 255}, // Light 1 (white)
		1:  {0, 0, 0},       // Dark 1 (black)
		2:  {238, 236, 225}, // Light 2 (light gray)
		3:  {31, 73, 125},   // Dark 2 (dark blue)
		4:  {79, 129, 189},  // Accent 1 (blue)
		5:  {192, 80, 77},   // Accent 2 (red)
		6:  {155, 187, 89},  // Accent 3 (green)
		7:  {128, 100, 162}, // Accent 4 (purple)
		8:  {75, 172, 198},  // Accent 5 (cyan)
		9:  {247, 150, 70},  // Accent 6 (orange)
		10: {0, 0, 255},     // Hyperlink (blue)
		11: {128, 0, 128},   // Followed Hyperlink (purple)
	}

	if color, ok := defaultTheme[themeIndex]; ok {
		return color[0], color[1], color[2], nil
	}
	return 0, 0, 0, fmt.Errorf("invalid theme index")
}

func applyTint(r, g, b uint8, tint float64) (uint8, uint8, uint8) {
	if tint == 0 {
		return r, g, b // No tint, return original
	}

	// Convert to float for calculation
	rf, gf, bf := float64(r), float64(g), float64(b)

	if tint > 0 {
		// TINT (lighter): blend with white (255, 255, 255)
		// Formula: color + (white - color) × tint
		rf = rf + (255.0-rf)*tint
		gf = gf + (255.0-gf)*tint
		bf = bf + (255.0-bf)*tint
	} else {
		// SHADE (darker): blend with black (0, 0, 0)
		// Formula: color × (1 + tint)
		// Note: tint is negative, so (1 + tint) reduces the color
		rf = rf * (1.0 + tint)
		gf = gf * (1.0 + tint)
		bf = bf * (1.0 + tint)
	}

	// Clamp values to 0-255 range
	return clamp(rf), clamp(gf), clamp(bf)
}

func clamp(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func resolveFontColor(font *excelize.Font, f *excelize.File) color.Color {
	// Priority 1: Direct RGB color
	if font.Color != "" {
		return parseExcelColor(font.Color)
	}

	// Priority 2: Theme-based color
	if font.ColorTheme != nil {
		themeIndex := *font.ColorTheme
		r, g, b, _ := getThemeColor(f, themeIndex)

		// Apply tint if present
		//if font.ColorTint != nil {
		tint := font.ColorTint
		r, g, b = applyTint(r, g, b, tint)
		//}

		return color.RGBA{R: r, G: g, B: b, A: 255}
	}

	// Priority 3: Indexed color (legacy)
	if font.ColorIndexed != 0 {
		//return getIndexedColor(font.ColorIndexed)
	}

	// Default: black
	return color.Black
}
