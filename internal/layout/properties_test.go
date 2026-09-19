package layout

import (
	"fmt"
	"github.com/godknowsiamgood/decorender/internal/utils"
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
			input: "1.0 blue wiggly",
			err:   fmt.Errorf("unknown token wiggly in border property"),
		},
		{
			name:  "Dash pattern defaults to the border width",
			input: "2 blue dashed",
			expected: utils.Border{
				Width:     2,
				Color:     color.RGBA{B: 255, A: 255},
				Type:      utils.BorderTypeOutset,
				Dashes:    [utils.MaxBorderDashes]float64{8, 8},
				DashCount: 2,
			},
		},
		{
			name:  "Dash pattern with its own lengths, in any token order",
			input: "dashed/10/4/2/4 red 1",
			expected: utils.Border{
				Width:     1,
				Color:     color.RGBA{R: 255, A: 255},
				Type:      utils.BorderTypeOutset,
				Dashes:    [utils.MaxBorderDashes]float64{10, 4, 2, 4},
				DashCount: 4,
			},
		},
		{
			name:  "Sides accumulate",
			input: "1 red top left",
			expected: utils.Border{
				Width: 1,
				Color: color.RGBA{R: 255, A: 255},
				Type:  utils.BorderTypeOutset,
				Sides: utils.BorderSideTop | utils.BorderSideLeft,
			},
		},
		{
			name:  "Dash pattern with a length that is not a number",
			input: "1 red dashed/4/x",
			err:   fmt.Errorf("dash length \"x\" in dashed/4/x is not a positive number"),
		},
		{
			name:  "Dash pattern of zero length",
			input: "1 red dashed/0/0",
			err:   fmt.Errorf("dash pattern of zero length draws nothing"),
		},
		{
			name:  "Dash pattern given twice",
			input: "1 red dashed dashed/4",
			err:   fmt.Errorf("trying to specify dash pattern dashed/4, but it is already set"),
		},
		{
			name:  "Dash pattern longer than the cap",
			input: "1 red dashed/1/1/1/1/1/1/1/1/1",
			err:   fmt.Errorf("dash pattern dashed/1/1/1/1/1/1/1/1/1 has more than 8 lengths"),
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
