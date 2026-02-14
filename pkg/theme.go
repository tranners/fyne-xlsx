package pkg

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type NoScrollShadowTheme struct {
	fyne.Theme
}

func (t *NoScrollShadowTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// Make all shadow colors transparent
	if name == theme.ColorNameShadow {
		return color.Transparent
	}
	// Delegate everything else to default theme
	return theme.DefaultTheme().Color(name, variant)
}
