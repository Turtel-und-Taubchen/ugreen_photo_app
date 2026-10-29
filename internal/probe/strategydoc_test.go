package probe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"photocurator/internal/testsupport"
)

func TestStrategyDoc_ListsEveryProbeField(t *testing.T) {
	path := filepath.Join(testsupport.RepoRoot(t), "docs", "concepts", "photocurator-metadaten-strategie.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	rows := map[string]bool{}
	for _, line := range strings.Split(doc, "\n") {
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		if len(cells) >= 2 {
			rows[strings.TrimSpace(cells[0])+"|"+strings.TrimSpace(cells[1])] = true
		}
	}
	for _, p := range Probes() {
		if !rows[string(p.Format)+"|"+p.FieldKey] {
			t.Errorf("matrix lacks row %s / %s", p.Format, p.FieldKey)
		}
	}
	if !strings.Contains(doc, "## Entscheidung nach Spike") {
		t.Error(`heading "Entscheidung nach Spike" missing`)
	}
}
