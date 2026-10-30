package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every quest counter must actually be incremented somewhere in the server. The world boss
// daily shipped unwinnable because its bump call was never added, and nothing caught it.
func TestEveryQuestCounterIsIncremented(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var src strings.Builder
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src.Write(b)
	}
	code := src.String()
	seen := map[string]bool{}
	for i := range quests {
		c := questCounter(quests[i].ID)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		if !strings.Contains(code, `bump("`+c+`"`) {
			t.Fatalf("quest %q reads counter %q but nothing ever increments it", quests[i].ID, c)
		}
	}
}
