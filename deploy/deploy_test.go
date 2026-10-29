package deploy

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type service struct {
	User        string         `yaml:"user"`
	Volumes     []string       `yaml:"volumes"`
	ReadOnly    bool           `yaml:"read_only"`
	CapDrop     []string       `yaml:"cap_drop"`
	SecurityOpt []string       `yaml:"security_opt"`
	MemLimit    string         `yaml:"mem_limit"`
	CPUs        float64        `yaml:"cpus"`
	Healthcheck map[string]any `yaml:"healthcheck"`
	Privileged  bool           `yaml:"privileged"`
	NetworkMode string         `yaml:"network_mode"`
	Pid         string         `yaml:"pid"`
	Devices     []string       `yaml:"devices"`
}

func loadService(t *testing.T) service {
	t.Helper()
	raw, err := os.ReadFile("docker-compose.yml")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Services map[string]service `yaml:"services"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	s, ok := doc.Services["photocurator"]
	if !ok {
		t.Fatal("service photocurator missing")
	}
	return s
}

func TestComposeMountsOnlyPhotosDataBackup(t *testing.T) {
	s := loadService(t)
	if len(s.Volumes) != 3 {
		t.Fatalf("want exactly 3 volumes, got %v", s.Volumes)
	}
	targets := map[string]bool{}
	for _, v := range s.Volumes {
		i := strings.LastIndex(v, ":")
		if i < 0 {
			t.Fatalf("volume %q has no target", v)
		}
		targets[v[i+1:]] = true
		if strings.Contains(v, "docker.sock") || strings.Contains(strings.ToLower(v), "inbox") {
			t.Errorf("forbidden mount %q", v)
		}
	}
	for _, want := range []string{"/photos", "/data", "/backup"} {
		if !targets[want] {
			t.Errorf("missing mount target %s", want)
		}
	}
	if s.Privileged || s.NetworkMode == "host" || s.Pid == "host" || len(s.Devices) > 0 {
		t.Errorf("privileged/host network/host pid/devices must not be set: %+v", s)
	}
}

func TestComposeServiceIsHardened(t *testing.T) {
	s := loadService(t)
	if s.User != "${PUID}:${PGID}" {
		t.Errorf("user = %q", s.User)
	}
	if !s.ReadOnly {
		t.Error("read_only must be true")
	}
	if len(s.CapDrop) != 1 || s.CapDrop[0] != "ALL" {
		t.Errorf("cap_drop = %v", s.CapDrop)
	}
	found := false
	for _, o := range s.SecurityOpt {
		if o == "no-new-privileges:true" {
			found = true
		}
	}
	if !found {
		t.Errorf("security_opt = %v", s.SecurityOpt)
	}
	if s.MemLimit == "" || s.CPUs <= 0 {
		t.Errorf("mem_limit=%q cpus=%v", s.MemLimit, s.CPUs)
	}
	if len(s.Healthcheck) == 0 {
		t.Error("healthcheck missing")
	}
}

func TestEnvExampleDocumentsEveryComposeVariable(t *testing.T) {
	raw, err := os.ReadFile("docker-compose.yml")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(".env.example")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	defined := map[string]bool{}
	prev := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if k, v, ok := strings.Cut(line, "="); ok && !strings.HasPrefix(line, "#") {
			defined[k] = true
			if !strings.HasPrefix(prev, "#") {
				t.Errorf("%s has no comment line directly above", k)
			}
			if strings.Contains(strings.ToLower(v), "inbox") {
				t.Errorf("%s example contains Inbox", k)
			}
		}
		prev = line
	}
	for _, m := range regexp.MustCompile(`\$\{([A-Z_]+)`).FindAllStringSubmatch(string(raw), -1) {
		if !defined[m[1]] {
			t.Errorf("compose variable %s missing in .env.example", m[1])
		}
	}
	for _, r := range []string{"PUID", "PGID", "PHOTOS_PATH", "DATA_PATH", "BACKUP_PATH", "PORT", "NOMINATIM_CONTACT", "TZ"} {
		if !defined[r] {
			t.Errorf("required variable %s missing", r)
		}
	}
}
