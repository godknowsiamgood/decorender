package layout

import (
	"errors"
	"fmt"
	"github.com/godknowsiamgood/decorender/internal/fonts"
	"github.com/godknowsiamgood/decorender/internal/parsing"
	"github.com/godknowsiamgood/decorender/internal/utils"
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

// calculateProperties is currently ugly function that needs refactoring.
// Maybe we should introduce some fields generic configuration.
func calculateProperties(n parsing.Node, context layoutPhaseContext, data any, parentData any, currentValueIndex int) CalculatedProperties {
	padding, _ := parseNValues(n.Padding, 4, context.size.W, context.size.H, data, parentData, currentValueIndex, false, false, context.cache)
	lineHeight, _ := parseNValues(n.LineHeight, 1, context.size.W, context.size.H, data, parentData, currentValueIndex, false, false, context.cache)
	borderRadius, _ := parseNValues(n.BorderRadius, 4, context.size.W, context.size.H, data, parentData, currentValueIndex, false, false, context.cache)

	sz, szErr := parseNValues(n.Size, 2, context.size.W, context.size.H, data, parentData, currentValueIndex, false, false, context.cache)
	width, widthErr := parseNValues(n.Width, 1, context.size.W, context.size.H, data, parentData, currentValueIndex, true, false, context.cache)
	height, heightErr := parseNValues(n.Height, 1, context.size.W, context.size.H, data, parentData, currentValueIndex, false, false, context.cache)
	if szErr != nil {
		sz[0], sz[1] = -1, -1
	}
	if widthErr == nil {
		sz[0] = width[0]
	}
	if heightErr == nil {
		sz[1] = height[0]
	}

	anchors := parseAnchors(n.Absolute, data, parentData, currentValueIndex, context.cache)
	if anchors.HasTop() && anchors.HasBottom() {
		sz[1] = context.size.H - anchors.Top() - anchors.Bottom()
	}
	if anchors.HasLeft() && anchors.HasRight() {
		sz[0] = context.size.W - anchors.Left() - anchors.Right()
	}

	backgroundColor := color.RGBA{A: 0}
	if n.BkgColor != "" {
		backgroundColor, _ = parseColor(replaceWithValuesUnsafe(n.BkgColor, data, parentData, currentValueIndex, context.cache))
	}

	bkgImageSize := validateStringValue(n.BkgImageSize, []string{"cover", "contain"})

	fontColor := context.props.FontColor // inherited
	if n.FontColor != "" {
		fontColor, _ = parseColor(replaceWithValuesUnsafe(n.FontColor, data, parentData, currentValueIndex, context.cache))
	}
	if n.Color != "" {
		fontColor, _ = parseColor(replaceWithValuesUnsafe(n.Color, data, parentData, currentValueIndex, context.cache))
	}

	fontDescription := context.props.FontDescription // inherited
	fontDescription = parseFontString(n.Font, fontDescription, context.size.W, context.size.H, data, parentData, currentValueIndex, context.cache)
	if n.FontFamily != "" {
		fontDescription.Family = replaceWithValuesUnsafe(n.FontFamily, data, parentData, currentValueIndex, context.cache)
	}
	if n.FontSize != "" {
		v, err := parseNValues(n.FontSize, 1, context.size.W, context.size.H, data, parentData, currentValueIndex, true, false, context.cache)
		if err == nil {
			fontDescription.Size = v[0]
		}
	}
	if n.FontWeight != "" {
		v, err := parseNValues(n.FontWeight, 1, context.size.W, context.size.H, data, parentData, currentValueIndex, true, false, context.cache)
		if err == nil {
			fontDescription.Weight = int(v[0])
		}
	}
	if n.FontStyle != "" {
		fontDescription.Style = font.StyleNormal
		if replaceWithValuesUnsafe(n.FontStyle, data, parentData, currentValueIndex, context.cache) == "italic" {
			fontDescription.Style = font.StyleItalic
		}
	}

	childrenDirection := validateStringValue(replaceWithValuesUnsafe(n.InnerDirection, data, parentData, currentValueIndex, context.cache), []string{"column", "row"})
	childrenJustify := validateStringValue(replaceWithValuesUnsafe(n.Justify, data, parentData, currentValueIndex, context.cache), []string{"start", "center", "end", "space-between", "space-evenly"})
	childrenColumnAlign := validateStringValue(replaceWithValuesUnsafe(n.ChildrenColumnAlign, data, parentData, currentValueIndex, context.cache), []string{"left", "center", "right"})
	childrenWrap := validateStringValue(replaceWithValuesUnsafe(n.ChildrenWrap, data, parentData, currentValueIndex, context.cache), []string{"wrap", "none"})

	innerGap, _ := parseNValues(n.InnerGap, 1, context.size.W, context.size.H, data, parentData, currentValueIndex, true, false, context.cache)

	rotation, _ := parseNValues(n.Rotation, 1, context.size.W, context.size.H, data, parentData, currentValueIndex, true, true, context.cache)

	border, _ := parseBorderProperty(replaceWithValuesUnsafe(n.Border, data, parentData, currentValueIndex, context.cache))

	offsetAnchors := parseAnchors(n.Offset, data, parentData, currentValueIndex, context.cache)

	if n.Text != "" {
		childrenDirection = "row"
	}

	resolvedLineHeight := lineHeight[0]
	if n.LineHeight == "" {
		resolvedLineHeight = context.props.LineHeight
	}

	resolvedBkgImageSize := BkgImageSizeCover
	if bkgImageSize == "contain" {
		resolvedBkgImageSize = BkgImageSizeContain
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
		LineHeight:             resolvedLineHeight,
		Padding:                utils.TopRightBottomLeft{padding[0], padding[1], padding[2], padding[3]},
		FontDescription:        fontDescription,
		BorderRadius:           borderRadius,
		AbsolutePosition:       anchors,
		InnerGap:               innerGap[0],
		Rotation:               rotation[0],
		BkgImageSize:           resolvedBkgImageSize,
		Border:                 border,
		Offset:                 utils.TopRightBottomLeft{offsetAnchors.Top(), offsetAnchors.Right(), offsetAnchors.Bottom(), offsetAnchors.Left()},
	}
}

// nextField returns the next whitespace-separated field of s along with the
// remainder. It is strings.Fields without the result slice, which showed up as
// a large share of allocations because every node re-parses its properties on
// every render.
func nextField(s string) (field string, rest string) {
	i := 0
	for i < len(s) && isSpaceByte(s[i]) {
		i++
	}
	s = s[i:]
	if s == "" {
		return "", ""
	}

	j := 0
	for j < len(s) && !isSpaceByte(s[j]) {
		j++
	}
	return s[:j], s[j:]
}

func isSpaceByte(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	}
	return false
}

// looksNumeric reports whether s could start a number, so that obviously
// non-numeric tokens skip strconv.ParseFloat - its error value allocates.
func looksNumeric(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	return c == '-' || c == '+' || c == '.' || (c >= '0' && c <= '9')
}

func parseAnchors(value string, data any, parentValue any, currentValueIndex int, cache *Cache) (result utils.AbsolutePosition) {
	value = replaceWithValuesUnsafe(value, data, parentValue, currentValueIndex, cache)

	for {
		var token string
		token, value = nextField(value)
		if token == "" {
			break
		}

		direction := token
		var offset float64

		// A token may carry an offset, e.g. "left/-10". Only the first segment
		// after the direction is used.
		if k := strings.IndexByte(token, '/'); k >= 0 {
			direction = token[:k]
			offsetPart := token[k+1:]
			if e := strings.IndexByte(offsetPart, '/'); e >= 0 {
				offsetPart = offsetPart[:e]
			}
			if looksNumeric(offsetPart) {
				offset, _ = strconv.ParseFloat(offsetPart, 64)
			}
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
	var widthIsSet bool
	var colorIsSet bool

	for {
		var t string
		t, value = nextField(value)
		if t == "" {
			break
		}

		if looksNumeric(t) {
			width, err := strconv.ParseFloat(t, 64)
			if err == nil {
				if widthIsSet {
					return res, fmt.Errorf("trying to specify border width %v, but width is already set", width)
				}
				widthIsSet = true
				res.Width = width
				continue
			}
		}

		// Keywords are checked first: parseColor reports failure with
		// fmt.Errorf, so asking it about "inset" allocates an error per node.
		switch t {
		case "inset":
			res.Type = utils.BorderTypeInset
			continue
		case "outset":
			res.Type = utils.BorderTypeOutset
			continue
		case "center":
			res.Type = utils.BorderTypeCenter
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

		return res, fmt.Errorf("unknown token %v in border property", t)
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

// scannedValue is one numeric token found in a property string.
type scannedValue struct {
	value float64
	unit  int
}

// scanValues extracts the numeric tokens of a property string, e.g.
// "10 20%" or "-5.5w". It replaces the equivalent regular expression
//
//	(?i)(-?\d+(\.\d+)?)(%|w|h|)
//
// which dominated allocations: every property of every node was re-scanned on
// every render, and FindAllStringSubmatch allocates a slice per match.
//
// Scanning stops early once more than max tokens are found, since callers
// treat that as a parse failure. The returned count may therefore exceed max
// by one; it is never more.
func scanValues(str string, max int, out *[4]scannedValue) int {
	count := 0

	for i := 0; i < len(str); {
		c := str[i]

		// A token is -?\d+(\.\d+)? - a leading minus only counts when a digit
		// follows it, matching the regex's backtracking behaviour.
		start := i
		if c == '-' {
			if i+1 >= len(str) || !isASCIIDigit(str[i+1]) {
				i++
				continue
			}
			i++
		} else if !isASCIIDigit(c) {
			i++
			continue
		}

		for i < len(str) && isASCIIDigit(str[i]) {
			i++
		}
		if i+1 < len(str) && str[i] == '.' && isASCIIDigit(str[i+1]) {
			i++
			for i < len(str) && isASCIIDigit(str[i]) {
				i++
			}
		}
		numEnd := i

		unit := unitAbs
		if i < len(str) {
			switch str[i] {
			case '%':
				unit = unitPercent
				i++
			case 'w', 'W':
				unit = unitWidth
				i++
			case 'h', 'H':
				unit = unitHeight
				i++
			}
		}

		if count > max {
			return count
		}

		// ParseFloat on a slice of the original string does not allocate.
		val, err := strconv.ParseFloat(str[start:numEnd], 64)
		if err != nil {
			return count
		}

		if count < len(out) {
			out[count] = scannedValue{value: val, unit: unit}
		}
		count++
	}

	return count
}

func isASCIIDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

var valuesEmptyErr = errors.New("values empty")
var valuesParseErr = errors.New("values format not correct")

func parseNValues(str string, max int, parentWidth float64, parentHeight float64, data any, parentData any, currentValueIndex int, relativeToWidth bool, allowNegative bool, cache *Cache) (utils.FourValues, error) {
	var result utils.FourValues

	if str == "" {
		return result, valuesEmptyErr
	}

	str = replaceWithValuesUnsafe(str, data, parentData, currentValueIndex, cache)

	var scanned [4]scannedValue
	count := scanValues(str, max, &scanned)
	if count > max || count == 0 {
		return result, valuesParseErr
	}

	for i := 0; i < count; i++ {
		val := scanned[i].value

		if !allowNegative && val < 0 {
			val = -val
		}

		isVertical := i%2 == 1
		if max == 1 {
			isVertical = !relativeToWidth
		}

		result[i] = prepareParsedValue(val, isVertical, scanned[i].unit, parentWidth, parentHeight)
	}

	if count == 1 {
		result[1] = result[0]
		result[2] = result[0]
		result[3] = result[0]
	}

	if count == 2 {
		result[2] = result[0]
		result[3] = result[1]
	}

	if count == 3 {
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

func parseFontString(prop string, fd fonts.FaceDescription, parentWidth float64, parentHeight float64, data any, parentData any, currentValueIndex int, cache *Cache) fonts.FaceDescription {
	if prop == "" {
		return fd
	}

	prop = replaceWithValuesUnsafe(prop, data, parentData, currentValueIndex, cache)
	prop = strings.ReplaceAll(prop, ",", " ")

	isSizeSet := false

	for {
		var token string
		token, prop = nextField(prop)
		if token == "" {
			break
		}

		v, err := parseNValues(token, 1, parentWidth, parentHeight, data, parentData, currentValueIndex, true, false, cache)
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
