package pkg

import (
	"io"

	"fyne.io/fyne/v2/widget"
)

func (wb WorkBookData) WorkSheetItem(item string) *WorkSheetData {

	if sh, ok := wb.WorkSheetsData[item]; ok {
		return sh
	}
	return nil
}

// This is the only Widget
type WorkbookWidget struct {
	widget.BaseWidget

	data *WorkBookData

	// UI State
	currentSheet         string
	toolbar              bool
	currentRenderer      *GridRenderer
	sheetDisplaySettings map[string]*SheetDisplaySettings

	selection *SelectionState
}

func NewWorkBookWidget(data *WorkBookData, config WidgetConfig) *WorkbookWidget {

	w := &WorkbookWidget{
		data:      data,
		toolbar:   config.Toolbar,
		selection: NewSelectionState(),
	}
	w.ExtendBaseWidget(w)
	return w
}

type WidgetConfig struct {
	Toolbar bool
}

type WorkbookLoader interface {
	Load(r io.Reader, fileName string) (*WorkBookData, error)
}
