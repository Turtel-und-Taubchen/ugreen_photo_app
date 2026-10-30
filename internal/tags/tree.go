package tags

import (
	"sort"
	"strings"
)

// Node is one tag in the tree. Count is the number of distinct photos carrying
// the tag or any descendant.
type Node struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Count    int    `json:"count"`
	Children []Node `json:"children"`
}

type entry struct {
	count    int
	last     int64
	counted  bool
	children map[string]struct{}
}

// BuildTree returns the root nodes for all paths in defs and pairs, including
// every ancestor. pairs must yield rows ordered by photoID, which lets each
// node remember only the last photo it counted. Paths are split on "/" without
// validation; empty segments are skipped. Siblings are in byte order.
func BuildTree(defs []string, pairs func(yield func(photoID int64, path string) bool)) []Node {
	nodes := map[string]*entry{"": {children: map[string]struct{}{}}}
	// ensure registers every ancestor of path and returns the path chain.
	ensure := func(path string) []string {
		var chain []string
		cur := ""
		for _, seg := range strings.Split(path, "/") {
			if seg == "" {
				continue
			}
			parent := cur
			if cur == "" {
				cur = seg
			} else {
				cur += "/" + seg
			}
			if _, ok := nodes[cur]; !ok {
				nodes[cur] = &entry{children: map[string]struct{}{}}
			}
			nodes[parent].children[seg] = struct{}{}
			chain = append(chain, cur)
		}
		return chain
	}
	for _, d := range defs {
		ensure(d)
	}
	pairs(func(photoID int64, path string) bool {
		for _, p := range ensure(path) {
			e := nodes[p]
			if !e.counted || e.last != photoID {
				e.count++
				e.last = photoID
				e.counted = true
			}
		}
		return true
	})
	var build func(parent string) []Node
	build = func(parent string) []Node {
		names := make([]string, 0, len(nodes[parent].children))
		for n := range nodes[parent].children {
			names = append(names, n)
		}
		sort.Strings(names)
		out := make([]Node, 0, len(names))
		for _, n := range names {
			p := n
			if parent != "" {
				p = parent + "/" + n
			}
			out = append(out, Node{Name: n, Path: p, Count: nodes[p].count, Children: build(p)})
		}
		return out
	}
	return build("")
}
