package game

import "testing"

// Levels at or below the old cap must keep the exact budget stored pets were built with.
func TestBuildPointsKeepsLegacyBudget(t *testing.T) {
	for lv := 1; lv <= softCapLv; lv++ {
		if got, want := buildPoints(lv), (lv-1)*3; got != want {
			t.Fatalf("lv %d: got %d, want %d", lv, got, want)
		}
	}
}

func TestBuildPointsPastCapGrantsTwo(t *testing.T) {
	base := buildPoints(softCapLv)
	for lv := softCapLv + 1; lv <= maxLv; lv++ {
		if got, want := buildPoints(lv), base+(lv-softCapLv)*2; got != want {
			t.Fatalf("lv %d: got %d, want %d", lv, got, want)
		}
	}
	if buildPoints(maxLv) <= buildPoints(maxLv-1) {
		t.Fatal("budget must keep growing to the cap")
	}
}
