package decorender

import (
	"image/png"
	"os"
	"testing"
)

func TestFull(t *testing.T) {
	d, err := NewRenderer("./test.yaml")
	if err != nil {
		t.Errorf("unexpected error while yaml parse: %v", err)
	}

	image, err := d.Render(nil)
	if err != nil {
		t.Errorf("unexpected error while rendering: %v", err)
	}

	file, _ := os.Create("test.png")
	defer file.Close()
	_ = png.Encode(file, image)
}
