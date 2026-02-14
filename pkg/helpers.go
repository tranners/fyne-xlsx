package pkg

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

func (wb WorkBook) WorkBookItem(item string) *WorkSheet {

	if sh, ok := wb.WorkSheets[item]; ok {
		return sh
	}

	return nil

}

func BorderStyleFromExcel(style int) BorderStyle {
	switch style {
	case 0:
		return BorderStyleNone
	case 1:
		return BorderStyleThin
	case 2:
		return BorderStyleMedium
	case 3:
		return BorderStyleDashed
	case 4:
		return BorderStyleDotted
	case 5:
		return BorderStyleThick
	case 6:
		return BorderStyleDouble
	case 7:
		return BorderStyleHair
	case 8:
		return BorderStyleMeduimDashed
	case 9:
		return BorderStyleDashDot
	case 10:
		return BorderStyleMeduimDashDot
	case 11:
		return BorderStyleDashDotDot
	case 12:
		return BorderStyleSlantDashDot
	default:
		return BorderStyleNone
	}
}

func (bs BorderStyle) GetWidth() float32 {
	switch bs {
	case BorderStyleNone:
		return 0
	case BorderStyleHair:
		return 1.0
	case BorderStyleThin:
		// first border option
		return 1.0
	case BorderStyleMedium, BorderStyleMeduimDashed, BorderStyleMeduimDashDot:
		// BorderStyleMedium, second level
		return 2.0
	case BorderStyleThick:
		// BorderStyleThick third level
		return 3.0
	case BorderStyleDouble:
		return 3.0 // maybe two lines here.
	case BorderStyleDashed, BorderStyleDotted:
		return 1.0
	case BorderStyleDashDot, BorderStyleDashDotDot, BorderStyleSlantDashDot:
		return 1.0
	default:
		return 1.0
	}
}

// formatDisplayValue converts the raw value to Excel's display format
func formatDisplayValue(value string, formatCode string, isString bool) string {
	//func (ev *ExcelViewer) formatDisplayValue(value string, formatCode string, isString bool) string {
	if isString || formatCode == "General" || formatCode == "@" {
		return value // Strings, General, or @ format display as-is
	}

	// Try to parse as a number for numeric or date formats
	if num, err := strconv.ParseFloat(value, 64); err == nil {
		// Check if it's a date format
		dateFormats := regexp.MustCompile(`(?i)[dmyhms]+(?:[^A-Za-z0-9\s]?[dmyhms]+)*`)
		if dateFormats.MatchString(formatCode) {
			// Excel dates are days since 1900-01-01
			excelEpoch := time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)
			days := int64(num)
			hours := int64((num - float64(days)) * 24)
			minutes := int64((num - float64(days) - float64(hours)/24) * 1440)
			seconds := int64((num - float64(days) - float64(hours)/24 - float64(minutes)/1440) * 86400)
			t := excelEpoch.AddDate(0, 0, int(days)).Add(time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second)

			// Map Excel format to Go time format
			goFormat := convertExcelDateFormatToGo(formatCode)
			return t.Format(goFormat)
		}

		// Handle numeric formats
		switch formatCode {
		case "0":
			return fmt.Sprintf("%d", int64(num))
		case "0.00":
			return fmt.Sprintf("%.2f", num)
		case "#,##0":
			return fmt.Sprintf("%d", int64(num)) // Simplified, no thousands separator
		case "#,##0.00":
			return fmt.Sprintf("%.2f", num) // Simplified
		case "0%":
			return fmt.Sprintf("%d%%", int64(num*100))
		case "0.00%":
			return fmt.Sprintf("%.2f%%", num*100)
		default:
			return value // Fallback to raw value for unsupported formats
		}
	}

	// Fallback for non-numeric values or invalid formats
	return value
}

// convertExcelDateFormatToGo maps Excel date formats to Go time formats (simplified)
func convertExcelDateFormatToGo(excelFormat string) string {
	formatMap := map[string]string{
		"m/d/yyyy":      "1/2/2006",
		"d-mmm-yy":      "2-Jan-06",
		"d-mmm":         "2-Jan",
		"mmm-yy":        "Jan-06",
		"h:mm AM/PM":    "3:04 PM",
		"h:mm:ss AM/PM": "3:04:05 PM",
		"h:mm":          "15:04",
		"h:mm:ss":       "15:04:05",
		"m/d/yyyy h:mm": "1/2/2006 15:04",
		"mm:ss":         "04:05",
		"yyyy/m/d":      "2006/1/2",
	}
	if goFormat, ok := formatMap[excelFormat]; ok {
		return goFormat
	}
	return "2006-01-02" // Default fallback
}

func getNumberFormatCode(file *excelize.File, sheetName, cell string) (string, bool, error) {
	var excelDisplayFormats = map[int]string{
		37: "#,##0;-#,##0",
		38: "#,##0;[Red]-#,##0", // Excel shows minus instead of parentheses
		39: "#,##0.00;-#,##0.00",
		40: "#,##0.00;[Red]-#,##0.00", // Same pattern for decimal version
		// Add other discrepancies as you discover them
	}
	// Map of built-in NumFmtID to format codes (complete list)
	var builtInNumFmts = map[int]string{
		0:  "General",
		1:  "0",
		2:  "0.00",
		3:  "#,##0",
		4:  "#,##0.00",
		5:  "$#,##0_);($#,##0)",
		6:  "$#,##0_);[Red]($#,##0)",
		7:  "$#,##0.00_);($#,##0.00)",
		8:  "$#,##0.00_);[Red]($#,##0.00)",
		9:  "0%",
		10: "0.00%",
		11: "0.00E+00",
		12: "# ?/?",
		13: "# ??/??",
		14: "m/d/yyyy",
		15: "d-mmm-yy",
		16: "d-mmm",
		17: "mmm-yy",
		18: "h:mm AM/PM",
		19: "h:mm:ss AM/PM",
		20: "h:mm",
		21: "h:mm:ss",
		22: "m/d/yyyy h:mm",
		// Note: 23-26 are reserved/unused in Excel
		27: "mm:ss",
		28: "[h]:mm:ss",
		29: "mm:ss.0",
		30: "##0.0E+0",
		31: "@",
		32: "yyyy/m/d",
		33: "yyyy/m/d",
		34: "yyyy/m/d",
		35: "yyyy/m/d",
		36: "m/d/yyyy",
		37: "#,##0_);(#,##0)",
		38: "#,##0_);[Red](#,##0)",
		39: "#,##0.00_);(#,##0.00)",
		40: "#,##0.00_);[Red](#,##0.00)",
		41: "_(* #,##0_);_(* (#,##0);_(* \"-\"_);_(@_)",
		42: "_($* #,##0_);_($* (#,##0);_($* \"-\"_);_(@_)",
		43: "_(* #,##0.00_);_(* (#,##0.00);_(* \"-\"??_);_(@_)",
		44: "_($* #,##0.00_);_($* (#,##0.00);_($* \"-\"??_);_(@_)",
		45: "mm:ss",
		46: "[h]:mm:ss",
		47: "mmss.0",
		48: "##0.0E+0",
		49: "@",
		// Removed duplicate entries 59-63 as they are not standard Excel built-in formats
	}
	// Get the cell's data type
	cellType, err := file.GetCellType(sheetName, cell)
	if err != nil {
		return "", false, fmt.Errorf("failed to get cell type for %s: %v", cell, err)
	}
	isString := cellType == excelize.CellTypeInlineString || cellType == excelize.CellTypeSharedString

	// Get the style ID of the cell
	styleID, err := file.GetCellStyle(sheetName, cell)
	if err != nil {
		return "", isString, fmt.Errorf("failed to get cell style for %s: %v", cell, err)
	}

	// Check if styles are available
	if file.Styles == nil {
		return "General", isString, nil
	}

	// Get the number format ID from the style
	if styleID < 0 || styleID >= len(file.Styles.CellXfs.Xf) {
		return "", isString, fmt.Errorf("invalid style ID: %d", styleID)
	}

	style := file.Styles.CellXfs.Xf[styleID]
	numFmtID := style.NumFmtID
	if numFmtID == nil {
		return "General", isString, nil
	}

	// Check custom formats in NumFmts FIRST (they override built-ins)
	if file.Styles.NumFmts != nil {
		for _, numFmt := range file.Styles.NumFmts.NumFmt {
			if numFmt.NumFmtID == *numFmtID {
				return numFmt.FormatCode, isString, nil
			}
		}

		// FALLBACK: If cell references a built-in format, check if there's a better custom format
		// This handles cases where Excel stores cell style with built-in NumFmtID but custom format exists
		if builtInFormat, isBuiltIn := builtInNumFmts[*numFmtID]; isBuiltIn {
			// Look for custom formats that enhance the built-in format (e.g., add color support)
			for _, numFmt := range file.Styles.NumFmts.NumFmt {
				// Check if this custom format is an enhanced version of the built-in format
				if isEnhancedFormat(builtInFormat, numFmt.FormatCode) {
					return numFmt.FormatCode, isString, nil
				}
			}
		}
	}
	// Check if Excel displays this format differently than stored
	if displayFormat, hasDisplayOverride := excelDisplayFormats[*numFmtID]; hasDisplayOverride {
		return displayFormat, isString, nil
	}

	// Check built-in formats only if no custom format found
	if format, ok := builtInNumFmts[*numFmtID]; ok {
		return format, isString, nil
	}

	return "", isString, fmt.Errorf("no format code found for NumFmtID: %d", *numFmtID)
}

func isEnhancedFormat(builtInFormat, customFormat string) bool {
	// Remove spaces for comparison
	builtIn := strings.ReplaceAll(builtInFormat, " ", "")
	custom := strings.ReplaceAll(customFormat, " ", "")

	// Check if the custom format contains the built-in format as a base
	// and adds enhancements like color codes, conditional formatting, etc.

	// Case 1: Custom format adds color support to built-in format
	// Example: "#,##0.00" (built-in) vs "#,##0.00;[Red]#,##0.00" (custom)
	if strings.Contains(custom, builtIn) {
		// Check for color codes like [Red], [Blue], etc.
		colorPattern := regexp.MustCompile(`\[(?:Red|Blue|Green|Yellow|Magenta|Cyan|White|Black)\]`)
		if colorPattern.MatchString(custom) {
			return true
		}

		// Check for conditional formatting (semicolon-separated sections)
		if strings.Count(custom, ";") > strings.Count(builtIn, ";") {
			return true
		}
	}

	// Case 2: Custom format is a direct enhancement with same base pattern
	// Example: "#,##0.00" vs "#,##0.00;[Red]-#,##0.00"
	if strings.HasPrefix(custom, builtIn+";") {
		return true
	}

	return false
}

// Add to pkg/helpers.go

// CellRefToCoordinates converts "A1" to (col: 1, row: 1)
func CellRefToCoordinates(cellRef string) (col, row int, err error) {
	if cellRef == "" {
		return 0, 0, fmt.Errorf("empty cell reference")
	}

	// Find where letters end and numbers begin
	i := 0
	for i < len(cellRef) && cellRef[i] >= 'A' && cellRef[i] <= 'Z' {
		i++
	}

	if i == 0 || i == len(cellRef) {
		return 0, 0, fmt.Errorf("invalid cell reference: %s", cellRef)
	}

	colStr := cellRef[:i]
	rowStr := cellRef[i:]

	// Convert column letters: A=1, B=2, ..., Z=26, AA=27, etc.
	col = 0
	for _, ch := range colStr {
		col = col*26 + int(ch-'A'+1)
	}

	// Parse row number
	row, err = strconv.Atoi(rowStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid row in %s: %v", cellRef, err)
	}

	return col, row, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
