package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"image/color"
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

type Anchor struct {
	Has    bool
	Offset float64
}
type Anchors [4]Anchor

func (a Anchors) Has() bool {
	return a[0].Has || a[1].Has || a[2].Has || a[3].Has
}
func (a Anchors) Left() float64 {
	return a[3].Offset
}
func (a Anchors) Top() float64 {
	return a[0].Offset
}
func (a Anchors) Right() float64 {
	return a[1].Offset
}
func (a Anchors) Bottom() float64 {
	return a[2].Offset
}

func (a Anchors) HasTop() bool {
	return a[0].Has
}
func (a Anchors) HasLeft() bool {
	return a[3].Has
}
func (a Anchors) HasRight() bool {
	return a[1].Has
}
func (a Anchors) HasBottom() bool {
	return a[2].Has
}

type BorderType int

const (
	BorderTypeCenter BorderType = iota
	BorderTypeOutset
	BorderTypeInset
)

type Border struct {
	Type  BorderType
	Width float64
	Color color.RGBA
}

func GetSha256(str string) string {
	hash := sha256.Sum256([]byte(str))
	return hex.EncodeToString(hash[:])
}
