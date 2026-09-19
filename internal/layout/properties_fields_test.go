package layout

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/godknowsiamgood/decorender/internal/utils"
)

// collectFields drains nextField the way the parsers do.
func collectFields(s string) []string {
	var out []string
	for {
		var f string
		f, s = nextField(s)
		if f == "" {
			break
		}
		out = append(out, f)
	}
	return out
}

func TestNextFieldMatchesStringsFields(t *testing.T) {
	cases := []string{
		"", " ", "  ", "a", "a b", "  a   b  ", "\ta\nb\r\nc ",
		"left/-10 top/55", "top right bottom left",
		"1 2 3 4", "inset 2 red", "Inter 23 400",
		"\v\f x \v", "trailing   ", "   leading",
	}
	for _, c := range cases {
		got := collectFields(c)
		want := strings.Fields(c)
		if len(got) != len(want) {
			t.Errorf("nextField(%q) = %q, strings.Fields = %q", c, got, want)
			continue
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("nextField(%q)[%d] = %q, want %q", c, i, got[i], want[i])
			}
		}
	}
}

func TestNextFieldFuzzMatchesStringsFields(t *testing.T) {
	const alphabet = "ab /\t\n\r\v\f-0123456789"

	rnd := rand.New(rand.NewSource(7))
	for i := 0; i < 20000; i++ {
		n := rnd.Intn(16)
		var sb strings.Builder
		for j := 0; j < n; j++ {
			sb.WriteByte(alphabet[rnd.Intn(len(alphabet))])
		}
		in := sb.String()

		got := collectFields(in)
		want := strings.Fields(in)
		if len(got) != len(want) {
			t.Fatalf("nextField(%q) = %q, strings.Fields = %q", in, got, want)
		}
		for k := range got {
			if got[k] != want[k] {
				t.Fatalf("nextField(%q)[%d] = %q, want %q", in, k, got[k], want[k])
			}
		}
	}
}

// referenceParseAnchors is the original strings.Fields + strings.Split version.
func referenceParseAnchors(value string) (result utils.AbsolutePosition) {
	for _, token := range strings.Fields(value) {
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

func TestParseAnchorsMatchesReference(t *testing.T) {
	cache := NewCache()

	cases := []string{
		"", "left", "right", "top", "bottom",
		"left right", "top bottom", "left top",
		"left/-10", "top/55", "left/-10 top/55",
		"left/10/20", "left/", "/10", "left//5",
		"left/abc", "left/1.5", "left/-1.5", "left/+2",
		"LEFT", "  left   top  ", "middle", "left/1e3",
	}
	for _, c := range cases {
		got := parseAnchors(c, nil, nil, 0, cache)
		want := referenceParseAnchors(c)
		if got != want {
			t.Errorf("parseAnchors(%q) = %+v, reference = %+v", c, got, want)
		}
	}
}

func TestParseAnchorsFuzzMatchesReference(t *testing.T) {
	const alphabet = "leftrighopbm/ -.0123456789"

	cache := NewCache()
	rnd := rand.New(rand.NewSource(11))
	for i := 0; i < 20000; i++ {
		n := rnd.Intn(20)
		var sb strings.Builder
		for j := 0; j < n; j++ {
			sb.WriteByte(alphabet[rnd.Intn(len(alphabet))])
		}
		in := sb.String()

		if got, want := parseAnchors(in, nil, nil, 0, cache), referenceParseAnchors(in); got != want {
			t.Fatalf("parseAnchors(%q) = %+v, reference = %+v", in, got, want)
		}
	}
}
