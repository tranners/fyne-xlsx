package pkg

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
)

type MergeModBounds struct {
	StartRow, StartCol int
	RowSpan, ColSpan   int
}

type VisibleMerge struct {
	MergeIdx int // Index back to modelMerges[] for data lookup

	// Visible bounds (for gridline suppression)
	VisRowStart, VisRowEnd int
	VisColStart, VisColEnd int

	// Rendering position
	VisAnchor CellID // First visible cell (rendering position)
	PixelSize fyne.Size

	// Data cache
	Value         string
	HasBackground bool
}

type MergeManager struct {
	// === IMMUTABLE MODEL DATA ===
	modelMerges      []MergeRange     // Original Excel merge ranges
	modelMergeBounds []MergeModBounds // Precomputed: startRow/Col, rowSpan/colSpan
	modelAnchors     []CellID         // Model anchor (top-left in model space)

	// === VISIBLE SPACE RENDERING DATA ===
	visibleMerges []VisibleMerge // Only merges with at least 1 visible cell

	// Quick lookups
	visCellToVisibleMergeIdx map[CellID]int // Vis cell → index in visibleMerges[]
	modelCellToMergeIdx      map[CellID]int // Model cell → index in modelMerges[]

	/// OLD MODEL STRUCTS

	// ONLY USED IN BOTDERS
	anchorToModelRange map[CellID]*MergeRange
	//anchorToModelBounds map[CellID]MergeModRange

	// key any cell reference, withing a merged range
	cellToMergeAnchor map[CellID]CellID

	// key merged range Anchor cell
	mergedeSizeByVisAnchor map[CellID]fyne.Size

	// used in the gridlines
	visIdxMergeCache map[CellID]VisIdxMergeInfo

	//anchorHasBackgroundCache map[CellID]bool
	anchorHasBackgroundByVisAnchor map[CellID]bool

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
		anchorToModelRange: make(map[CellID]*MergeRange),
		//anchorToModelBounds:    make(map[CellID]MergeModRange),
		cellToMergeAnchor:      make(map[CellID]CellID),
		mergedeSizeByVisAnchor: make(map[CellID]fyne.Size),

		visCellToVisibleMergeIdx: make(map[CellID]int),
		modelCellToMergeIdx:      make(map[CellID]int),
	}

	return m
}

func (mm *MergeManager) Init() {

	mergeCells := mm.data.MergeCells

	for item, modelMerge := range mergeCells {

		startModRow, startModCol, endModRow, endModCol, err := modelMerge.GetMergeModBounds()
		if err != nil {
			continue
		}

		modelMergeBounds := MergeModBounds{
			StartRow: startModRow,
			StartCol: startModCol,
			RowSpan:  endModRow - startModRow + 1,
			ColSpan:  endModCol - startModCol + 1,
		}

		modelAnchor := CellID{Row: startModRow, Col: startModCol}

		// append ro slices
		mm.modelMerges = append(mm.modelMerges, modelMerge)

		mm.modelMergeBounds = append(mm.modelMergeBounds, modelMergeBounds)

		mm.modelAnchors = append(mm.modelAnchors, modelAnchor)

		cellID := CellID{
			Row: modelMergeBounds.StartRow,
			Col: modelMergeBounds.StartCol,
		}
		mm.modelCellToMergeIdx[cellID] = item

		for row := startModRow; row <= endModRow; row++ {
			for col := startModCol; col <= endModCol; col++ {
				mm.modelCellToMergeIdx[CellID{Row: row, Col: col}] = item
			}
		}
	}

}

func (mm *MergeManager) buildMergeLookup(mergeCells []MergeRange) {
	cm := mm.cm

	data := mm.data

	//mm.cellToMergeAnchor = make(map[CellID]CellID)
	mm.anchorToModelRange = make(map[CellID]*MergeRange)
	mm.visIdxMergeCache = make(map[CellID]VisIdxMergeInfo)
	mm.mergedeSizeByVisAnchor = make(map[CellID]fyne.Size)
	mm.anchorHasBackgroundByVisAnchor = make(map[CellID]bool)

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

		//mm.anchorToModelBounds[anchor] = mergeModRange

		startVisRow = -1
		endVisRow = -1
		startVisCol = -1
		endVisCol = -1

		height := float32(0)
		for modIdx := mergeModRange.StartModelRow; modIdx < mergeModRange.StartModelRow+mergeModRange.RowSpan; modIdx++ {
			visIdx := cm.GetRowVisIdxFromModIdx(modIdx)
			if visIdx != -1 {
				if startVisRow == -1 {
					startVisRow = visIdx
				}
				endVisRow = visIdx
				height += cm.GetHeightByVisIdx(visIdx)
			}
		}

		width := float32(0)
		for modIdx := mergeModRange.StartModelCol; modIdx < mergeModRange.StartModelCol+mergeModRange.ColSpan; modIdx++ {
			visIdx := cm.GetColVisIdxFromModIdx(modIdx)
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

		//for row := startModRow; row <= endModRow; row++ {
		//	for col := startModCol; col <= endModCol; col++ {
		//		mm.cellToMergeAnchor[CellID{Row: row, Col: col}] = visibleAnchor // Changed!
		//	}
		//}

		// merged range Size()
		// key merged range anchor cell
		mm.mergedeSizeByVisAnchor[visibleAnchor] = fyne.NewSize(width, height)
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
			mm.anchorHasBackgroundByVisAnchor[anchorVisIdx] = hasBackground
		} else {
			mm.anchorHasBackgroundByVisAnchor[anchorVisIdx] = false
		}
	}
}

func (mm *MergeManager) Rebuild() {
	var startVisRow int
	var endVisRow int
	var startVisCol int
	var endVisCol int

	cm := mm.cm

	mm.visibleMerges = mm.visibleMerges[:0]

	for k := range mm.visCellToVisibleMergeIdx {
		delete(mm.visCellToVisibleMergeIdx, k)
	}

	for item, mergeBounds := range mm.modelMergeBounds {

		startVisRow = -1
		endVisRow = -1
		startVisCol = -1
		endVisCol = -1

		height := float32(0)
		for modIdx := mergeBounds.StartRow; modIdx < mergeBounds.StartRow+mergeBounds.RowSpan; modIdx++ {
			visIdx := cm.GetRowVisIdxFromModIdx(modIdx)
			if visIdx != -1 {
				if startVisRow == -1 {
					startVisRow = visIdx
				}
				endVisRow = visIdx
				height += cm.GetHeightByVisIdx(visIdx)
			}
		}

		width := float32(0)
		for modIdx := mergeBounds.StartCol; modIdx < mergeBounds.StartCol+mergeBounds.ColSpan; modIdx++ {
			visIdx := cm.GetColVisIdxFromModIdx(modIdx)
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

		visibleMerge := VisibleMerge{
			MergeIdx:    item,
			VisRowStart: startVisRow,
			VisRowEnd:   endVisRow,
			VisColStart: startVisCol,
			VisColEnd:   endVisCol,

			VisAnchor: CellID{
				Row: startVisRow,
				Col: startVisCol,
			},

			PixelSize: fyne.Size{
				Height: height,
				Width:  width,
			},

			Value:         mm.modelMerges[item].Data,
			HasBackground: false,
		}
		mm.visibleMerges = append(mm.visibleMerges, visibleMerge)

		mm.visCellToVisibleMergeIdx[visibleMerge.VisAnchor] = len(mm.visibleMerges) - 1

	}

}

func (mm *MergeManager) isCellInMergedRange(cellModID CellID) bool {
	if _, merged := mm.modelCellToMergeIdx[cellModID]; merged {
		return true
	}
	return false
}

func (mm *MergeManager) hasMergeRangeBackgroundByVisAnchorId(visAnchorId CellID) bool {
	if _, exists := mm.anchorHasBackgroundByVisAnchor[visAnchorId]; exists {
		return true
	}
	return false
}

func (mm *MergeManager) GetMergedRangeByVisId(cellVisId CellID) (VisIdxMergeInfo, bool) {
	if _, exist := mm.visIdxMergeCache[cellVisId]; exist {
		return mm.visIdxMergeCache[cellVisId], true
	}
	return VisIdxMergeInfo{}, false
}

// only used in border logic atm
func (mm *MergeManager) IsVisibleMergeAnchor(cellModID CellID) bool {
	_, exists := mm.mergedeSizeByVisAnchor[cellModID]
	return exists
}

func (mm *MergeManager) GetMergeSize(anchor CellID) (fyne.Size, bool) {
	if size, exists := mm.mergedeSizeByVisAnchor[anchor]; exists {
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

// ONLY call from borders; leave for now
func (mm *MergeManager) GetCellAnchor(cellID CellID) (anchor CellID, isMerged bool) {
	anchor, isMerged = mm.cellToMergeAnchor[cellID]
	if !isMerged {
		anchor = cellID
	}
	return anchor, isMerged
}

func (mm *MergeManager) ForEachVisibleMerge(fn func(merge *VisibleMerge)) {
	for i := range mm.visibleMerges {
		fn(&mm.visibleMerges[i])
	}
}
