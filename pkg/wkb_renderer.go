package pkg

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

type workbookRenderer struct {
	widget *WorkbookWidget

	// UI components
	fontmanager     *FontManager
	tabs            *container.AppTabs
	toolbar         *fyne.Container
	gridRenderers   map[string]*GridRenderer
	currentRenderer string
}

func (w *WorkbookWidget) CreateRenderer() fyne.WidgetRenderer {

	fontManager := NewFontManager()

	gridRenderers := make(map[string]*GridRenderer)

	for shtName, shtData := range w.data.WorkSheetsData {
		gridRenderers[shtName] = NewGridRenderer(shtData, fontManager)
	}

	tabs := container.NewAppTabs()
	tabs.SetTabLocation(container.TabLocationBottom)

	// Create a tab for each sheet
	for _, sheetName := range w.data.WorkSheetList {
		if gridRenderer, exists := gridRenderers[sheetName]; exists {
			// Create scroll container for the grid
			content := gridRenderer.GetContainer()
			tabs.Append(container.NewTabItem(sheetName, content))
		}
	}

	// Set initial current sheet
	if len(w.data.WorkSheetList) > 0 {
		w.currentSheet = w.data.WorkSheetList[0]
	}

	// Handle tab switching
	tabs.OnSelected = func(tab *container.TabItem) {
		w.currentSheet = tab.Text
		if renderer, ok := gridRenderers[tab.Text]; ok {
			renderer.UpdateScrollContentSize()
		}
	}

	// Also set the initial sheet's renderer
	if len(w.data.WorkSheetList) > 0 {
		w.currentSheet = w.data.WorkSheetList[0]
		w.currentRenderer = gridRenderers[w.currentSheet] // ← Set initial
	}

	r := &workbookRenderer{
		widget:        w,
		fontmanager:   fontManager,
		gridRenderers: gridRenderers,
		tabs:          tabs,
	}

	return r
}

func (r *workbookRenderer) Refresh() {
	if currentGrid := r.gridRenderers[r.widget.currentSheet]; currentGrid != nil {
		currentGrid.Render()
	}
}

func (r *workbookRenderer) Layout(size fyne.Size) {
	r.tabs.Resize(size)
	r.tabs.Move(fyne.NewPos(0, 0))
	if currentGrid := r.gridRenderers[r.widget.currentSheet]; currentGrid != nil {
		currentGrid.OnResize(size)

	}
}

func (r *workbookRenderer) Destroy() {}

func (r *workbookRenderer) MinSize() fyne.Size {
	return fyne.NewSize(200, 200)
}

func (r *workbookRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.tabs}
}
