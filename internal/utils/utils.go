package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"image/color"
	"math"
)

type Size struct {
	W float64
	H float64
}

type Pos struct {
	Left float64
	Top  float64
}

type FourValues [4]float64

func (fv *FourValues) HasValues() bool {
	return fv[0] > 0 || fv[1] > 0 || fv[2] > 0 || fv[3] > 0
}
func (fv *FourValues) Width() float64 {
	return fv[0]
}
func (fv *FourValues) Height() float64 {
	return fv[1]
}

type TopRightBottomLeft [4]float64

func (s *TopRightBottomLeft) Top() float64 {
	return s[0]
}
func (s *TopRightBottomLeft) Left() float64 {
	return s[3]
}
func (s *TopRightBottomLeft) Right() float64 {
	return s[1]
}
func (s *TopRightBottomLeft) Bottom() float64 {
	return s[2]
}

type AbsolutePos struct {
	Offset float64
	Has    bool
}
type AbsolutePosition [4]AbsolutePos

func (a *AbsolutePosition) Has() bool {
	return a[0].Has || a[1].Has || a[2].Has || a[3].Has
}
func (a *AbsolutePosition) Left() float64 {
	return a[3].Offset
}
func (a *AbsolutePosition) Top() float64 {
	return a[0].Offset
}
func (a *AbsolutePosition) Right() float64 {
	return a[1].Offset
}
func (a *AbsolutePosition) Bottom() float64 {
	return a[2].Offset
}

func (a *AbsolutePosition) HasTop() bool {
	return a[0].Has
}
func (a *AbsolutePosition) HasLeft() bool {
	return a[3].Has
}
func (a *AbsolutePosition) HasRight() bool {
	return a[1].Has
}
func (a *AbsolutePosition) HasBottom() bool {
	return a[2].Has
}

type BorderType int

const (
	BorderTypeOutset BorderType = iota
	BorderTypeCenter
	BorderTypeInset
)

// BorderSides is a set of the edges a border is drawn on. The zero value
// means every edge, so a border keeps covering the whole node unless the
// layout names the sides it wants.
type BorderSides uint8

const (
	BorderSideTop BorderSides = 1 << iota
	BorderSideRight
	BorderSideBottom
	BorderSideLeft
)

const BorderSidesAll = BorderSideTop | BorderSideRight | BorderSideBottom | BorderSideLeft

// MaxBorderDashes caps the dash pattern. Canvas takes an array of any length,
// but patterns past a handful of entries are not legible on a border, and a
// fixed array keeps a Border comparable and free of allocation.
const MaxBorderDashes = 8

type Border struct {
	Type  BorderType
	Width float64
	Color color.RGBA
	Sides BorderSides
	// Dashes alternates drawn and skipped lengths along the outline, like
	// the array given to canvas setLineDash. No entries draws a solid border.
	Dashes    [MaxBorderDashes]float64
	DashCount int
}

func (b *Border) HasSide(side BorderSides) bool {
	return b.Sides == 0 || b.Sides&side != 0
}

func (b *Border) IsDashed() bool {
	return b.DashPeriod() > 0.0001
}

// DashPeriod is the length after which the pattern repeats, or zero when the
// border is solid.
//
// An odd number of entries repeats with drawn and skipped lengths swapped -
// [5 10 5] behaves as [5 10 5 5 10 5] - which is what canvas does by
// duplicating such an array.
func (b *Border) DashPeriod() float64 {
	var total float64
	for i := 0; i < b.DashCount; i++ {
		total += b.Dashes[i]
	}
	if total <= 0 {
		return 0
	}
	if b.DashCount%2 == 1 {
		return total * 2
	}
	return total
}

// DashCovers reports whether the outline is drawn at pos, a distance measured
// along it. period comes from DashPeriod and is passed in because the caller
// asks this once per pixel.
func (b *Border) DashCovers(pos float64, period float64) bool {
	if period <= 0 {
		return true
	}

	pos = math.Mod(pos, period)
	if pos < 0 {
		pos += period
	}

	drawn := true
	for i := 0; i < 2*b.DashCount; i++ {
		length := b.Dashes[i%b.DashCount]
		if pos < length {
			return drawn
		}
		pos -= length
		drawn = !drawn
	}

	return drawn
}

// NeedsMask reports whether the border covers less than the whole outline.
func (b *Border) NeedsMask() bool {
	return b.IsDashed() || (b.Sides != 0 && b.Sides != BorderSidesAll)
}

func (b *Border) GetOutsetOffset() float64 {
	switch b.Type {
	case BorderTypeOutset:
		return b.Width
	case BorderTypeCenter:
		return b.Width / 2
	default:
		return 0
	}
}

func GetSha256(str string) string {
	hash := sha256.Sum256([]byte(str))
	return hex.EncodeToString(hash[:])
}

type Stack[T any] []T

func (s *Stack[T]) Push(value T) {
	*s = append(*s, value)
}

func (s *Stack[T]) Pop() T {
	if len(*s) == 0 {
		var zeroValue T
		return zeroValue
	}
	index := len(*s) - 1
	element := (*s)[index]
	*s = (*s)[:index]
	return element
}

func (s *Stack[T]) Last() T {
	if len(*s) == 0 {
		var v T
		return v
	} else {
		return (*s)[len(*s)-1]
	}
}

func (s *Stack[T]) Len() int {
	return len(*s)
}
