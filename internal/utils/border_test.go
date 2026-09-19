package utils

import "testing"

func dashed(lengths ...float64) Border {
	var b Border
	for i, l := range lengths {
		b.Dashes[i] = l
	}
	b.DashCount = len(lengths)
	return b
}

func TestDashPeriod(t *testing.T) {
	cases := []struct {
		name   string
		border Border
		want   float64
	}{
		{"solid", Border{}, 0},
		{"dash and gap", dashed(4, 4), 8},
		{"lengths of zero draw nothing", dashed(0, 0), 0},
		// An odd pattern comes back to its start only after two laps, which is
		// how canvas behaves with an array it has duplicated.
		{"odd pattern repeats twice", dashed(5, 10, 5), 40},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.border.DashPeriod(); got != c.want {
				t.Errorf("DashPeriod() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestDashCovers(t *testing.T) {
	// A solid border covers everything, including positions past any period.
	solid := Border{}
	for _, pos := range []float64{0, 1, 1000} {
		if !solid.DashCovers(pos, solid.DashPeriod()) {
			t.Errorf("solid border does not cover %v", pos)
		}
	}

	pattern := dashed(4, 4)
	period := pattern.DashPeriod()
	for pos, want := range map[float64]bool{
		0: true, 3.9: true, 4: false, 7.9: false,
		8: true, 11.9: true, 12: false,
		-1: false, // before the start, i.e. inside the previous gap
	} {
		if got := pattern.DashCovers(pos, period); got != want {
			t.Errorf("dashed/4/4 covers %v = %v, want %v", pos, got, want)
		}
	}

	// [5 10 5] behaves as [5 10 5 5 10 5]: the second lap swaps drawn and
	// skipped, so the pattern runs 5 on, 10 off, 5 on, 5 off, 10 on, 5 off.
	odd := dashed(5, 10, 5)
	period = odd.DashPeriod()
	for pos, want := range map[float64]bool{
		0: true, 4.9: true, 5: false, 14.9: false,
		15: true, 19.9: true, 20: false, 24.9: false,
		25: true, 34.9: true, 35: false, 39.9: false,
	} {
		if got := odd.DashCovers(pos, period); got != want {
			t.Errorf("dashed/5/10/5 covers %v = %v, want %v", pos, got, want)
		}
	}
}

func TestHasSide(t *testing.T) {
	// No sides named means the border is drawn all the way round.
	all := Border{}
	for _, side := range []BorderSides{BorderSideTop, BorderSideRight, BorderSideBottom, BorderSideLeft} {
		if !all.HasSide(side) {
			t.Errorf("a border with no sides named skips %v", side)
		}
	}

	top := Border{Sides: BorderSideTop}
	if !top.HasSide(BorderSideTop) {
		t.Error("a top border does not cover the top")
	}
	if top.HasSide(BorderSideBottom) {
		t.Error("a top border covers the bottom")
	}
}
