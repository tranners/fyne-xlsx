package pkg

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

type LineOrientation int

const (
	Horizontal LineOrientation = iota
	Vertical
)

type LineConfig struct {
	PrimaryAxis     int
	SecondAxisStart int
	SecondAxisEnd   int
	UnPositioned    bool
	Orientation     LineOrientation
}

type mode int

const (
	MODE_SEARCHING mode = iota
	MODE_STARTING
	MODE_COMPLETED
)

var borderGridlineColor = color.NRGBA{R: 212, G: 212, B: 212, A: 225}

type lineIndex struct {
	P1 map[int]int
	P2 map[int]int
}

type LineItems struct {
	Lines       []*canvas.Line
	ConfigLines []LineConfig // Using the unified LineConfig you already added
}

type RecycleManager struct {
	Flagger  *PrimitiveGridlineFlagger
	Recycler *PrimitiveGridlineRecycler
}

type hLineConfig struct {
	Row          int
	ColStart     int
	ColEnd       int
	UnPositioned bool
}

type vLineConfig struct {
	Col          int
	RowStart     int
	RowEnd       int
	UnPositioned bool
}

type TransparencyCache map[CellID]bool

type hLineItems struct {
	Lines       []*canvas.Line
	ConfigLines []hLineConfig
}

type vLineItems struct {
	Lines       []*canvas.Line
	ConfigLines []vLineConfig
}

type PrimitiveGridLineRenderer struct {
	ctx *RenderContext

	LineIndex map[GridRegion]map[LineOrientation]map[int]lineIndex
	LineItems map[GridRegion]map[LineOrientation]LineItems

	Managers map[GridRegion]map[LineOrientation]*RecycleManager

	//hLineIndex map[GridRegion]map[int]lineIndex
	//hLineItems map[GridRegion]hLineItems

	hFlaggedItems map[GridRegion]*PrimitiveGridlineFlagger
	hItemRecycler map[GridRegion]*PrimitiveGridlineRecycler

	vFlaggedItems map[GridRegion]*PrimitiveGridlineFlagger
	vItemRecycler map[GridRegion]*PrimitiveGridlineRecycler
}

type PrimitiveGridlineRecycler struct {
	items []PrimitiveGridlineRecyclerItem
}

type PrimitiveGridlineRecyclerItem struct {
	id  int
	obj *canvas.Line
}

func NewLineManager() *RecycleManager {
	return &RecycleManager{
		Flagger:  NewPrimitiveGridlineFlagger(),
		Recycler: NewPrimitiveGridlineRecycler(),
	}
}

func NewPrimitiveGridlineRecycler() *PrimitiveGridlineRecycler {
	return &PrimitiveGridlineRecycler{
		items: []PrimitiveGridlineRecyclerItem{},
	}
}

type PrimitiveGridlineFlagger struct {
	items []int
}

func NewPrimitiveGridlineFlagger() *PrimitiveGridlineFlagger {
	return &PrimitiveGridlineFlagger{
		items: []int{},
	}
}

func NewPrimitiveGridLineRenderer(ctx *RenderContext) *PrimitiveGridLineRenderer {
	return &PrimitiveGridLineRenderer{
		ctx: ctx,

		LineIndex: map[GridRegion]map[LineOrientation]map[int]lineIndex{
			RegionMain: {
				Horizontal: make(map[int]lineIndex),
				Vertical:   make(map[int]lineIndex),
			},
			RegionFixedCorner: {
				Horizontal: make(map[int]lineIndex),
				Vertical:   make(map[int]lineIndex),
			},
			RegionFrozenRows: {
				Horizontal: make(map[int]lineIndex),
				Vertical:   make(map[int]lineIndex),
			},
			RegionFrozenCols: {
				Horizontal: make(map[int]lineIndex),
				Vertical:   make(map[int]lineIndex),
			},
		},
		LineItems: map[GridRegion]map[LineOrientation]LineItems{
			RegionMain: {
				Horizontal: {
					Lines:       []*canvas.Line{},
					ConfigLines: []LineConfig{},
				},
				Vertical: {
					Lines:       []*canvas.Line{},
					ConfigLines: []LineConfig{},
				},
			},
			RegionFixedCorner: {
				Horizontal: {
					Lines:       []*canvas.Line{},
					ConfigLines: []LineConfig{},
				},
				Vertical: {
					Lines:       []*canvas.Line{},
					ConfigLines: []LineConfig{},
				},
			},
			RegionFrozenRows: {
				Horizontal: {
					Lines:       []*canvas.Line{},
					ConfigLines: []LineConfig{},
				},
				Vertical: {
					Lines:       []*canvas.Line{},
					ConfigLines: []LineConfig{},
				},
			},
			RegionFrozenCols: {
				Horizontal: {
					Lines:       []*canvas.Line{},
					ConfigLines: []LineConfig{},
				},
				Vertical: {
					Lines:       []*canvas.Line{},
					ConfigLines: []LineConfig{},
				},
			},
		},
		Managers: map[GridRegion]map[LineOrientation]*RecycleManager{
			RegionMain: {
				Horizontal: NewLineManager(),
				Vertical:   NewLineManager(),
			},
			RegionFixedCorner: {
				Horizontal: NewLineManager(),
				Vertical:   NewLineManager(),
			},
			RegionFrozenRows: {
				Horizontal: NewLineManager(),
				Vertical:   NewLineManager(),
			},
			RegionFrozenCols: {
				Horizontal: NewLineManager(),
				Vertical:   NewLineManager(),
			},
		},
		/*
			hLineItems: map[GridRegion]hLineItems{
				RegionMain: {
					Lines:       []*canvas.Line{},
					ConfigLines: []hLineConfig{},
				},
				RegionFixedCorner: {
					Lines:       []*canvas.Line{},
					ConfigLines: []hLineConfig{},
				},
				RegionFrozenRows: {
					Lines:       []*canvas.Line{},
					ConfigLines: []hLineConfig{},
				},
				RegionFrozenCols: {
					Lines:       []*canvas.Line{},
					ConfigLines: []hLineConfig{},
				},
			},

				hLineIndex: map[GridRegion]map[int]lineIndex{
					RegionMain:        make(map[int]lineIndex),
					RegionFixedCorner: make(map[int]lineIndex),
					RegionFrozenRows:  make(map[int]lineIndex),
					RegionFrozenCols:  make(map[int]lineIndex),
				},
		*/
		hFlaggedItems: map[GridRegion]*PrimitiveGridlineFlagger{
			RegionMain:        NewPrimitiveGridlineFlagger(),
			RegionFixedCorner: NewPrimitiveGridlineFlagger(),
			RegionFrozenRows:  NewPrimitiveGridlineFlagger(),
			RegionFrozenCols:  NewPrimitiveGridlineFlagger(),
		},
		hItemRecycler: map[GridRegion]*PrimitiveGridlineRecycler{
			RegionMain:        NewPrimitiveGridlineRecycler(),
			RegionFixedCorner: NewPrimitiveGridlineRecycler(),
			RegionFrozenRows:  NewPrimitiveGridlineRecycler(),
			RegionFrozenCols:  NewPrimitiveGridlineRecycler(),
		},
		vFlaggedItems: map[GridRegion]*PrimitiveGridlineFlagger{
			RegionMain:        NewPrimitiveGridlineFlagger(),
			RegionFixedCorner: NewPrimitiveGridlineFlagger(),
			RegionFrozenRows:  NewPrimitiveGridlineFlagger(),
			RegionFrozenCols:  NewPrimitiveGridlineFlagger(),
		},
		vItemRecycler: map[GridRegion]*PrimitiveGridlineRecycler{
			RegionMain:        NewPrimitiveGridlineRecycler(),
			RegionFixedCorner: NewPrimitiveGridlineRecycler(),
			RegionFrozenRows:  NewPrimitiveGridlineRecycler(),
			RegionFrozenCols:  NewPrimitiveGridlineRecycler(),
		},
	}
}

func (cyl *PrimitiveGridlineFlagger) Get() (int, bool) {
	if len(cyl.items) > 0 {
		last := len(cyl.items) - 1
		c := cyl.items[last]
		cyl.items = cyl.items[:last]
		return c, true
	}
	return -1, false
}

func (pf *PrimitiveGridlineFlagger) Put(id int) {
	pf.items = append(pf.items, id)
}

func (pf *PrimitiveGridlineFlagger) Reset() {
	pf.items = pf.items[:0]
}

func (cyl *PrimitiveGridlineRecycler) Get() (PrimitiveGridlineRecyclerItem, bool) {
	if len(cyl.items) > 0 {
		last := len(cyl.items) - 1
		c := cyl.items[last]
		cyl.items = cyl.items[:last]
		return c, true
	}
	obj := canvas.NewLine(borderGridlineColor)
	return PrimitiveGridlineRecyclerItem{id: -1, obj: obj}, false
}

func (cyl *PrimitiveGridlineRecycler) Put(id int) {
	item := PrimitiveGridlineRecyclerItem{id: id}
	cyl.items = append(cyl.items, item)
}

func (cyl *PrimitiveGridlineRecycler) Size() int {

	return len(cyl.items)
}

func (pglr *PrimitiveGridLineRenderer) stashInCorner(region GridRegion, orientation LineOrientation) {
	managers := pglr.Managers
	flagger := managers[region][orientation].Flagger
	recycler := managers[region][orientation].Recycler
	lineItems := pglr.LineItems[region][orientation]

	for _, itemId := range flagger.items {

		obj := pglr.LineItems[region][orientation].Lines[itemId]
		objConfig := lineItems.ConfigLines[itemId]
		obj.Move(fyne.NewPos(-9999, -9999))

		recycler.Put(itemId)

		delete(pglr.LineIndex[region][orientation][objConfig.PrimaryAxis].P1, objConfig.SecondAxisStart)
		delete(pglr.LineIndex[region][orientation][objConfig.PrimaryAxis].P2, objConfig.SecondAxisEnd)

		pglr.LineItems[region][orientation].ConfigLines[itemId] = LineConfig{
			PrimaryAxis:     -1,
			SecondAxisStart: -1,
			SecondAxisEnd:   -1,
			Orientation:     orientation,
			UnPositioned:    true,
		}

	}
	// empty it
	flagger.Reset()
}

func (pglr *PrimitiveGridLineRenderer) removeAxes1(prevFirstVisIdx, currFirstVixIdx int, region GridRegion, orientation LineOrientation) {
	flagger := pglr.Managers[region][orientation].Flagger

	for primaryVisIdx := prevFirstVisIdx; primaryVisIdx < currFirstVixIdx; primaryVisIdx++ {
		lineIdx := pglr.LineIndex[region][orientation][primaryVisIdx]
		for _, itemId := range lineIdx.P1 {
			flagger.Put(itemId)
		}
		delete(pglr.LineIndex[region][orientation], primaryVisIdx)
	}
}

func (pglr *PrimitiveGridLineRenderer) removeAxes2(currLastVisIdx, prevLastVisIdx int, region GridRegion, orientation LineOrientation) {
	flagger := pglr.Managers[region][orientation].Flagger

	for primaryVisIdx := currLastVisIdx + 1; primaryVisIdx <= prevLastVisIdx; primaryVisIdx++ {
		lineIdx := pglr.LineIndex[region][orientation][primaryVisIdx]
		for _, itemId := range lineIdx.P1 {
			flagger.Put(itemId)
		}
		delete(pglr.LineIndex[region][orientation], primaryVisIdx)
	}
}

func (pglr *PrimitiveGridLineRenderer) trimEdge1(currFirstVisIdxPrimary,
	currLastVixIdxPrimary,
	prevFirstVisIdxSecondary,
	currFirstVisIdxSecondary int,
	region GridRegion, orientation LineOrientation) {

	flagger := pglr.Managers[region][orientation].Flagger

	for primaryVisIdx := currFirstVisIdxPrimary; primaryVisIdx <= currLastVixIdxPrimary; primaryVisIdx++ {
		lineIdx := pglr.LineIndex[region][orientation][primaryVisIdx]
		for secondaryVisIdx := prevFirstVisIdxSecondary; secondaryVisIdx < currFirstVisIdxSecondary; secondaryVisIdx++ {
			if itemId, exists := lineIdx.P1[secondaryVisIdx]; exists {
				config := pglr.LineItems[region][orientation].ConfigLines[itemId]
				if config.SecondAxisEnd < currFirstVisIdxSecondary {
					flagger.Put(itemId)
				} else {
					pglr.updateLinePosition1(itemId, primaryVisIdx, currFirstVisIdxSecondary, region, orientation)
				}
			}
		}
	}
}

func (pglr *PrimitiveGridLineRenderer) trimEdge2(currFirstVisIdxPrimary,
	currLastVixIdxPrimary,
	prevLastVisIdxSecondary,
	currLastVisIdxSecondary int,
	region GridRegion, orientation LineOrientation) {

	flagger := pglr.Managers[region][orientation].Flagger

	for primaryVisIdx := currFirstVisIdxPrimary; primaryVisIdx <= currLastVixIdxPrimary; primaryVisIdx++ {
		lineIdx := pglr.LineIndex[region][orientation][primaryVisIdx]
		for secondaryVisIdx := prevLastVisIdxSecondary; secondaryVisIdx > currLastVisIdxSecondary; secondaryVisIdx-- {
			if itemId, exists := lineIdx.P2[secondaryVisIdx]; exists {
				config := pglr.LineItems[region][orientation].ConfigLines[itemId]
				if config.SecondAxisStart > currLastVisIdxSecondary {
					flagger.Put(itemId)
				} else {
					pglr.updateLinePosition2(itemId, primaryVisIdx, currLastVisIdxSecondary, region, orientation)
				}
			}
		}
	}
}

func (pglr *PrimitiveGridLineRenderer) updateLinePosition1(itemId, primaryVisIdx, startVisSecondaryIdx int, region GridRegion, orientation LineOrientation) {

	lineItem := pglr.LineItems[region][orientation].ConfigLines[itemId]

	originalSecondaryStart := lineItem.SecondAxisStart

	lineItem.SecondAxisStart = startVisSecondaryIdx
	lineItem.UnPositioned = true

	pglr.LineItems[region][orientation].ConfigLines[itemId] = lineItem
	pglr.LineIndex[region][orientation][primaryVisIdx].P1[startVisSecondaryIdx] = itemId

	delete(pglr.LineIndex[region][orientation][primaryVisIdx].P1, originalSecondaryStart)
}

func (pglr *PrimitiveGridLineRenderer) updateLinePosition2(itemId, primaryVisIdx, endVisSecondaryIdx int, region GridRegion, orientation LineOrientation) {

	lineItem := pglr.LineItems[region][orientation].ConfigLines[itemId]

	originalSecondaryEnd := lineItem.SecondAxisEnd

	lineItem.SecondAxisEnd = endVisSecondaryIdx
	lineItem.UnPositioned = true

	pglr.LineItems[region][orientation].ConfigLines[itemId] = lineItem
	pglr.LineIndex[region][orientation][primaryVisIdx].P2[endVisSecondaryIdx] = itemId

	delete(pglr.LineIndex[region][orientation][primaryVisIdx].P1, originalSecondaryEnd)
}

func (pglr *PrimitiveGridLineRenderer) addNewLine(container *fyne.Container, primaryVisIdx, secondaryStartVisIdx, secondaryEndVisIdx int, region GridRegion, orientation LineOrientation) {
	var lineItem *canvas.Line

	managers := pglr.Managers
	flagger := managers[region][orientation].Flagger
	recycler := managers[region][orientation].Recycler

	newItem := LineConfig{
		PrimaryAxis:     primaryVisIdx,
		SecondAxisStart: secondaryStartVisIdx,
		SecondAxisEnd:   secondaryEndVisIdx,
		Orientation:     orientation,
		UnPositioned:    true,
	}

	if flaggedItemId, exist := flagger.Get(); exist {
		// Hot Swap available
		origItem := pglr.LineItems[region][orientation].ConfigLines[flaggedItemId]

		pglr.LineItems[region][orientation].ConfigLines[flaggedItemId] = newItem

		delete(pglr.LineIndex[region][orientation][origItem.PrimaryAxis].P1, origItem.SecondAxisStart)
		delete(pglr.LineIndex[region][orientation][origItem.PrimaryAxis].P2, origItem.SecondAxisEnd)

		pglr.LineIndex[region][orientation][primaryVisIdx].P1[secondaryStartVisIdx] = flaggedItemId
		pglr.LineIndex[region][orientation][primaryVisIdx].P2[secondaryEndVisIdx] = flaggedItemId
	} else {
		primitiveGridLineRecyleItem, recycledItem := recycler.Get()

		if recycledItem {
			// cold Swap available
			pglr.LineItems[region][orientation].ConfigLines[primitiveGridLineRecyleItem.id] = newItem

			pglr.LineIndex[region][orientation][primaryVisIdx].P1[secondaryStartVisIdx] = primitiveGridLineRecyleItem.id
			pglr.LineIndex[region][orientation][primaryVisIdx].P2[secondaryEndVisIdx] = primitiveGridLineRecyleItem.id
		} else {
			// New canvas Object
			lineItem = primitiveGridLineRecyleItem.obj

			lineItems := pglr.LineItems[region][orientation]

			lineItems.ConfigLines = append(lineItems.ConfigLines, newItem)
			lineItems.Lines = append(lineItems.Lines, primitiveGridLineRecyleItem.obj)

			pglr.LineItems[region][orientation] = lineItems

			itemId := len(lineItems.Lines) - 1

			pglr.LineIndex[region][orientation][primaryVisIdx].P1[secondaryStartVisIdx] = itemId
			pglr.LineIndex[region][orientation][primaryVisIdx].P2[secondaryEndVisIdx] = itemId

			// the only plavce where we add content
			container.Add(lineItem)
		}
	}
}

func (pglr *PrimitiveGridLineRenderer) isGridLineRequiredH(cache TransparencyCache, rowVisIdx, colVisIdx int) bool {
	mm := pglr.ctx.MergeManager

	id := CellID{Row: rowVisIdx, Col: colVisIdx}
	if mergeVisRange, exists := mm.GetMergedRangeByVisId(id); exists {
		if rowVisIdx == mergeVisRange.VisRowEnd {
			anchorVisIdx := CellID{mergeVisRange.VisRowStart, mergeVisRange.VisColStart}
			if isTransparent := mm.isMergeRangeTransparent(anchorVisIdx); !isTransparent {
				return false
			}
		} else {
			// in the merged range somewhere
			return false
		}
	} else {
		if needed, ok := cache[id]; ok {
			if !needed {
				return false
			}
		}
	}

	// check the other cell, bordering the gridline
	rowVisIdx++
	id = CellID{Row: rowVisIdx, Col: colVisIdx}
	if mergeVisRange, exists := mm.GetMergedRangeByVisId(id); exists {
		if rowVisIdx == mergeVisRange.VisRowStart {
			anchorVisIdx := CellID{mergeVisRange.VisRowStart, mergeVisRange.VisColStart}
			if isTransparent := mm.isMergeRangeTransparent(anchorVisIdx); !isTransparent {
				return false
			}
		} else {
			// in the merged range somewhere
			return false
		}
	} else {
		if needed, ok := cache[id]; ok {
			if !needed {
				return false
			}
		}
	}
	return true
}

func (pglr *PrimitiveGridLineRenderer) isGridLineRequiredV(cache TransparencyCache, rowVisIdx, colVisIdx int) bool {
	mm := pglr.ctx.MergeManager

	id := CellID{Row: rowVisIdx, Col: colVisIdx}
	if mergeVisRange, exists := mm.GetMergedRangeByVisId(id); exists {
		if rowVisIdx == mergeVisRange.VisRowEnd {
			anchorVisIdx := CellID{mergeVisRange.VisRowStart, mergeVisRange.VisColStart}
			if isTransparent := mm.isMergeRangeTransparent(anchorVisIdx); !isTransparent {
				return false
			}
		} else {
			// in the merged range somewhere
			return false
		}
	} else {
		if needed, ok := cache[id]; ok {
			if !needed {
				return false
			}
		}
	}

	// check the other cell, bordering the gridline
	rowVisIdx++
	id = CellID{Row: rowVisIdx, Col: colVisIdx}
	if mergeVisRange, exists := mm.GetMergedRangeByVisId(id); exists {
		if rowVisIdx == mergeVisRange.VisRowStart {
			anchorVisIdx := CellID{mergeVisRange.VisRowStart, mergeVisRange.VisColStart}
			if isTransparent := mm.isMergeRangeTransparent(anchorVisIdx); !isTransparent {
				return false
			}
		} else {
			// in the merged range somewhere
			return false
		}
	} else {
		if needed, ok := cache[id]; ok {
			if !needed {
				return false
			}
		}
	}
	return true
}

func (pglr *PrimitiveGridLineRenderer) renderEdge1(container *fyne.Container,
	primaryStart, primaryEnd int,
	start, end int,
	cache TransparencyCache,
	region GridRegion,
	orientation LineOrientation,
	fn func(isTransparent TransparencyCache, i, j int) bool) {
	var mode mode
	var currentVisIdx int
	var startVisIdx int
	var endVisIdx int

	for primaryVisIdx := primaryStart; primaryVisIdx <= primaryEnd; primaryVisIdx++ {
		if _, exist := pglr.LineIndex[region][orientation][primaryVisIdx]; !exist {
			pglr.LineIndex[region][orientation][primaryVisIdx] = lineIndex{
				P1: make(map[int]int),
				P2: make(map[int]int),
			}
		}

		startVisIdx = -1
		endVisIdx = -1
		currentVisIdx = start
		mode = MODE_SEARCHING
		for {
			if fn(cache, primaryVisIdx, currentVisIdx) {
				if mode == MODE_SEARCHING {
					startVisIdx = currentVisIdx
					mode = MODE_STARTING
				}
			} else if mode == MODE_STARTING {
				endVisIdx = currentVisIdx - 1
				pglr.addNewLine(container, primaryVisIdx, startVisIdx, endVisIdx, region, orientation)

				mode = MODE_SEARCHING
			}
			if currentVisIdx == end {
				if mode == MODE_STARTING {
					endVisIdx = currentVisIdx
					if itemId, exist := pglr.LineIndex[region][orientation][primaryVisIdx].P1[endVisIdx+1]; exist {
						pglr.updateLinePosition1(itemId, primaryVisIdx, startVisIdx, region, orientation)

					} else {
						pglr.addNewLine(container, primaryVisIdx, startVisIdx, endVisIdx, region, orientation)
					}
				}
				break
			}
			currentVisIdx++
		}
	}
}

func (pglr *PrimitiveGridLineRenderer) renderEdge2(container *fyne.Container,
	primaryStart, rowEnd int,
	start, end int,
	cache TransparencyCache,
	region GridRegion,
	orientation LineOrientation,
	fn func(isTransparent TransparencyCache, i, j int) bool) {

	var mode mode
	var currentVisIdx int
	var startVisIdx int
	var endVisIdx int

	for primaryVisIdx := primaryStart; primaryVisIdx <= rowEnd; primaryVisIdx++ {
		if _, exist := pglr.LineIndex[region][orientation][primaryVisIdx]; !exist {
			pglr.LineIndex[region][orientation][primaryVisIdx] = lineIndex{
				P1: make(map[int]int),
				P2: make(map[int]int),
			}
		}

		startVisIdx = -1
		endVisIdx = -1
		currentVisIdx = end
		mode = MODE_SEARCHING
		for {
			if fn(cache, primaryVisIdx, currentVisIdx) {
				if mode == MODE_SEARCHING {
					endVisIdx = currentVisIdx
					mode = MODE_STARTING
				}
			} else if mode == MODE_STARTING {
				startVisIdx = currentVisIdx + 1

				pglr.addNewLine(container, primaryVisIdx, startVisIdx, endVisIdx, region, orientation)

				mode = MODE_SEARCHING
			}
			if currentVisIdx == start {
				if mode == MODE_STARTING {
					startVisIdx = currentVisIdx
					//if itemId, exist := pglr.hLineIndex[region][primaryVisIdx].P2[startVisIdx-1]; exist {
					if itemId, exist := pglr.LineIndex[region][orientation][primaryVisIdx].P2[startVisIdx-1]; exist {
						pglr.updateLinePosition2(itemId, primaryVisIdx, endVisIdx, region, orientation)

					} else {
						pglr.addNewLine(container, primaryVisIdx, startVisIdx, endVisIdx, region, orientation)
					}
				}
				break
			}
			currentVisIdx--
		}
	}
}
