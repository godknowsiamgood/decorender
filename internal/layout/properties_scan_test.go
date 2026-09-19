package layout

import (
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The regular expression scanValues replaced. Kept here as the oracle: the
// scanner must agree with it on every input.
var referenceValueRegex = regexp.MustCompile(`(?i)(-?\d+(\.\d+)?)(%|w|h|)`)

func referenceScan(str string) []scannedValue {
	matches := referenceValueRegex.FindAllStringSubmatch(str, -1)
	out := make([]scannedValue, 0, len(matches))
	for _, m := range matches {
		val, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			break
		}
		unit := unitAbs
		switch strings.ToLower(m[3]) {
		case "%":
			unit = unitPercent
		case "w":
			unit = unitWidth
		case "h":
			unit = unitHeight
		}
		out = append(out, scannedValue{value: val, unit: unit})
	}
	return out
}

// compare runs both implementations with a generous max so neither exits early.
func compare(t *testing.T, input string) {
	t.Helper()

	want := referenceScan(input)

	// A max this large means the early exit never fires, so counts are directly
	// comparable with the regex's total.
	var got [4]scannedValue
	n := scanValues(input, 1<<30, &got)

	if n != len(want) {
		t.Errorf("scanValues(%q) found %d values, regex found %d", input, n, len(want))
		return
	}
	for i := 0; i < n && i < len(got); i++ {
		if got[i] != want[i] {
			t.Errorf("scanValues(%q)[%d] = %+v, regex = %+v", input, i, got[i], want[i])
		}
	}
}

func TestScanValuesMatchesRegex(t *testing.T) {
	cases := []string{
		"", " ", "0", "10", "10 20", "10 20 30 40",
		"100%", "50% 25%", "1.5w", "2h", "0.5W", "3H",
		"-5", "-5.5", "-5 -10", "--5", "-", "-.5", ".5",
		"5.5.5", "1.", "1.2.3", "007", "1e5",
		"Inter 23 400", "italic", "normal", "Inter", "Inter2",
		"10px", "10 20 30 40 50", "  12   34  ",
		"salmon", "0xaabbcc", "rgb(1,2,3)",
		"left/-10 top/55", "wwww", "hhh", "%%%",
		"1w2h3%", "9999999999", "0.000001",
	}

	for _, c := range cases {
		compare(t, c)
	}
}

func TestScanValuesFuzzMatchesRegex(t *testing.T) {
	const alphabet = "0123456789.-%wWhH xyInter/,"

	rnd := rand.New(rand.NewSource(1))
	for i := 0; i < 50000; i++ {
		n := rnd.Intn(24)
		var sb strings.Builder
		for j := 0; j < n; j++ {
			sb.WriteByte(alphabet[rnd.Intn(len(alphabet))])
		}
		compare(t, sb.String())
	}
}

// Callers treat "more values than expected" as a parse failure, so the scanner
// only has to report that the limit was exceeded, not the exact total.
func TestScanValuesStopsPastMax(t *testing.T) {
	var out [4]scannedValue
	if n := scanValues("1 2 3 4 5 6 7 8", 2, &out); n <= 2 {
		t.Errorf("expected the scan to report exceeding max=2, got %d", n)
	}
	if n := scanValues("1 2", 2, &out); n != 2 {
		t.Errorf("expected exactly 2 values, got %d", n)
	}
}

func BenchmarkScanValues(b *testing.B) {
	var out [4]scannedValue
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		scanValues("10 20 30 40", 4, &out)
	}
}

func BenchmarkReferenceRegex(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		referenceValueRegex.FindAllStringSubmatch("10 20 30 40", -1)
	}
}
