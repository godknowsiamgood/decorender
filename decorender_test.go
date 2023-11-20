package decorender

import (
	"testing"
)

func TestFull(t *testing.T) {
	//width, height := 200, 200
	//img := image.NewRGBA(image.Rect(0, 0, width, height))
	//
	//// Define the colors with alpha for semi-transparency
	//red := color.RGBA{255, 0, 0, 10}    // Semi-transparent red
	//green := color.RGBA{0, 255, 0, 128} // Semi-transparent green
	//
	//// Draw the first rectangle (red)
	//draw.Draw(img, image.Rect(20, 20, 120, 120), &image.Uniform{red}, image.Point{}, draw.Src)
	//
	//// Draw the second rectangle (green), overlapping the first
	//draw.Draw(img, image.Rect(80, 80, 180, 180), &image.Uniform{green}, image.Point{}, draw.Over)
	//
	//// Save the image to a file
	//f, err := os.Create("rectangles.png")
	//if err != nil {
	//	panic(err)
	//}
	//defer f.Close()
	//
	//if err := png.Encode(f, img); err != nil {
	//	panic(err)
	//}

	d, err := NewRenderer("./test.yaml")
	if err != nil {
		t.Errorf("unexpected error while yaml parse: %v", err)
	}

	data := struct {
		StringsSlice []string
	}{
		StringsSlice: []string{"one", "two", "three", "four"},
	}

	err = d.RenderToFile(data, "test.png")
	if err != nil {
		t.Errorf("unexpected error while rendering: %v", err)
	}
}

func BenchmarkRender(b *testing.B) {
	d, _ := NewRenderer("./test.yaml")
	data := struct {
		StringsSlice []string
	}{
		StringsSlice: []string{"one", "two", "three", "four"},
	}
	for i := 0; i < b.N; i++ {
		_ = d.Render(data, EncodeFormatJPG, nil)
	}
}
