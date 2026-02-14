package pkg

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
)

type MergeManager struct {
	cellToMergeAnchor map[CellID]CellID
	anchorToRange     map[CellID]*MergeRange
	merges            []NewMergeRange
	//visibleMerges     []VisibleMerge
	visibleMergeSet map[CellID]bool
	anchorToSize    map[CellID]fyne.Size

	// used in the gridlines
	visIdxMergeCache map[CellID]VisIdxMergeInfo

	anchorHasBackgroundCache map[CellID]bool

	cm   *CoordinateManager
	data *WorkSheetData
}

type NewMergeRange struct {
	StartModelRow int
	StartModelCol int
	RowSpan       int
	ColSpan       int
}

type VisIdxMergeInfo struct {
	VisRowStart int
	VisRowEnd   int
	VisColStart int
	VisColEnd   int
}

func NewMergeManager(cm *CoordinateManager, data *WorkSheetData) *MergeManager {
	m := &MergeManager{cm: cm, data: data,
		anchorToRange:     make(map[CellID]*MergeRange),
		cellToMergeAnchor: make(map[CellID]CellID),
		visibleMergeSet:   make(map[CellID]bool),
		anchorToSize:      make(map[CellID]fyne.Size),
	}

	return m
}

func (mm *MergeManager) buildMergeLookup(mergeCells []MergeRange) {

	//x := mm.c
	cm := mm.cm

	data := mm.data

	mm.merges = mm.merges[:0]
	mm.cellToMergeAnchor = make(map[CellID]CellID)
	mm.anchorToRange = make(map[CellID]*MergeRange)
	mm.visIdxMergeCache = make(map[CellID]VisIdxMergeInfo)
	mm.visibleMergeSet = make(map[CellID]bool)
	mm.anchorToSize = make(map[CellID]fyne.Size)
	mm.anchorHasBackgroundCache = make(map[CellID]bool)

	var startVisRow int
	var endVisRow int
	var startVisCol int
	var endVisCol int

	for _, merge := range mergeCells {
		startRow, startCol, endRow, endCol, err := merge.GetBounds()
		if err != nil {
			continue
		}

		anchor := CellID{Row: startRow, Col: startCol}
		mm.anchorToRange[anchor] = &merge

		newMergeRange := NewMergeRange{
			StartModelRow: startRow,
			StartModelCol: startCol,
			RowSpan:       endRow - startRow + 1,
			ColSpan:       endCol - startCol + 1,
		}

		mm.merges = append(mm.merges, newMergeRange)

		// Build ModIdx→Anchor map
		for row := startRow; row <= endRow; row++ {
			for col := startCol; col <= endCol; col++ {
				mm.cellToMergeAnchor[CellID{Row: row, Col: col}] = anchor
			}
		}

		startVisRow = -1
		endVisRow = -1
		startVisCol = -1
		endVisCol = -1

		height := float32(0)
		for i := newMergeRange.StartModelRow; i < newMergeRange.StartModelRow+newMergeRange.RowSpan; i++ {
			visIdx := cm.GetRowVisIdxFromModIdx(i)
			if visIdx != -1 {
				if startVisRow == -1 {
					startVisRow = visIdx
				}
				endVisRow = visIdx
				height += cm.GetHeightByVisIdx(visIdx)
			}
		}

		width := float32(0)
		for i := newMergeRange.StartModelCol; i < newMergeRange.StartModelCol+newMergeRange.ColSpan; i++ {
			visIdx := cm.GetColVisIdxFromModIdx(i)
			if visIdx != -1 {
				if startVisCol == -1 {
					startVisCol = visIdx
				}
				endVisCol = visIdx
				width += cm.GetWidthByVisIdx(visIdx)
			}
		}

		if startVisRow == -1 || startVisCol == -1 {
			continue //  completely hidden range
		}

		mm.visibleMergeSet[anchor] = true
		mm.anchorToSize[anchor] = fyne.NewSize(width, height)

		info := VisIdxMergeInfo{
			VisRowStart: startVisRow,
			VisRowEnd:   endVisRow,
			VisColStart: startVisCol,
			VisColEnd:   endVisCol,
		}

		for visRow := startVisRow; visRow <= endVisRow; visRow++ {
			for visCol := startVisCol; visCol <= endVisCol; visCol++ {
				mm.visIdxMergeCache[CellID{visRow, visCol}] = info
			}
		}

		anchorVisIdx := CellID{Row: startVisRow, Col: startVisCol}
		if cellData, exists := data.GridData[anchor]; exists {
			hasBackground := cellData != nil &&
				cellData.Style != nil &&
				cellData.Style.Fill.BgColor != color.Transparent
			mm.anchorHasBackgroundCache[anchorVisIdx] = hasBackground
		} else {
			mm.anchorHasBackgroundCache[anchorVisIdx] = false
		}
	}
}

func (mm *MergeManager) IsCellMerged(cellModID CellID) (CellID, bool) {
	if anchor, merged := mm.cellToMergeAnchor[cellModID]; merged {
		return anchor, true
	}
	return CellID{Row: 0, Col: 0}, false
}

func (mm *MergeManager) IsMergedRange(cellVisId CellID) bool {
	if _, merged := mm.visIdxMergeCache[cellVisId]; merged {
		return true
	}
	return false
}

func (mm *MergeManager) IsVisibleMergeAnchor(cellModID CellID) bool {
	if mm.visibleMergeSet != nil && mm.visibleMergeSet[cellModID] {
		return true
	}
	return false

}

func (mm *MergeManager) GetMergeSize(anchor CellID) (fyne.Size, bool) {
	size, exists := mm.anchorToSize[anchor]
	return size, exists
}

func (m *MergeRange) GetBounds() (startModRowIdx, startModColIdx, endModRowIdx, endModColIdx int, err error) {
	parts := strings.Split(m.Range, ":")
	if len(parts) != 2 {
		return 0, 0, 0, 0, fmt.Errorf("invalid merge range: %s", m.Range)
	}

	startModColIdx, startModRowIdx, err = CellRefToCoordinates(parts[0])
	if err != nil {
		return 0, 0, 0, 0, err
	}

	endModColIdx, endModRowIdx, err = CellRefToCoordinates(parts[1])
	if err != nil {
		return 0, 0, 0, 0, err
	}

	return startModRowIdx - 1, startModColIdx - 1, endModRowIdx - 1, endModColIdx - 1, nil
}

func (m *MergeRange) GetAnchor() (CellID, error) {
	startRow, startCol, _, _, err := m.GetBounds()
	if err != nil {
		return CellID{}, err
	}
	return CellID{Row: startRow, Col: startCol}, nil
}

func (mm *MergeManager) GetCellAnchor(cellID CellID) (anchor CellID, isMerged bool) {
	anchor, isMerged = mm.cellToMergeAnchor[cellID]
	if !isMerged {
		anchor = cellID
	}
	return anchor, isMerged
}

// Check if merge bounds overlap the viewport bounds
func (mm *MergeManager) IsMergeInViewport(anchor CellID, viewport Viewport) bool {
	// Get the merge range from anchor
	mergeRange, exists := mm.anchorToRange[anchor]
	if !exists {
		return false
	}

	startRow, startCol, endRow, endCol, _ := mergeRange.GetBounds()

	vpStartRow := mm.cm.GetRowModIdxFromVisIdx(viewport.FirstRowVisIdx)
	vpEndRow := mm.cm.GetRowModIdxFromVisIdx(viewport.LastRowVisIdx)
	vpStartCol := mm.cm.GetColModIdxFromVisIdx(viewport.FirstColVisIdx)
	vpEndCol := mm.cm.GetColModIdxFromVisIdx(viewport.LastColVisIdx)

	rowOverlap := !(endRow < vpStartRow || startRow > vpEndRow)
	colOverlap := !(endCol < vpStartCol || startCol > vpEndCol)

	return rowOverlap && colOverlap
}
