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

	// Use internal package

	l := internal.NewLoader()

	wrkbook, err := l.Load(file, "Income Statement")
	//wrkbook, err := l.Load(file, "")
	if err != nil {
		log.Fatal(err)
	}

	config := pkg.WidgetConfig{Toolbar: false}

	//widget := pkg.NewWorkBook(wrkbook, config)

	widget := pkg.NewWorkBookWidget(wrkbook, config)

	// That's it!
	a := app.NewWithID("ExcelViewer")
	a.Settings().SetTheme(&pkg.NoScrollShadowTheme{Theme: theme.DefaultTheme()})
	w := a.NewWindow("Excel Viewer")
	w.Resize(fyne.NewSize(1200, 800))
	w.SetContent(widget)
	w.ShowAndRun()

}
