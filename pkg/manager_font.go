package pkg

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

// Embed font files (you'll need to download Carlito fonts and place in fonts/ directory)

//go:embed fonts/Carlito-Bold.ttf
var carlitoBoldBytes []byte

//go:embed fonts/Carlito-Regular.ttf
var carlitoRegularBytes []byte

//go:embed fonts/Carlito-Italic.ttf
var carlitoItalicBytes []byte

//go:embed fonts/Carlito-BoldItalic.ttf
var carlitoBoldItalicBytes []byte

// FontManager handles font resources for the application
type FontManager struct {
	Regular    fyne.Resource
	Bold       fyne.Resource
	Italic     fyne.Resource
	BoldItalic fyne.Resource
}

// NewFontManager creates and initializes the font manager
func NewFontManager() *FontManager {
	return &FontManager{
		Regular:    fyne.NewStaticResource("Carlito-Regular.ttf", carlitoRegularBytes),
		Bold:       fyne.NewStaticResource("Carlito-Bold.ttf", carlitoBoldBytes),
		Italic:     fyne.NewStaticResource("Carlito-Italic.ttf", carlitoItalicBytes),
		BoldItalic: fyne.NewStaticResource("Carlito-BoldItalic.ttf", carlitoBoldItalicBytes),
	}
}

// SelectFont returns the appropriate font resource based on style flags
func (fm *FontManager) SelectFont(bold, italic bool) fyne.Resource {
	switch {
	case bold && italic:
		return fm.BoldItalic
	case bold:
		return fm.Bold
	case italic:
		return fm.Italic
	default:
		return fm.Regular
	}
}
