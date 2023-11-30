package layout

import (
	"fmt"
	"github.com/godknowsiamgood/decorender/utils"
	"github.com/stretchr/testify/assert"
	"image/color"
	"testing"
)

func TestParseBorderProperty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected utils.Border
		err      error
	}{
		{
			name:  "Valid input with all properties",
			input: "2.5 red inset",
			expected: utils.Border{
				Type:  utils.BorderTypeInset,
				Width: 2.5,
				Color: color.RGBA{R: 255, A: 255},
			},
			err: nil,
		},
		{
			name:  "Valid input with width and color",
			input: "1.0 blue",
			expected: utils.Border{
				Width: 1.0,
				Color: color.RGBA{B: 255, A: 255},
				Type:  utils.BorderTypeOutset,
			},
			err: nil,
		},
		{
			name:  "Valid input with type only",
			input: "outset",
			expected: utils.Border{
				Type: utils.BorderTypeOutset,
			},
			err: nil,
		},
		{
			name:  "Invalid input with duplicate width",
			input: "1.0 2.0 blue",
			err:   fmt.Errorf("trying to specify border width 2, but width is already set"),
		},
		{
			name:  "Invalid input with duplicate color",
			input: "red green 1.0",
			err:   fmt.Errorf("trying to specify border color {0 128 0 255}, but color is already set"),
		},
		{
			name:  "Invalid input with unknown token",
			input: "1.0 blue dashed",
			err:   fmt.Errorf("unknown token dashed in border property"),
		},
		{
			name:  "Empty input",
			input: "",
		},
		{
			name:  "Invalid numeric value",
			input: "abc red inset",
			err:   fmt.Errorf("unknown token abc in border property"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseBorderProperty(tt.input)
			if tt.err != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
