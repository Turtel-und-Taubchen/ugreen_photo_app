package metadata

import (
	"fmt"
	"slices"
	"strings"
)

// TagTerms splits a "Parent/Child" tag path into its hierarchy value ("Parent|Child") and its
// terms (Parent, Child).
func TagTerms(path string) (hier string, terms []string) {
	terms = segments(path)
	return strings.Join(terms, "|"), terms
}

// hierValue normalises a stored hierarchy value the way Parse reads it.
func hierValue(raw string) string {
	var parts []string
	for _, p := range strings.Split(raw, "|") {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "|")
}

// TagArgs builds the ExifTool arguments (without file path) that add and remove tag paths on
// a file of format whose freshly read ExifTool fields ("Group:Tag" -> value, as Parse takes
// them) are fields. Only the difference is written: a value a field already holds is not added
// again, and a value no field holds is not removed, so an empty result means the file is
// already current. Values other tools wrote are never touched.
func TagArgs(format string, fields map[string]any, add, remove []string) ([]string, error) {
	rules, ok := table[format]
	if !ok {
		return nil, fmt.Errorf("unknown format %q", format)
	}
	for _, tag := range append(slices.Clone(add), remove...) {
		if len(segments(tag)) == 0 {
			return nil, fmt.Errorf("empty tag %q", tag)
		}
	}
	var args []string
	emit := func(tag, op, value string) {
		args = append(args, "-"+tag+op+value)
	}

	removing := map[string]bool{}
	for _, tag := range remove {
		removing[strings.Join(segments(tag), "/")] = true
	}

	// held is what each target field carries, including values added by this call, so a term
	// shared by two added paths is written once.
	held := map[string][]string{}
	current := func(w write) []string {
		if _, ok := held[w.Tag]; !ok {
			v, _ := lookup(fields, w.Tag)
			var vals []string
			if w.Src == srcPath {
				for _, raw := range values(v) {
					vals = append(vals, hierValue(raw))
				}
			} else {
				vals = keywords(fields, w.Tag)
			}
			held[w.Tag] = vals
		}
		return held[w.Tag]
	}
	for _, tag := range add {
		hier, terms := TagTerms(tag)
		// A path both added and removed ends up removed, as Expect has it.
		if removing[strings.Join(terms, "/")] {
			continue
		}
		for _, w := range rules[fieldTags].Add {
			want := terms
			if w.Src == srcPath {
				want = []string{hier}
			}
			for _, v := range want {
				if !slices.Contains(current(w), v) {
					emit(w.Tag, w.Op, v)
					held[w.Tag] = append(held[w.Tag], v)
				}
			}
		}
	}
	if len(remove) == 0 {
		return args, nil
	}

	// A term stays wherever it is while any remaining tag needs it: as a segment of a remaining
	// path, or as a remaining flat tag.
	needed := map[string]bool{}
	for _, tag := range remainingTags(Parse(fields), Change{AddTags: add, RemoveTags: remove}) {
		for _, s := range segments(tag) {
			needed[s] = true
		}
	}
	gone := map[string]bool{}
	done := map[string]bool{}
	once := func(tag, op, value string) {
		if k := tag + "\x00" + value; !done[k] {
			done[k] = true
			emit(tag, op, value)
		}
	}
	for _, tag := range remove {
		hier, terms := TagTerms(tag)
		// Exactly this path; deeper paths below it are other tags and stay.
		for _, w := range rules[fieldTags].Remove {
			v, _ := lookup(fields, w.Tag)
			for _, raw := range values(v) {
				if hierValue(raw) == hier {
					once(w.Tag, w.Op, raw)
				}
			}
		}
		for _, s := range terms {
			if needed[s] {
				continue
			}
			gone[s] = true
			for _, k := range KeywordTags {
				if k != xpKeywords && slices.Contains(keywords(fields, k), s) {
					once(k, opRemove, s)
				}
			}
		}
	}
	// XPKeywords is one string, so -= cannot reach a single entry; it is rewritten with the
	// other entries kept in their order, and only when an entry actually goes.
	if xp := keywords(fields, xpKeywords); len(xp) > 0 {
		rest := slices.DeleteFunc(slices.Clone(xp), func(s string) bool { return gone[s] })
		if len(rest) != len(xp) {
			emit(xpKeywords, opSet, strings.Join(rest, ";"))
		}
	}
	return args, nil
}
