package layout

import (
	"errors"
	"fmt"
	"github.com/godknowsiamgood/decorender/fonts"
	"github.com/godknowsiamgood/decorender/parsing"
	"github.com/godknowsiamgood/decorender/utils"
	"github.com/samber/lo"
	"golang.org/x/image/font"
	"image/color"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const (
	unitAbs = iota
	unitPercent
	unitWidth
	unitHeight
)

func calculateProperties(n parsing.Node, context layoutPhaseContext, data any) CalculatedProperties {
	padding, _ := parseNValues(n.Padding, 4, context.size.W, context.size.H, data, false, false)
	lineHeight, _ := parseNValues(n.LineHeight, 1, context.size.W, context.size.H, data, true, false)
	borderRadius, _ := parseNValues(n.BorderRadius, 4, context.size.W, context.size.H, data, false, false)

	sz, szErr := parseNValues(n.Size, 2, context.size.W, context.size.H, data, false, false)
	width, widthErr := parseNValues(n.Width, 1, context.size.W, context.size.H, data, false, false)
	height, heightErr := parseNValues(n.Height, 1, context.size.W, context.size.H, data, true, false)
	if szErr != nil {
		sz[0], sz[1] = -1, -1
	}
	if widthErr == nil {
		sz[0] = width[0]
	}
	if heightErr == nil {
		sz[1] = height[0]
	}

	anchors := parseAbsoluteAnchor(n.Absolute, data)
	if anchors[0] && anchors[2] {
		sz[1] = context.size.H
	}
	if anchors[1] && anchors[3] {
		sz[0] = context.size.W
	}

	backgroundColor := color.RGBA{A: 0}
	if n.BkgColor != "" {
		backgroundColor, _ = parseColor(utils.ReplaceWithValues(n.BkgColor, data))
	}

	fontColor := context.props.FontColor // inherited
	if n.FontColor != "" {
		fontColor, _ = parseColor(utils.ReplaceWithValues(n.FontColor, data))
	}
	if n.Color != "" {
		fontColor, _ = parseColor(utils.ReplaceWithValues(n.Color, data))
	}

	fontDescription := context.props.FontDescription // inherited
	fontDescription = parseFontString(n.Font, fontDescription, context.size.W, context.size.H, data)
	if n.FontFamily != "" {
		fontDescription.Family = utils.ReplaceWithValues(n.FontFamily, data)
	}
	if n.FontSize != "" {
		v, err := parseNValues(n.FontSize, 1, context.size.W, context.size.H, data, true, false)
		if err == nil {
			fontDescription.Size = v[0]
		}
	}
	if n.FontWeight != "" {
		v, err := parseNValues(n.FontWeight, 1, context.size.W, context.size.H, data, true, false)
		if err == nil {
			fontDescription.Weight = int(v[0])
		}
	}
	if n.FontStyle != "" {
		fontDescription.Style = lo.Ternary(utils.ReplaceWithValues(n.FontStyle, data) == "italic", font.StyleItalic, font.StyleNormal)
	}

	childrenDirection := validateStringValue(utils.ReplaceWithValues(n.InnerDirection, data), []string{"column", "row"})
	childrenJustify := validateStringValue(utils.ReplaceWithValues(n.Justify, data), []string{"start", "center", "end", "space-between", "space-evenly"})
	childrenColumnAlign := validateStringValue(utils.ReplaceWithValues(n.ChildrenColumnAlign, data), []string{"left", "center", "right"})
	childrenWrap := validateStringValue(utils.ReplaceWithValues(n.ChildrenWrap, data), []string{"wrap", "none"})

	innerGap, _ := parseNValues(n.InnerGap, 1, context.size.W, context.size.H, data, true, false)

	rotation, _ := parseNValues(n.Rotation, 1, context.size.W, context.size.H, data, true, true)

	if n.Text != "" {
		childrenDirection = "row"
	}

	return CalculatedProperties{
		Size:                   utils.Size{W: sz[0], H: sz[1]},
		BkgColor:               backgroundColor,
		FontColor:              fontColor,
		ChildAlign:             "",
		IsChildrenDirectionRow: childrenDirection == "row",
		Justify:                childrenJustify,
		ChildrenColumnAlign:    childrenColumnAlign,
		IsWrappingEnabled:      childrenWrap == "wrap",
		LineHeight:             lo.Ternary(n.LineHeight == "", context.props.LineHeight, lineHeight[0]),
		Padding:                padding,
		FontDescription:        fontDescription,
		BorderRadius:           borderRadius,
		Anchors:                anchors,
		InnerGap:               innerGap[0],
		Rotation:               rotation[0],
	}
}

func parseAbsoluteAnchor(value string, data any) (result utils.Anchors) {
	value = utils.ReplaceWithValues(value, data)
	tokens := strings.Fields(value)
	for _, token := range tokens {
		switch token {
		case "top":
			result[0] = true
		case "right":
			result[1] = true
		case "bottom":
			result[2] = true
		case "left":
			result[3] = true
		}
	}
	return result
}

func prepareParsedValue(value float64, isVertical bool, unit int, parentWidth float64, parentHeight float64) float64 {
	switch unit {
	case unitAbs:
		return value
	case unitPercent:
		value /= 100.0
		if isVertical {
			return value * parentHeight
		} else {
			return value * parentWidth
		}
	case unitWidth:
		return value * parentWidth
	case unitHeight:
		return value * parentHeight
	}

	return 0
}

var parseValueRegex = regexp.MustCompile(`(?i)(-?\d+(\.\d+)?)(%|w|h|)`)

func parseNValues(str string, max int, parentWidth float64, parentHeight float64, data any, relativeToWidth bool, allowNegative bool) (utils.FourValues, error) {
	str = utils.ReplaceWithValues(str, data)

	var result utils.FourValues

	matches := parseValueRegex.FindAllStringSubmatch(str, -1)
	if len(matches) > max || len(matches) == 0 {
		return result, errors.New("failed to parse values")
	}

	for i, match := range matches {
		val, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			return result, err
		}

		if !allowNegative && val < 0 {
			val = -val
		}

		unit := unitAbs
		switch strings.ToLower(match[3]) {
		case "%":
			unit = unitPercent
		case "w":
			unit = unitWidth
		case "h":
			unit = unitHeight
		}

		isVertical := i%2 == 1
		if max == 1 {
			isVertical = !relativeToWidth
		}

		result[i] = prepareParsedValue(val, isVertical, unit, parentWidth, parentHeight)
	}

	if len(matches) == 1 {
		result[1] = result[0]
		result[2] = result[0]
		result[3] = result[0]
	}

	if len(matches) == 2 {
		result[2] = result[0]
		result[3] = result[1]
	}

	if len(matches) == 3 {
		result[3] = result[1]
	}

	return result, nil
}

var hexRegex = regexp.MustCompile(`^#([a-fA-F0-9]{6})([a-fA-F0-9]{2})?$`)
var rgbRegex = regexp.MustCompile(`^rgb(a)?\((\d{1,3}),\s*(\d{1,3}),\s*(\d{1,3})(,\s*(0|1|0?\.\d+))?\)$`)

func parseColor(c string) (color.RGBA, error) {
	c = strings.ToLower(strings.TrimSpace(c))

	if c == "" {
		return color.RGBA{A: 255}, nil
	}

	for _, cc := range utils.PredefinedColors {
		if cc.Name == c {
			return cc.Color, nil
		}
	}

	if matches := hexRegex.FindStringSubmatch(c); matches != nil {
		r, _ := strconv.ParseInt(matches[1][0:2], 16, 64)
		g, _ := strconv.ParseInt(matches[1][2:4], 16, 64)
		b, _ := strconv.ParseInt(matches[1][4:6], 16, 64)
		a := int64(255)
		if matches[2] != "" {
			a, _ = strconv.ParseInt(matches[2], 16, 64)
		}
		if a <= 255 && r <= 255 && g <= 255 && b <= 255 {
			return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}, nil
		}
	}

	if matches := rgbRegex.FindStringSubmatch(c); matches != nil {
		r, _ := strconv.Atoi(matches[2])
		g, _ := strconv.Atoi(matches[3])
		b, _ := strconv.Atoi(matches[4])
		a := 255.0
		if matches[5] != "" {
			a, _ = strconv.ParseFloat(matches[6], 64)
			a *= 255
		}
		if a <= 255 && r <= 255 && g <= 255 && b <= 255 {
			return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(math.Floor(a))}, nil
		}
	}

	return color.RGBA{}, fmt.Errorf("error parsing color \"%s\"", c)
}

func validateStringValue(v string, options []string) string {
	for _, o := range options {
		if o == v {
			return v
		}
	}
	return options[0]
}

func parseFontString(prop string, fd fonts.FaceDescription, parentWidth float64, parentHeight float64, data any) fonts.FaceDescription {
	if prop == "" {
		return fd
	}

	prop = utils.ReplaceWithValues(prop, data)
	prop = strings.ReplaceAll(prop, ",", " ")

	isSizeSet := false

	tokens := strings.Fields(prop)
	for _, token := range tokens {
		v, err := parseNValues(token, 1, parentWidth, parentHeight, data, true, false)
		if err != nil {
			if token == "italic" {
				fd.Style = font.StyleItalic
			} else if token == "normal" {
				fd.Style = font.StyleNormal
			} else {
				fd.Family = token
			}
		} else {
			if !isSizeSet {
				fd.Size = v[0]
				isSizeSet = true
			} else {
				fd.Weight = int(v[0])
			}
		}
	}

	return fd
}
