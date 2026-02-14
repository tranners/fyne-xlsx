package xlsutil

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func CellNameToCoords(cell string) (int, int, error) {
	colName, row, err := splitCellName(cell)
	if err != nil {
		return -1, -1, fmt.Errorf("cannot convert cell %q to coordinates: %v", cell, err)
	}
	if row > 1048576 {
		return -1, -1, errors.New("row number exceeds maximum limit")
	}
	col, err := ColumnNameToNumber(colName)

	return col, row, err
}

func splitCellName(cell string) (string, int, error) {
	alpha := func(r rune) bool {
		return ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z') || (r == 36)
	}
	if strings.IndexFunc(cell, alpha) == 0 {
		i := strings.LastIndexFunc(cell, alpha)
		if i >= 0 && i < len(cell)-1 {
			col, rowStr := strings.ReplaceAll(cell[:i+1], "$", ""), cell[i+1:]
			if row, err := strconv.Atoi(rowStr); err == nil && row > 0 {
				return col, row, nil
			}
		}
	}
	return "", -1, fmt.Errorf("invalid cell name %q", cell)
}

func ColumnNameToNumber(name string) (int, error) {
	if len(name) == 0 {
		return -1, fmt.Errorf("invalid column name %q", name)
	}
	col := 0
	multi := 1
	for i := len(name) - 1; i >= 0; i-- {
		r := name[i]
		if r >= 'A' && r <= 'Z' {
			col += int(r-'A'+1) * multi
		} else if r >= 'a' && r <= 'z' {
			col += int(r-'a'+1) * multi
		} else {
			return -1, fmt.Errorf("invalid column name %q", name)
		}
		multi *= 26
	}
	if col > 16384 {
		return -1, fmt.Errorf("the column number must be greater than or equal to %d and less than or equal to %d", 1, 16384)
	}
	return col, nil
}

func CoordsToCellName(col, row int, abs ...bool) (string, error) {
	if col < 1 || row < 1 {
		return "", fmt.Errorf("invalid cell reference [%d, %d]", col, row)
	}
	if row > 1048576 {
		return "", errors.New("row number exceeds maximum limit")
	}
	sign := ""
	for _, a := range abs {
		if a {
			sign = "$"
		}
	}
	colName, err := ColumnNumberToName(col)
	return sign + colName + sign + strconv.Itoa(row), err
}

func ColumnNumberToName(num int) (string, error) {
	if num < 1 || num > 16384 {
		return "", fmt.Errorf("the column number must be greater than or equal to %d and less than or equal to %d", 1, 16384)
	}
	estimatedLength := 0
	for n := num; n > 0; n = (n - 1) / 26 {
		estimatedLength++
	}

	result := make([]byte, estimatedLength)
	for num > 0 {
		estimatedLength--
		result[estimatedLength] = byte((num-1)%26 + 'A')
		num = (num - 1) / 26
	}
	return string(result), nil
}

func GetStartAxis(rng string) string {
	return strings.Split(rng, ":")[0]
}

// GetEndAxis returns the bottom right cell reference of merged range, for
// example: "D4".
func GetEndAxis(rng string) string {
	coordinates := strings.Split(rng, ":")
	if len(coordinates) == 2 {
		return coordinates[1]
	}
	return coordinates[0]
}

/*
func ConvertBorderStyle(style int) pkg.BorderStyle {
	switch style {
	case 0:
		return pkg.BorderStyleNone
	case 1:
		return pkg.BorderStyleThin
	case 2:
		return pkg.BorderStyleMedium
	case 3:
		return pkg.BorderStyleDashed
	case 4:
		return pkg.BorderStyleDotted
	case 5:
		return pkg.BorderStyleThick
	case 6:
		return pkg.BorderStyleDouble
	case 7:
		return pkg.BorderStyleHair
	case 8:
		return pkg.BorderStyleMeduimDashed
	case 9:
		return pkg.BorderStyleDashDot
	case 10:
		return pkg.BorderStyleMeduimDashDot
	case 11:
		return pkg.BorderStyleDashDotDot
	case 12:
		return pkg.BorderStyleSlantDashDot
	default:
		return pkg.BorderStyleNone
	}
}

*/
