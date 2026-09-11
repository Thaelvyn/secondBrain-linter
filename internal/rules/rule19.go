package rules

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule19: every entity/collection folder note's children_counts map equals the
// actual direct-child counts (ADR 0004 rule 19, warning). See folderNoteKind
// for the exact scoping. One finding per mismatching key; a missing
// children_counts field is one finding for the whole note.
func rule19(c *Context) {
	for _, dir := range sortedKeys(c.FolderNotes) {
		actual, ok := c.actualChildren(dir)
		if !ok {
			continue
		}
		f := c.FolderNotes[dir]
		stored, present := fmIntMap(f.Frontmatter, "children_counts")
		if !present {
			c.Add(19, vault.SeverityWarning, f.Path,
				fmt.Sprintf("children_counts is missing; expected %s", renderCounts(actual)))
			continue
		}
		for _, key := range unionKeys(stored, actual) {
			if stored[key] != actual[key] {
				c.Add(19, vault.SeverityWarning, f.Path,
					fmt.Sprintf("children_counts[%q]: stored %d, actual %d", key, stored[key], actual[key]))
			}
		}
	}
}

// renderCounts renders an actual children_counts map deterministically, in
// inline YAML form. Keys that would not round-trip as YAML strings (notably
// 4-digit year dirs, which yaml.v3 turns into integer mapping keys) are
// double-quoted.
func renderCounts(m map[string]int) string {
	if len(m) == 0 {
		return "{}"
	}
	keys := sortedKeys(m)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %d", yamlKey(k), m[k]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func yamlKey(k string) string {
	if yamlSafeKeyRe.MatchString(k) {
		return k
	}
	return strconv.Quote(k)
}

// unionKeys returns the sorted union of two string-keyed maps.
func unionKeys(a, b map[string]int) []string {
	seen := map[string]bool{}
	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}
	return sortedKeys(seen)
}
