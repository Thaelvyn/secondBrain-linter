package rules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// relationEdge is a parsed, well-formed relation item on a person folder note.
type relationEdge struct {
	kind   string
	target string // canonical resolved folder-note file path
}

// relationEdges returns the well-formed, taxonomy-known, resolvable relation
// edges of a folder note's body `## Relations` section. Malformed items and
// invalid targets are skipped here and reported by rule 24.
func (c *Context) relationEdges(f *vault.File) []relationEdge {
	if f == nil || c.Tax == nil {
		return nil
	}
	var out []relationEdge
	for _, item := range relationsBodyItems(f.Content) {
		m := relationItemRe.FindStringSubmatch(item)
		if m == nil {
			continue
		}
		kind := m[1]
		if _, ok := c.Tax.RelationKinds[kind]; !ok {
			continue
		}
		target := resolveWiki(wikiTarget(m[2]), c.Vault)
		if target == "" || target == f.Path {
			continue
		}
		if !c.isPersonNote(target) {
			continue
		}
		out = append(out, relationEdge{kind: kind, target: target})
	}
	return out
}

// mirrorKind returns the kind B must use to mirror an A —K→ B edge: K itself
// when K is symmetric, otherwise inverse(K). ok is false when no mirror kind
// is defined (kind unknown, or neither symmetric nor inverse).
func (c *Context) mirrorKind(kind string) (string, bool) {
	k, ok := c.Tax.RelationKinds[kind]
	if !ok {
		return "", false
	}
	if k.Symmetric {
		return kind, true
	}
	if k.Inverse != "" {
		return k.Inverse, true
	}
	return "", false
}

// hasEdge reports whether f's body Relations already carry an edge of kind
// pointing at the canonical target path.
func (c *Context) hasEdge(f *vault.File, kind, target string) bool {
	for _, e := range c.relationEdges(f) {
		if e.kind == kind && e.target == target {
			return true
		}
	}
	return false
}

// missingMirrors returns, per folder-note path, the sorted list of relation
// items to append so every inbound person edge is mirrored. Items render as
// "<kind> [[<source>]]" with the source folder-note file path (no .md).
func (c *Context) missingMirrors() map[string][]string {
	out := map[string][]string{}
	if c.Tax == nil {
		return out
	}
	for _, aDir := range sortedKeys(c.FolderNotes) {
		if !c.isPersonDir(aDir) {
			continue
		}
		a := c.FolderNotes[aDir]
		for _, e := range c.relationEdges(a) {
			mk, ok := c.mirrorKind(e.kind)
			if !ok {
				continue
			}
			b := c.Vault.ByPath[e.target]
			if b == nil {
				continue
			}
			if c.hasEdge(b, mk, a.Path) {
				continue
			}
			item := mk + " [[" + strings.TrimSuffix(a.Path, ".md") + "]]"
			out[b.Path] = appendUnique(out[b.Path], item)
		}
	}
	for p := range out {
		sort.Strings(out[p])
	}
	return out
}

func appendUnique(list []string, s string) []string {
	for _, e := range list {
		if e == s {
			return list
		}
	}
	return append(list, s)
}

// rule25: every people relation edge A —K→ B in a body `## Relations` section
// must be mirrored on B — the symmetric kind itself or inverse(K) — pointing
// back at A (canonical resolved targets, file-only resolver). A missing mirror
// is a completeness-drift warning (same class as R19 children_counts drift)
// and is repaired by --fix: the missing item is appended to B's `## Relations`
// section, created at the end of the file when absent. The finding message
// follows the ADR template: "B lacks '<kind> [[<A>]]'" where B and A are the
// folder-note file paths without .md.
func rule25(c *Context) {
	for bPath, items := range c.missingMirrors() {
		for _, item := range items {
			c.Add(25, vault.SeverityWarning, bPath,
				fmt.Sprintf("%s lacks '%s'", strings.TrimSuffix(bPath, ".md"), item))
		}
	}
}
