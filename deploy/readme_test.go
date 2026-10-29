package deploy

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestInstallGuideCoversEveryStep(t *testing.T) {
	raw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	readme := string(raw)
	for _, h := range []string{
		"## Ordner anlegen", "## Nutzer und UID ermitteln", "## Rechte", "## .env anlegen",
		"## Image übertragen", "## Projekt anlegen", "## Erster Aufruf", "## Update",
	} {
		if !strings.Contains(readme, "\n"+h+"\n") {
			t.Errorf("heading %q missing", h)
		}
	}
	env, err := os.ReadFile(".env.example")
	if err != nil {
		t.Fatal(err)
	}
	names := regexp.MustCompile(`(?m)^#?\s*([A-Z][A-Z0-9_]*)=`).FindAllStringSubmatch(string(env), -1)
	if len(names) == 0 {
		t.Fatal("no variables found in .env.example")
	}
	for _, m := range names {
		if !strings.Contains(readme, m[1]) {
			t.Errorf("variable %s not mentioned in README", m[1])
		}
	}
}
