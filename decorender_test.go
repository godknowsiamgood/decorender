package decorender

import (
	"testing"
)

func TestFull(t *testing.T) {
	d, err := NewRenderer("./test.yaml")
	if err != nil {
		t.Errorf("unexpected error while yaml parse: %v", err)
	}

	data := struct {
		A            int
		StringsSlice []string
	}{
		A:            44,
		StringsSlice: []string{"one", "two", "three", "four"},
	}

	err = d.RenderToFile(data, "test.png", &Options{UseSample: false})

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
		_ = d.Render(data, EncodeFormatJPG, nil, nil)
	}
}
