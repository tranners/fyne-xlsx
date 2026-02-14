package main

import (
	"log"
	"os"

	// Import public pkg
	"github.com/tranners/fyne-xlsx/pkg"

	// Import private internal
	"github.com/tranners/fyne-xlsx/internal"
	// External dependency
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
)

func main() {
	filename := "specimen.xlsx"
	file, _ := os.Open(filename)
	defer file.Close()

	l := internal.NewLoader()

	wrkbook, err := l.Load(file, "")
	if err != nil {
		log.Fatal(err)
	}

	config := pkg.WidgetConfig{Toolbar: false}

	widget := pkg.NewWorkBookWidget(wrkbook, config)

	a := app.NewWithID("ExcelViewer")
	w := a.NewWindow("Excel Viewer")
	a.Settings().SetTheme(&pkg.NoScrollShadowTheme{Theme: theme.DefaultTheme()})
	w.Resize(fyne.NewSize(1200, 800))
	w.SetContent(widget)
	w.ShowAndRun()

}
