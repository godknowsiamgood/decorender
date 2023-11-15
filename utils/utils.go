package utils

import (
	"crypto/sha256"
	"encoding/hex"
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

type Anchors [4]bool

func (a Anchors) Has() bool {
	return a[0] || a[1] || a[2] || a[3]
}
func (a Anchors) Left() bool {
	return a[3]
}
func (a Anchors) Top() bool {
	return a[0]
}
func (a Anchors) Right() bool {
	return a[1]
}
func (a Anchors) Bottom() bool {
	return a[2]
}

func GetSha256(str string) string {
	hash := sha256.Sum256([]byte(str))
	return hex.EncodeToString(hash[:])
}
