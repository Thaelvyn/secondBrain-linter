package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// peopleCategory is the taxonomy category whose entity folders are people.
const peopleCategory = "people"

// relationsHeadingRe matches the optional `## Relations` body heading; a
// trailing inline comment (e.g. `<!-- optional -->`) is tolerated on the
// heading text like rule 11's `## Related`. The trailing group keeps it from
// matching `## Related` (and rule 11's regex from matching `## Relations`).
var relationsHeadingRe = regexp.MustCompile(`^##\s+Relations(?:\s|$)`)

// relationItemRe matches the machine-parsed prefix of a body Relations item:
// a lowercase kind, whitespace, then a full wikilink. Any prose after the
// closing `]]` is human free text and is ignored by the rules.
var relationItemRe = regexp.MustCompile(`^([a-z][a-z0-9_]*)\s+\[\[(.+?)\]\](.*)$`)

// relationsBodyItems returns the trimmed text of every top-level list item in
// the optional `## Relations` body section, or nil when the heading is absent.
// The section runs from the heading line to the next H2 heading or EOF. The
// heading line itself and non-list lines (blank lines, prose) are ignored.
func relationsBodyItems(content string) []string {
	lines := strings.Split(bodyOf(content), "\n")
	start := -1
	for i, line := range lines {
		if relationsHeadingRe.MatchString(strings.TrimSpace(line)) {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return nil
	}
	var out []string
	for _, line := range lines[start:] {
		if h2StartRe.MatchString(line) {
			break
		}
		if m := listItemRe.FindStringSubmatch(line); m != nil {
			out = append(out, strings.TrimSpace(m[1]))
		}
	}
	return out
}

// isPersonDir reports whether dir is an entity folder directly under a
// `people` category of kind entity in its scope: personal/people/{entity} or
// work/{company|_shared}/people/{entity}.
func (c *Context) isPersonDir(dir string) bool {
	if c.Tax == nil || c.folderNoteKind(dir) != "entity" {
		return false
	}
	segs := splitPath(dir)
	switch len(segs) {
	case 3:
		if segs[1] != peopleCategory {
			return false
		}
		if scope, ok := c.Tax.Scopes[segs[0]]; ok {
			cat, ok := scope.Categories[peopleCategory]
			return ok && cat.Kind == "entity"
		}
	case 4:
		if segs[0] != "work" || segs[2] != peopleCategory {
			return false
		}
		if scope, ok := c.Tax.Scopes["work"]; ok {
			cat, ok := scope.Categories[peopleCategory]
			return ok && cat.Kind == "entity"
		}
	}
	return false
}

// isPersonNote reports whether path resolves to a folder note whose frontmatter
// declares `type: entity` (a person note).
func (c *Context) isPersonNote(path string) bool {
	f := c.Vault.ByPath[path]
	if f == nil || !isFolderNote(f) {
		return false
	}
	t, _ := fmString(f.Frontmatter, "type")
	return t == "entity"
}

// rule24: person folder notes carry an optional body `## Relations` section
// whose list items are typed edges: each item's machine-parsed prefix is
// "<kind> [[<target>]]" (trailing prose ignored), the kind exists in the
// taxonomy relation_kinds catalog and the target resolves (file-only resolver)
// to a person folder note (`type: entity`), not the note itself. `met` and
// `self` must be booleans and only appear on person-type folder notes (`type:
// entity`); all `self: true` notes across the vault must share a single owner
// `name` and list every other self note in `related`. All findings are errors;
// there is no --fix path.
func rule24(c *Context) {
	if c.Tax == nil || len(c.Tax.RelationKinds) == 0 {
		return
	}
	var selfNotes []*vault.File
	for _, dir := range sortedKeys(c.FolderNotes) {
		f := c.FolderNotes[dir]
		personDir := c.isPersonDir(dir)
		if personDir {
			c.checkRelations(f)
		}
		kind, _ := fmString(f.Frontmatter, "type")
		personType := personDir && kind == "entity"
		for _, key := range []string{"met", "self"} {
			if _, ok := f.Frontmatter[key]; !ok {
				continue
			}
			if !personType {
				c.Add(24, vault.SeverityError, f.Path, key+" is only allowed on person folder notes")
				continue
			}
			b, isBool := f.Frontmatter[key].(bool)
			if !isBool {
				c.Add(24, vault.SeverityError, f.Path, key+" must be a boolean")
				continue
			}
			if key == "self" && b {
				selfNotes = append(selfNotes, f)
			}
		}
	}
	c.checkSelfNotes(selfNotes)
}

// checkRelations validates the body `## Relations` items of a person folder
// note. Only the item prefix is parsed: text after the closing `]]` is free
// prose and is ignored.
func (c *Context) checkRelations(f *vault.File) {
	seen := map[string]bool{}
	for _, item := range relationsBodyItems(f.Content) {
		m := relationItemRe.FindStringSubmatch(item)
		if m == nil {
			c.Add(24, vault.SeverityError, f.Path,
				fmt.Sprintf("malformed relation %q: expected '<kind> [[<target>]]'", item))
			continue
		}
		kind, rawTarget := m[1], m[2]
		prefix := kind + " [[" + rawTarget + "]]"
		if _, ok := c.Tax.RelationKinds[kind]; !ok {
			c.Add(24, vault.SeverityError, f.Path, fmt.Sprintf("unknown relation kind %q", kind))
			continue
		}
		target := resolveWiki(wikiTarget(rawTarget), c.Vault)
		if target == "" {
			c.Add(24, vault.SeverityError, f.Path,
				fmt.Sprintf("relation target %q does not resolve to an existing file", rawTarget))
			continue
		}
		if target == f.Path {
			c.Add(24, vault.SeverityError, f.Path, fmt.Sprintf("relation %q is a self-loop", prefix))
			continue
		}
		if !c.isPersonNote(target) {
			c.Add(24, vault.SeverityError, f.Path,
				fmt.Sprintf("relation target %q must resolve to a person folder note (type: entity)", rawTarget))
			continue
		}
		pair := kind + "\x00" + target
		if seen[pair] {
			c.Add(24, vault.SeverityError, f.Path, fmt.Sprintf("duplicate relation %q", prefix))
			continue
		}
		seen[pair] = true
	}
}

// checkSelfNotes enforces the self-owner invariants: all `self: true` notes
// share one `name` (a single owner identity), and every self note lists every
// other self note in `related`.
func (c *Context) checkSelfNotes(notes []*vault.File) {
	if len(notes) < 2 {
		return
	}
	ref := notes[0]
	refName, _ := fmString(ref.Frontmatter, "name")
	for _, f := range notes[1:] {
		name, _ := fmString(f.Frontmatter, "name")
		if name != refName {
			c.Add(24, vault.SeverityError, f.Path,
				fmt.Sprintf("self note name %q does not match owner name %q (at most one owner)", name, refName))
		}
	}
	for _, f := range notes {
		linked := resolveRelated(f, c.Vault)
		for _, other := range notes {
			if other.Path == f.Path {
				continue
			}
			if !linked[other.Path] {
				c.Add(24, vault.SeverityError, f.Path,
					fmt.Sprintf("self note must list every other self note in related: missing %s", other.Path))
			}
		}
	}
}
