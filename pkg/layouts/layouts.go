package layouts

import (
	"fyne.io/fyne/v2"
)

type FixedSizeLayout struct {
	size fyne.Size
}

func NewFixedSizeLayout(size fyne.Size) *FixedSizeLayout {
	return &FixedSizeLayout{size: size}
}

func (f *FixedSizeLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return f.size
}

func (f *FixedSizeLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, obj := range objects {
		obj.Resize(f.size)
		obj.Move(fyne.NewPos(0, 0))
	}
	//fmt.Printf("[LAYOUT] Called @ %s size=%v objects=%v\n", time.Now().Format("15:04:05.000"), size, len(objects))
}
