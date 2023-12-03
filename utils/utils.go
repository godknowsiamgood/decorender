package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image/color"
	"sync"
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

func (fv FourValues) HasValues() bool {
	return fv[0] > 0 || fv[1] > 0 || fv[2] > 0 || fv[3] > 0
}
func (fv FourValues) Width() float64 {
	return fv[0]
}
func (fv FourValues) Height() float64 {
	return fv[1]
}

type TopRightBottomLeft [4]float64

func (s TopRightBottomLeft) Top() float64 {
	return s[0]
}
func (s TopRightBottomLeft) Left() float64 {
	return s[3]
}
func (s TopRightBottomLeft) Right() float64 {
	return s[1]
}
func (s TopRightBottomLeft) Bottom() float64 {
	return s[2]
}

type AbsolutePos struct {
	Has    bool
	Offset float64
}
type AbsolutePosition [4]AbsolutePos

func (a AbsolutePosition) Has() bool {
	return a[0].Has || a[1].Has || a[2].Has || a[3].Has
}
func (a AbsolutePosition) Left() float64 {
	return a[3].Offset
}
func (a AbsolutePosition) Top() float64 {
	return a[0].Offset
}
func (a AbsolutePosition) Right() float64 {
	return a[1].Offset
}
func (a AbsolutePosition) Bottom() float64 {
	return a[2].Offset
}

func (a AbsolutePosition) HasTop() bool {
	return a[0].Has
}
func (a AbsolutePosition) HasLeft() bool {
	return a[3].Has
}
func (a AbsolutePosition) HasRight() bool {
	return a[1].Has
}
func (a AbsolutePosition) HasBottom() bool {
	return a[2].Has
}

type BorderType int

const (
	BorderTypeOutset BorderType = iota
	BorderTypeCenter
	BorderTypeInset
)

type Border struct {
	Type  BorderType
	Width float64
	Color color.RGBA
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

type DebugPool struct {
	free []any
	mx   sync.Mutex
	New  func() any
	cnt  int
}

func (p *DebugPool) Get() any {
	p.mx.Lock()
	defer p.mx.Unlock()
	if len(p.free) == 0 {
		p.cnt += 1
		return p.New()
	} else {
		v := p.free[len(p.free)-1]
		p.free = p.free[0 : len(p.free)-2]
		p.cnt += 1
		return v
	}
}

func (p *DebugPool) Put(v any) {
	p.mx.Lock()
	defer p.mx.Unlock()
	p.cnt -= 1
	p.free = append(p.free, v)
	fmt.Println("back", p.cnt)
}
