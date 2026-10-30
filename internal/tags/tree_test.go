package tags

import (
	"encoding/json"
	"strings"
	"testing"
)

type pair struct {
	photo int64
	path  string
}

func seq(ps ...pair) func(func(int64, string) bool) {
	return func(yield func(int64, string) bool) {
		for _, p := range ps {
			if !yield(p.photo, p.path) {
				return
			}
		}
	}
}

func TestAPhotoCarryingTwoTagsUnderOneParentCountsOnceForTheParent(t *testing.T) {
	tree := BuildTree(nil, seq(pair{1, "Anlass/Geburtstag"}, pair{1, "Anlass/Hochzeit"}, pair{2, "Anlass/Hochzeit"}))
	if len(tree) != 1 || tree[0].Path != "Anlass" || tree[0].Count != 2 {
		t.Fatalf("root = %+v", tree)
	}
	kids := tree[0].Children
	if len(kids) != 2 || kids[0].Name != "Geburtstag" || kids[0].Count != 1 || kids[1].Name != "Hochzeit" || kids[1].Count != 2 {
		t.Fatalf("children = %+v", kids)
	}
}

func TestAnEmptyDefinitionAppearsWithCountZeroAndItsParents(t *testing.T) {
	tree := BuildTree([]string{"Reise/Italien"}, seq())
	if len(tree) != 1 || tree[0].Path != "Reise" || tree[0].Count != 0 {
		t.Fatalf("tree = %+v", tree)
	}
	if c := tree[0].Children; len(c) != 1 || c[0].Path != "Reise/Italien" || c[0].Count != 0 {
		t.Fatalf("children = %+v", c)
	}
}

func TestLeavesMarshalChildrenAsEmptyArray(t *testing.T) {
	b, err := json.Marshal(BuildTree([]string{"A"}, seq()))
	if err != nil || !strings.Contains(string(b), `"children":[]`) {
		t.Fatalf("json = %s, err = %v", b, err)
	}
}

func TestIndexPathsAreSplitLeniently(t *testing.T) {
	tree := BuildTree(nil, seq(pair{1, "/A//b/"}))
	if len(tree) != 1 || tree[0].Path != "A" || tree[0].Children[0].Path != "A/b" {
		t.Fatalf("tree = %+v", tree)
	}
}
