package decorender

import (
	"os"
	"testing"
)

func TestFull(t *testing.T) {
	d, err := NewRenderer("./test.yaml", &Options{LocalFiles: os.DirFS(".")})
	if err != nil {
		t.Errorf("unexpected error while yaml parse: %v", err)
		return
	}

	data := struct {
		StringsSlice []string
	}{
		StringsSlice: []string{"one", "two", "three", "four"},
	}

	err = d.RenderToFile(data, "test.png", &RenderOptions{
		UseSample: false,
	})

	if err != nil {
		t.Errorf("unexpected error while rendering: %v", err)
	}
}

func BenchmarkRender(b *testing.B) {
	d, _ := NewRenderer("./cmd/decorender_server/bilets/bilets.yaml", &Options{
		LocalFiles: os.DirFS("cmd/decorender_server"),
	})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = d.RenderAndWrite(nil, EncodeFormatJPG, nil, &RenderOptions{UseSample: true})
		}
	})
}
