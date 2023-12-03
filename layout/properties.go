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

func calculateProperties(n parsing.Node, context layoutPhaseContext, data any, parentData any, currentValueIndex int) CalculatedProperties {
	padding, _ := parseNValues(n.Padding, 4, context.size.W, context.size.H, data, parentData, currentValueIndex, false, false)
	lineHeight, _ := parseNValues(n.LineHeight, 1, context.size.W, context.size.H, data, parentData, currentValueIndex, false, false)
	borderRadius, _ := parseNValues(n.BorderRadius, 4, context.size.W, context.size.H, data, parentData, currentValueIndex, false, false)

	sz, szErr := parseNValues(n.Size, 2, context.size.W, context.size.H, data, parentData, currentValueIndex, false, false)
	width, widthErr := parseNValues(n.Width, 1, context.size.W, context.size.H, data, parentData, currentValueIndex, true, false)
	height, heightErr := parseNValues(n.Height, 1, context.size.W, context.size.H, data, parentData, currentValueIndex, false, false)
	if szErr != nil {
		sz[0], sz[1] = -1, -1
	}
	if widthErr == nil {
		sz[0] = width[0]
	}
	if heightErr == nil {
		sz[1] = height[0]
	}

	anchors := parseAnchors(n.Absolute, data, parentData, currentValueIndex)
	if anchors.HasTop() && anchors.HasBottom() {
		sz[1] = context.size.H - anchors.Top() - anchors.Bottom()
	}
	if anchors.HasLeft() && anchors.HasRight() {
		sz[0] = context.size.W - anchors.Left() - anchors.Right()
	}

	backgroundColor := color.RGBA{A: 0}
	if n.BkgColor != "" {
		backgroundColor, _ = parseColor(utils.ReplaceWithValuesUnsafe(n.BkgColor, data, parentData, currentValueIndex))
	}

	bkgImageSize := validateStringValue(n.BkgImageSize, []string{"cover", "contain"})

	fontColor := context.props.FontColor // inherited
	if n.FontColor != "" {
		fontColor, _ = parseColor(utils.ReplaceWithValuesUnsafe(n.FontColor, data, parentData, currentValueIndex))
	}
	if n.Color != "" {
		fontColor, _ = parseColor(utils.ReplaceWithValuesUnsafe(n.Color, data, parentData, currentValueIndex))
	}

	fontDescription := context.props.FontDescription // inherited
	fontDescription = parseFontString(n.Font, fontDescription, context.size.W, context.size.H, data, parentData, currentValueIndex)
	if n.FontFamily != "" {
		fontDescription.Family = utils.ReplaceWithValuesUnsafe(n.FontFamily, data, parentData, currentValueIndex)
	}
	if n.FontSize != "" {
		v, err := parseNValues(n.FontSize, 1, context.size.W, context.size.H, data, parentData, currentValueIndex, true, false)
		if err == nil {
			fontDescription.Size = v[0]
		}
	}
	if n.FontWeight != "" {
		v, err := parseNValues(n.FontWeight, 1, context.size.W, context.size.H, data, parentData, currentValueIndex, true, false)
		if err == nil {
			fontDescription.Weight = int(v[0])
		}
	}
	if n.FontStyle != "" {
		fontDescription.Style = lo.Ternary(utils.ReplaceWithValuesUnsafe(n.FontStyle, data, parentData, currentValueIndex) == "italic", font.StyleItalic, font.StyleNormal)
	}

	childrenDirection := validateStringValue(utils.ReplaceWithValuesUnsafe(n.InnerDirection, data, parentData, currentValueIndex), []string{"column", "row"})
	childrenJustify := validateStringValue(utils.ReplaceWithValuesUnsafe(n.Justify, data, parentData, currentValueIndex), []string{"start", "center", "end", "space-between", "space-evenly"})
	childrenColumnAlign := validateStringValue(utils.ReplaceWithValuesUnsafe(n.ChildrenColumnAlign, data, parentData, currentValueIndex), []string{"left", "center", "right"})
	childrenWrap := validateStringValue(utils.ReplaceWithValuesUnsafe(n.ChildrenWrap, data, parentData, currentValueIndex), []string{"wrap", "none"})

	innerGap, _ := parseNValues(n.InnerGap, 1, context.size.W, context.size.H, data, parentData, currentValueIndex, true, false)

	rotation, _ := parseNValues(n.Rotation, 1, context.size.W, context.size.H, data, parentData, currentValueIndex, true, true)

	border, _ := parseBorderProperty(utils.ReplaceWithValuesUnsafe(n.Border, data, parentData, currentValueIndex))

	offsetAnchors := parseAnchors(n.Offset, data, parentData, currentValueIndex)

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
		Padding:                utils.TopRightBottomLeft{padding[0], padding[1], padding[2], padding[3]},
		FontDescription:        fontDescription,
		BorderRadius:           borderRadius,
		AbsolutePosition:       anchors,
		InnerGap:               innerGap[0],
		Rotation:               rotation[0],
		BkgImageSize:           lo.Ternary(bkgImageSize == "contain", BkgImageSizeContain, BkgImageSizeCover),
		Border:                 border,
		Offset:                 utils.TopRightBottomLeft{offsetAnchors.Top(), offsetAnchors.Right(), offsetAnchors.Bottom(), offsetAnchors.Left()},
	}
}

func parseAnchors(value string, data any, parentValue any, currentValueIndex int) (result utils.AbsolutePosition) {
	value = utils.ReplaceWithValuesUnsafe(value, data, parentValue, currentValueIndex)
	tokens := strings.Fields(value)
	for _, token := range tokens {
		tokenParts := strings.Split(token, "/")

		var direction string
		var offset float64
		if len(tokenParts) > 0 {
			direction = tokenParts[0]
		}
		if len(tokenParts) > 1 {
			offset, _ = strconv.ParseFloat(tokenParts[1], 64)
		}

		switch direction {
		case "top":
			result[0] = utils.AbsolutePos{Has: true, Offset: offset}
		case "right":
			result[1] = utils.AbsolutePos{Has: true, Offset: offset}
		case "bottom":
			result[2] = utils.AbsolutePos{Has: true, Offset: offset}
		case "left":
			result[3] = utils.AbsolutePos{Has: true, Offset: offset}
		}
	}
	return result
}

func parseBorderProperty(value string) (res utils.Border, err error) {
	tokens := strings.Fields(value)

	var widthIsSet bool
	var colorIsSet bool

	for _, t := range tokens {
		width, err := strconv.ParseFloat(t, 64)
		if err == nil {
			if widthIsSet {
				return res, fmt.Errorf("trying to specify border width %v, but width is already set", width)
			}
			widthIsSet = true
			res.Width = width
			continue
		}

		c, err := parseColor(t)
		if err == nil {
			if colorIsSet {
				return res, fmt.Errorf("trying to specify border color %v, but color is already set", c)
			}
			colorIsSet = true
			res.Color = c
			continue
		}

		switch t {
		case "inset":
			res.Type = utils.BorderTypeInset
		case "outset":
			res.Type = utils.BorderTypeOutset
		case "center":
			res.Type = utils.BorderTypeCenter
		default:
			return res, fmt.Errorf("unknown token %v in border property", t)
		}
	}

	return res, nil
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

var valuesEmptyErr = errors.New("values empty")
var valuesParseErr = errors.New("values format not correct")

func parseNValues(str string, max int, parentWidth float64, parentHeight float64, data any, parentData any, currentValueIndex int, relativeToWidth bool, allowNegative bool) (utils.FourValues, error) {
	var result utils.FourValues

	if str == "" {
		return result, valuesEmptyErr
	}

	str = utils.ReplaceWithValuesUnsafe(str, data, parentData, currentValueIndex)

	matches := parseValueRegex.FindAllStringSubmatch(str, -1)
	if len(matches) > max || len(matches) == 0 {
		return result, valuesParseErr
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

var hexRegex = regexp.MustCompile(`^0x([a-fA-F0-9]{6})([a-fA-F0-9]{2})?$`)
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

func parseFontString(prop string, fd fonts.FaceDescription, parentWidth float64, parentHeight float64, data any, parentData any, currentValueIndex int) fonts.FaceDescription {
	if prop == "" {
		return fd
	}

	prop = utils.ReplaceWithValuesUnsafe(prop, data, parentData, currentValueIndex)
	prop = strings.ReplaceAll(prop, ",", " ")

	isSizeSet := false

	tokens := strings.Fields(prop)
	for _, token := range tokens {
		v, err := parseNValues(token, 1, parentWidth, parentHeight, data, parentData, currentValueIndex, true, false)
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
