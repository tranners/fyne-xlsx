package main

import (
	"log"
	"os"

	"github.com/tranners/fyne-xlsx/pkg"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
	"github.com/tranners/fyne-xlsx/internal"
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
