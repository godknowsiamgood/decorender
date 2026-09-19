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

// resolver evaluates the template form of property values and remembers the
// first failure, so that calculateProperties can report it instead of
// rendering something the layout never asked for.
//
// An expression that does not compile, or that fails against the data, is a
// mistake in the layout rather than a value with a sensible default. Until
// now every property but color swallowed it: a misquoted expression in
// `absolute`, for example, left the node with no anchors at all, so it
// silently dropped back into the flow. Colors already reported this; the rest
// of the properties now do too.
type resolver struct {
	data       any
	parentData any
	valueIndex int
	cache      *Cache
	err        error
}

func (r *resolver) str(prop string, source string) string {
	v, err := replaceWithValues(source, r.data, r.parentData, r.valueIndex, r.cache)
	if err != nil {
		if r.err == nil {
			r.err = fmt.Errorf("%s %q: %w", prop, source, err)
		}
		return ""
	}
	return v
}

// values resolves and parses a numeric property, e.g. "10 20%" or "0.5w".
// The parse error is the caller's to ignore - properties fall back to their
// default when they do not parse - while an expression failure is kept.
func (r *resolver) values(prop string, source string, max int, parentWidth float64, parentHeight float64, relativeToWidth bool, allowNegative bool) (utils.FourValues, error) {
	return parseNValues(r.str(prop, source), max, parentWidth, parentHeight, relativeToWidth, allowNegative)
}

// gaps resolves innerGap, which is one gap for both axes or the pair
// "<row> <column>", as CSS writes it: the first value spaces rows from each
// other, the second one spaces children within a row. A legend that wraps
// needs them apart - its lines sit close together while its items stay far
// enough apart to be read as separate.
//
// The two are parsed one by one rather than as a pair, so that a percentage
// lands on the axis it is spent on: a row gap is a share of the height, a
// column gap a share of the width.
func (r *resolver) gaps(prop string, source string, size utils.Size) (row float64, column float64) {
	fields := strings.Fields(r.str(prop, source))
	if len(fields) == 1 {
		fields = append(fields, fields[0])
	}
	if len(fields) != 2 {
		return 0, 0
	}
	rowValue, _ := parseNValues(fields[0], 1, size.W, size.H, false, false)
	columnValue, _ := parseNValues(fields[1], 1, size.W, size.H, true, false)
	return rowValue[0], columnValue[0]
}

// calculateProperties is currently ugly function that needs refactoring.
// Maybe we should introduce some fields generic configuration.
//
// Only colors are reported as errors. Every other property here falls back to
// a default when it does not parse, which is a deliberate looseness the
// renderer has always had; a color has no usable fallback, because the zero
// RGBA is fully transparent and simply makes the element vanish.
func calculateProperties(n parsing.Node, context layoutPhaseContext, data any, parentData any, currentValueIndex int) (CalculatedProperties, error) {
	res := resolver{data: data, parentData: parentData, valueIndex: currentValueIndex, cache: context.cache}

	padding, _ := res.values("padding", n.Padding, 4, context.size.W, context.size.H, false, false)
	lineHeight, _ := res.values("lineHeight", n.LineHeight, 1, context.size.W, context.size.H, false, false)
	borderRadius, _ := res.values("borderRadius", n.BorderRadius, 4, context.size.W, context.size.H, false, false)

	sz, szErr := res.values("size", n.Size, 2, context.size.W, context.size.H, false, false)
	width, widthErr := res.values("width", n.Width, 1, context.size.W, context.size.H, true, false)
	height, heightErr := res.values("height", n.Height, 1, context.size.W, context.size.H, false, false)
	if szErr != nil {
		sz[0], sz[1] = -1, -1
	}
	if widthErr == nil {
		sz[0] = width[0]
	}
	if heightErr == nil {
		sz[1] = height[0]
	}

	anchors, anchorsErr := parseAnchors(res.str("absolute", n.Absolute))
	if anchorsErr != nil {
		return CalculatedProperties{}, fmt.Errorf("absolute: %w", anchorsErr)
	}
	if anchors.HasTop() && anchors.HasBottom() {
		sz[1] = context.size.H - anchors.Top() - anchors.Bottom()
	}
	if anchors.HasLeft() && anchors.HasRight() {
		sz[0] = context.size.W - anchors.Left() - anchors.Right()
	}

	backgroundColor := color.RGBA{A: 0}
	if n.BkgColor != "" {
		c, err := resolveColor("bkgColor", n.BkgColor, &res)
		if err != nil {
			return CalculatedProperties{}, err
		}
		backgroundColor = c
	}

	bkgImageSize := validateStringValue(res.str("bkgImageSize", n.BkgImageSize), []string{"cover", "contain"})

	fontColor := context.props.FontColor // inherited
	// Order matters: `color` is the alias and wins when both are given.
	for _, f := range []struct{ prop, source string }{{"fontColor", n.FontColor}, {"color", n.Color}} {
		if f.source == "" {
			continue
		}
		c, err := resolveColor(f.prop, f.source, &res)
		if err != nil {
			return CalculatedProperties{}, err
		}
		fontColor = c
	}

	fontDescription := context.props.FontDescription // inherited
	fontDescription = parseFontString(res.str("font", n.Font), fontDescription, context.size.W, context.size.H)
	if n.FontFamily != "" {
		fontDescription.Family = res.str("fontFamily", n.FontFamily)
	}
	if n.FontSize != "" {
		v, err := res.values("fontSize", n.FontSize, 1, context.size.W, context.size.H, true, false)
		if err == nil {
			fontDescription.Size = v[0]
		}
	}
	if n.FontWeight != "" {
		v, err := res.values("fontWeight", n.FontWeight, 1, context.size.W, context.size.H, true, false)
		if err == nil {
			fontDescription.Weight = int(v[0])
		}
	}
	if n.FontStyle != "" {
		fontDescription.Style = font.StyleNormal
		if res.str("fontStyle", n.FontStyle) == "italic" {
			fontDescription.Style = font.StyleItalic
		}
	}

	childrenDirection := validateStringValue(res.str("innerDirection", n.InnerDirection), []string{"column", "row"})
	childrenJustify := validateStringValue(res.str("justify", n.Justify), []string{"start", "center", "end", "space-between", "space-evenly"})
	childrenColumnAlign := validateStringValue(res.str("innerColumnAlign", n.ChildrenColumnAlign), []string{"left", "center", "right"})
	childrenRowAlign := validateStringValue(res.str("innerRowAlign", n.ChildrenRowAlign), []string{"top", "center", "bottom"})
	childrenWrap := validateStringValue(res.str("innerWrap", n.ChildrenWrap), []string{"wrap", "none"})

	innerRowGap, innerColumnGap := res.gaps("innerGap", n.InnerGap, context.size)

	rotation, _ := res.values("rotate", n.Rotation, 1, context.size.W, context.size.H, true, true)

	border, err := parseBorderProperty(res.str("border", n.Border))
	if err != nil {
		return CalculatedProperties{}, fmt.Errorf("border: %w", err)
	}

	offsetAnchors, offsetErr := parseAnchors(res.str("offset", n.Offset))
	if offsetErr != nil {
		return CalculatedProperties{}, fmt.Errorf("offset: %w", offsetErr)
	}

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

	if res.err != nil {
		return CalculatedProperties{}, res.err
	}

	return CalculatedProperties{
		Size:                   utils.Size{W: sz[0], H: sz[1]},
		BkgColor:               backgroundColor,
		FontColor:              fontColor,
		ChildAlign:             "",
		IsChildrenDirectionRow: childrenDirection == "row",
		Justify:                childrenJustify,
		ChildrenColumnAlign:    childrenColumnAlign,
		ChildrenRowAlign:       childrenRowAlign,
		IsWrappingEnabled:      childrenWrap == "wrap",
		LineHeight:             resolvedLineHeight,
		Padding:                utils.TopRightBottomLeft{padding[0], padding[1], padding[2], padding[3]},
		FontDescription:        fontDescription,
		BorderRadius:           borderRadius,
		AbsolutePosition:       anchors,
		InnerRowGap:            innerRowGap,
		InnerColumnGap:         innerColumnGap,
		Rotation:               rotation[0],
		BkgImageSize:           resolvedBkgImageSize,
		Border:                 border,
		Offset:                 utils.TopRightBottomLeft{offsetAnchors.Top(), offsetAnchors.Right(), offsetAnchors.Bottom(), offsetAnchors.Left()},
	}, nil
}

// resolveColor evaluates a color property and says which one failed.
//
// A color that did not parse used to be discarded along with its error,
// leaving the zero RGBA: fully transparent. A misspelt color name, or an
// expression that did not resolve, therefore produced an invisible element and
// no diagnostic whatsoever.
func resolveColor(prop string, source string, res *resolver) (color.RGBA, error) {
	resolved := res.str(prop, source)
	if res.err != nil {
		return color.RGBA{}, res.err
	}

	c, err := parseColor(resolved)
	if err == nil {
		return c, nil
	}

	// An expression rarely looks like what it evaluated to, so name both.
	if strings.HasPrefix(source, "~") {
		return color.RGBA{}, fmt.Errorf("%s %q: %w", prop, source, err)
	}
	return color.RGBA{}, fmt.Errorf("%s: %w", prop, err)
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

// parseAnchors reads the directions of `absolute` or `offset`.
//
// A token that names no direction is reported rather than skipped: it is
// always a mistake, and skipping it left the node with no anchors at all, so
// it quietly went back into the flow instead of being positioned. An
// expression that evaluates to a string literal - the usual result of
// misplaced quotes - lands here.
func parseAnchors(value string) (result utils.AbsolutePosition, err error) {
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
		default:
			if err == nil {
				err = fmt.Errorf("unknown direction %q, expected top, right, bottom or left", direction)
			}
		}
	}
	return result, err
}

func parseBorderProperty(value string) (res utils.Border, err error) {
	var widthIsSet bool
	var colorIsSet bool
	var isDashed bool

	for {
		var t string
		t, value = nextField(value)
		if t == "" {
			break
		}

		// A dash pattern may carry its lengths, e.g. "dashed/6/3", the same
		// way an anchor carries its offset.
		if t == "dashed" || strings.HasPrefix(t, "dashed/") {
			if isDashed {
				return res, fmt.Errorf("trying to specify dash pattern %v, but it is already set", t)
			}
			isDashed = true

			lengths := strings.Split(t, "/")[1:]
			if len(lengths) > utils.MaxBorderDashes {
				return res, fmt.Errorf("dash pattern %v has more than %v lengths", t, utils.MaxBorderDashes)
			}
			for i, l := range lengths {
				v, convErr := strconv.ParseFloat(l, 64)
				if convErr != nil || v < 0 {
					return res, fmt.Errorf("dash length %q in %v is not a positive number", l, t)
				}
				res.Dashes[i] = v
			}
			res.DashCount = len(lengths)
			continue
		}

		switch t {
		case "top":
			res.Sides |= utils.BorderSideTop
			continue
		case "right":
			res.Sides |= utils.BorderSideRight
			continue
		case "bottom":
			res.Sides |= utils.BorderSideBottom
			continue
		case "left":
			res.Sides |= utils.BorderSideLeft
			continue
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

	// A pattern given without lengths follows the border width, which is only
	// known once every token has been read - they may come in any order.
	if isDashed && res.DashCount == 0 {
		res.Dashes[0] = res.Width * 4
		res.Dashes[1] = res.Width * 4
		res.DashCount = 2
	}

	if isDashed && !res.IsDashed() {
		return res, fmt.Errorf("dash pattern of zero length draws nothing")
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

func parseNValues(str string, max int, parentWidth float64, parentHeight float64, relativeToWidth bool, allowNegative bool) (utils.FourValues, error) {
	var result utils.FourValues

	if str == "" {
		return result, valuesEmptyErr
	}

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

func parseFontString(prop string, fd fonts.FaceDescription, parentWidth float64, parentHeight float64) fonts.FaceDescription {
	if prop == "" {
		return fd
	}

	prop = strings.ReplaceAll(prop, ",", " ")

	isSizeSet := false

	for {
		var token string
		token, prop = nextField(prop)
		if token == "" {
			break
		}

		v, err := parseNValues(token, 1, parentWidth, parentHeight, true, false)
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

// nodeRef names a node in an error message. Ids are optional, so fall back to
// the node's text, and then to nothing at all.
func nodeRef(n parsing.Node) string {
	if n.Id != "" {
		return "node " + n.Id
	}
	if n.Text != "" {
		text := []rune(n.Text)
		if len(text) > 30 {
			return fmt.Sprintf("node with text %q...", string(text[:30]))
		}
		return fmt.Sprintf("node with text %q", n.Text)
	}
	return "node"
}
