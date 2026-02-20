package pkg

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
)

type MergeManager struct {
	// Model Anchor / Model Merged Ranges
	anchorToModelRange map[CellID]*MergeRange

	// key any cell reference, withing a merged range
	cellToMergeAnchor map[CellID]CellID

	// key merged range Anchor cell
	mergedeSizeByModAnchor map[CellID]fyne.Size

	// used in the gridlines
	visIdxMergeCache map[CellID]VisIdxMergeInfo

	anchorHasBackgroundCache map[CellID]bool

	cm   *CoordinateManager
	data *WorkSheetData
}

type MergeModRange struct {
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
		anchorToModelRange:     make(map[CellID]*MergeRange),
		cellToMergeAnchor:      make(map[CellID]CellID),
		mergedeSizeByModAnchor: make(map[CellID]fyne.Size),
	}

	return m
}

func (mm *MergeManager) buildMergeLookup(mergeCells []MergeRange) {
	cm := mm.cm

	data := mm.data

	mm.cellToMergeAnchor = make(map[CellID]CellID)
	mm.anchorToModelRange = make(map[CellID]*MergeRange)
	mm.visIdxMergeCache = make(map[CellID]VisIdxMergeInfo)
	mm.mergedeSizeByModAnchor = make(map[CellID]fyne.Size)
	mm.anchorHasBackgroundCache = make(map[CellID]bool)

	var startVisRow int
	var endVisRow int
	var startVisCol int
	var endVisCol int

	for _, merge := range mergeCells {
		startModRow, startModCol, endModRow, endModCol, err := merge.GetMergeModBounds()
		if err != nil {
			continue
		}

		anchor := CellID{Row: startModRow, Col: startModCol}
		mm.anchorToModelRange[anchor] = &merge

		mergeModRange := MergeModRange{
			StartModelRow: startModRow,
			StartModelCol: startModCol,
			RowSpan:       endModRow - startModRow + 1,
			ColSpan:       endModCol - startModCol + 1,
		}

		// Build ModIdx→Anchor map\
		/*
			for row := startModRow; row <= endModRow; row++ {
				for col := startModCol; col <= endModCol; col++ {
					mm.cellToMergeAnchor[CellID{Row: row, Col: col}] = anchor
				}
			}
		*/

		startVisRow = -1
		endVisRow = -1
		startVisCol = -1
		endVisCol = -1

		height := float32(0)
		for i := mergeModRange.StartModelRow; i < mergeModRange.StartModelRow+mergeModRange.RowSpan; i++ {
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
		for i := mergeModRange.StartModelCol; i < mergeModRange.StartModelCol+mergeModRange.ColSpan; i++ {
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

		visibleAnchorModRow := cm.GetRowModIdxFromVisIdx(startVisRow)
		visibleAnchorModCol := cm.GetColModIdxFromVisIdx(startVisCol)
		visibleAnchor := CellID{Row: visibleAnchorModRow, Col: visibleAnchorModCol}

		for row := startModRow; row <= endModRow; row++ {
			for col := startModCol; col <= endModCol; col++ {
				mm.cellToMergeAnchor[CellID{Row: row, Col: col}] = visibleAnchor // Changed!
			}
		}

		// merged range Size()
		// key merged range anchor cell
		mm.mergedeSizeByModAnchor[visibleAnchor] = fyne.NewSize(width, height)
		//mm.mergedeSizeByModAnchor[anchor] = fyne.NewSize(width, height)

		mergeVisInfo := VisIdxMergeInfo{
			VisRowStart: startVisRow,
			VisRowEnd:   endVisRow,
			VisColStart: startVisCol,
			VisColEnd:   endVisCol,
		}

		for visRow := startVisRow; visRow <= endVisRow; visRow++ {
			for visCol := startVisCol; visCol <= endVisCol; visCol++ {
				mm.visIdxMergeCache[CellID{visRow, visCol}] = mergeVisInfo
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

func (mm *MergeManager) isMergeRangeTransparent(cellModId CellID) bool {
	if _, exists := mm.anchorHasBackgroundCache[cellModId]; exists {
		return false
	}
	return true
}

func (mm *MergeManager) GetMergedRangeByVisId(cellVisId CellID) (VisIdxMergeInfo, bool) {
	if _, exist := mm.visIdxMergeCache[cellVisId]; exist {
		return mm.visIdxMergeCache[cellVisId], true
	}
	return VisIdxMergeInfo{}, false
}

func (mm *MergeManager) IsVisibleMergeAnchor(cellModID CellID) bool {
	_, exists := mm.mergedeSizeByModAnchor[cellModID]
	return exists
}

func (mm *MergeManager) GetMergeSize(anchor CellID) (fyne.Size, bool) {
	if size, exists := mm.mergedeSizeByModAnchor[anchor]; exists {
		return size, true
	} else {
		return fyne.Size{Width: -1, Height: -1}, true
	}
}

func (m *MergeRange) GetMergeModBounds() (startModRowIdx, startModColIdx, endModRowIdx, endModColIdx int, err error) {
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

func (mm *MergeManager) ForEachVisibleMerge(fn func(anchor CellID)) {
	for anchor := range mm.mergedeSizeByModAnchor {
		fn(anchor)
	}
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
	mergeRange, exists := mm.anchorToModelRange[anchor]
	if !exists {
		return false
	}

	startRow, startCol, endRow, endCol, _ := mergeRange.GetMergeModBounds()

	vpStartRow := mm.cm.GetRowModIdxFromVisIdx(viewport.FirstRowVisIdx)
	vpEndRow := mm.cm.GetRowModIdxFromVisIdx(viewport.LastRowVisIdx)
	vpStartCol := mm.cm.GetColModIdxFromVisIdx(viewport.FirstColVisIdx)
	vpEndCol := mm.cm.GetColModIdxFromVisIdx(viewport.LastColVisIdx)

	rowOverlap := !(endRow < vpStartRow || startRow > vpEndRow)
	colOverlap := !(endCol < vpStartCol || startCol > vpEndCol)

	return rowOverlap && colOverlap
}
