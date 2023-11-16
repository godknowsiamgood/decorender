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
		StringsSlice []string
	}{
		StringsSlice: []string{"one", "two", "three", "four"},
	}

	err = d.RenderToFile(data, "test.png")
	if err != nil {
		t.Errorf("unexpected error while rendering: %v", err)
	}
}
