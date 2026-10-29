package probe

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// MutateKeepFile is changed with its mtime preserved; MutateNewFile gets a fresh mtime.
	MutateKeepFile = "pcmut-keep.jpg"
	MutateNewFile  = "pcmut-new.jpg"

	mutatePrefix = "pcprobe-jpeg-mutate-"
)

// MutateFile is a detection-test file and the marker it carries in XMP-dc:Subject at generation.
type MutateFile struct {
	File   string
	Marker string
	Keep   bool // true: mtime is preserved on mutation
}

// MutateFiles lists the detection-test files; they are not part of the field matrix.
func MutateFiles() []MutateFile {
	return []MutateFile{
		{MutateKeepFile, mutatePrefix + "keep-v1", true},
		{MutateNewFile, mutatePrefix + "new-v1", false},
	}
}

// Mutate bumps the marker version of both files. It validates both files first and changes
// nothing if either is missing or carries a foreign marker.
func Mutate(dir string, out io.Writer) error {
	type plan struct {
		f        MutateFile
		old, new string
	}
	var plans []plan
	for _, f := range MutateFiles() {
		path := filepath.Join(dir, f.File)
		dump, err := ReadTags(path)
		if err != nil {
			return fmt.Errorf("%s: %w", f.File, err)
		}
		var old string
		for _, e := range dump {
			if e.Group == "XMP-dc" && e.Name == "Subject" {
				old = e.Value
			}
		}
		base, ver, ok := strings.Cut(old, "-v")
		n, convErr := strconv.Atoi(ver)
		if !strings.HasPrefix(old, mutatePrefix) || !ok || convErr != nil || n < 1 {
			return fmt.Errorf("%s: unexpected marker %q", f.File, old)
		}
		plans = append(plans, plan{f, old, base + "-v" + strconv.Itoa(n+1)})
	}
	// Not atomic: if a write fails, files already written stay mutated. Re-running then
	// continues from the next marker version of each file.
	for _, p := range plans {
		head := []string{"-overwrite_original"}
		if p.f.Keep {
			head = append(head, "-P")
		}
		assigns := []TagValue{{"XMP-dc:Subject-", p.old}, {"XMP-dc:Subject+", p.new}}
		if err := writeTagsWith(filepath.Join(dir, p.f.File), head, assigns); err != nil {
			return err
		}
		fmt.Fprintf(out, "%s: %s -> %s\n", p.f.File, p.old, p.new)
	}
	return nil
}

func runMutate(args []string, stderr io.Writer) int {
	fl := flag.NewFlagSet("probe mutate", flag.ContinueOnError)
	fl.SetOutput(stderr)
	dir := fl.String("dir", "", "directory with the generated probe files")
	if err := fl.Parse(args); err != nil {
		return 2
	}
	if *dir == "" {
		fmt.Fprint(stderr, "--dir is required\n"+usage)
		return 2
	}
	if err := Mutate(*dir, stderr); err != nil {
		fmt.Fprintf(stderr, "probe mutate: %v\n", err)
		return 1
	}
	return 0
}
