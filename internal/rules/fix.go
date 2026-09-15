package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// FixOptions controls which deterministic fixes run.
type FixOptions struct {
	// RecountOnly limits the run to the rule 19 children_counts rewrite.
	RecountOnly bool
	// Changed, when non-nil, restricts fixes to findings affected by the
	// changed paths (see AffectedByChange). A nil map fixes everything.
	Changed map[string]bool
}

// Fix applies the deterministic ADR 0004 fixes in place and returns the sorted
// vault-relative paths that were created or modified. Fixes never rename files
// (slug fixes stay report-only) and never touch unrelated content: the
// children_counts and related fields are the only frontmatter keys rewritten,
// and a missing folder note is created whole.
func (c *Context) Fix(opts FixOptions) ([]string, error) {
	changed := map[string]bool{}

	if !opts.RecountOnly {
		for _, dir := range c.missingFolderNoteDirs() {
			if !AffectedByChange(3, dir, opts.Changed) {
				continue
			}
			note := dir + "/" + lastSegment(dir) + ".md"
			if err := c.writeFile(note, c.newFolderNote(dir)); err != nil {
				return nil, err
			}
			changed[note] = true
		}
	}

	for _, dir := range sortedKeys(c.FolderNotes) {
		f := c.FolderNotes[dir]
		if !AffectedByChange(19, f.Path, opts.Changed) {
			continue
		}
		actual, ok := c.actualChildren(dir)
		if !ok {
			continue
		}
		nc, did := setFrontmatterField(f.Content, "children_counts", "children_counts: "+renderCounts(actual)+"\n")
		if !did {
			continue
		}
		if err := c.writeFile(f.Path, nc); err != nil {
			return nil, err
		}
		f.Content = nc
		changed[f.Path] = true
	}

	if !opts.RecountOnly {
		for _, f := range c.relatedFixTargets() {
			if !AffectedByChange(11, f.Path, opts.Changed) {
				continue
			}
			nc, did := normalizeRelatedField(f)
			if !did {
				continue
			}
			if err := c.writeFile(f.Path, nc); err != nil {
				return nil, err
			}
			f.Content = nc
			changed[f.Path] = true
		}
	}

	if !opts.RecountOnly {
		missing := c.missingMirrors()
		for _, bPath := range sortedKeys(missing) {
			if !AffectedByChange(25, bPath, opts.Changed) {
				continue
			}
			b := c.Vault.ByPath[bPath]
			if b == nil {
				continue
			}
			nc := b.Content
			for _, item := range missing[bPath] {
				var did bool
				nc, did = appendRelationsBodyItem(nc, item)
				if !did {
					break
				}
			}
			if nc == b.Content {
				continue
			}
			if err := c.writeFile(bPath, nc); err != nil {
				return nil, err
			}
			b.Content = nc
			changed[bPath] = true
		}
	}

	return sortedKeys(changed), nil
}

func (c *Context) writeFile(rel, content string) error {
	return os.WriteFile(filepath.Join(c.Vault.Root, filepath.FromSlash(rel)), []byte(content), 0o644)
}

// newFolderNote builds the folder note created by the R3 fix.
func (c *Context) newFolderNote(dir string) string {
	kind, scope := c.folderNoteMeta(dir)
	actual, ok := c.actualChildren(dir)
	if !ok || actual == nil {
		actual = map[string]int{}
	}
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "type: %s\n", kind)
	fmt.Fprintf(&b, "name: %s\n", titleCaseSlug(lastSegment(dir)))
	fmt.Fprintf(&b, "scope: %s\n", scope)
	b.WriteString("description: \"\"\n")
	b.WriteString("keywords: []\n")
	b.WriteString("aliases: []\n")
	b.WriteString("related: []\n")
	b.WriteString("status: active\n")
	fmt.Fprintf(&b, "created: %s\n", time.Now().Format("2006-01-02"))
	fmt.Fprintf(&b, "children_counts: %s\n", renderCounts(actual))
	b.WriteString("---\n")
	return b.String()
}

// folderNoteMeta classifies a dir for the R3 fix: the taxonomy kind when the
// dir is a category or entity dir, otherwise entity when the parent is an
// entity dir/category, otherwise collection. Scope is the top-level segment.
func (c *Context) folderNoteMeta(dir string) (kind, scope string) {
	kind = c.folderNoteKind(dir)
	segs := splitPath(dir)
	if len(segs) > 0 {
		scope = segs[0]
	}
	if kind == "" {
		if p := parentDir(dir); p != "" && c.folderNoteKind(p) == "entity" {
			kind = "entity"
		} else {
			kind = "collection"
		}
	}
	return kind, scope
}

// titleCaseSlug renders a kebab-case dir name as a title ("apollo-missing" ->
// "Apollo Missing").
func titleCaseSlug(s string) string {
	parts := strings.Split(s, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

// relatedFixTargets returns entries and folder notes, sorted by path.
func (c *Context) relatedFixTargets() []*vault.File {
	seen := map[string]bool{}
	var out []*vault.File
	add := func(f *vault.File) {
		if f == nil || seen[f.Path] {
			return
		}
		seen[f.Path] = true
		out = append(out, f)
	}
	for _, ei := range c.Entries {
		add(ei.File)
	}
	for _, dir := range sortedKeys(c.FolderNotes) {
		add(c.FolderNotes[dir])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// normalizeRelatedField rewrites the related field as a wikilink block list
// when any item is a bare path. Items already bracketed (valid or malformed)
// are left for the report-only R11 finding.
func normalizeRelatedField(f *vault.File) (string, bool) {
	items := stringList(f.Frontmatter, "related")
	if len(items) == 0 {
		return f.Content, false
	}
	fixed := false
	out := make([]string, 0, len(items))
	for _, item := range items {
		if isWikiLink(item) || strings.ContainsAny(item, "[]") {
			out = append(out, item)
			continue
		}
		out = append(out, "[["+strings.TrimSpace(item)+"]]")
		fixed = true
	}
	if !fixed {
		return f.Content, false
	}
	var b strings.Builder
	b.WriteString("related:\n")
	for _, item := range out {
		fmt.Fprintf(&b, "  - %q\n", item)
	}
	return setFrontmatterField(f.Content, "related", b.String())
}

// AffectedByChange reports whether a finding for the given rule/path is in
// scope for a --changed run. A nil map means every path is in scope. File
// findings match exactly; rule 3 reports on a dir and rule 19 on a folder
// note, so they also match when any changed file lives under their dir.
func AffectedByChange(rule int, path string, changed map[string]bool) bool {
	if changed == nil {
		return true
	}
	if changed[path] {
		return true
	}
	var prefix string
	switch rule {
	case 3:
		prefix = path + "/"
	case 19:
		prefix = parentDir(path) + "/"
	default:
		return false
	}
	for p := range changed {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

// FilterByChanged keeps only findings affected by the changed set.
func FilterByChanged(fs []vault.Finding, changed map[string]bool) []vault.Finding {
	if changed == nil {
		return fs
	}
	out := make([]vault.Finding, 0, len(fs))
	for _, f := range fs {
		if AffectedByChange(f.Rule, f.Path, changed) {
			out = append(out, f)
		}
	}
	return out
}

// setFrontmatterField replaces the top-level key line and its YAML
// continuation lines with replacement (which must carry its own trailing
// newline and start with "key:"), or inserts replacement before the closing
// "---" when the key is absent. The rest of the file is preserved byte for
// byte. The bool reports whether content actually changed.
func setFrontmatterField(content, key, replacement string) (string, bool) {
	lines := strings.SplitAfter(content, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r\n") != "---" {
		return content, false
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r\n") == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return content, false
	}
	keyLine := -1
	for i := 1; i < end; i++ {
		if matchTopLevelKey(lines[i], key) {
			keyLine = i
			break
		}
	}
	out := make([]string, 0, len(lines)+1)
	if keyLine >= 0 {
		k := keyLine + 1
		for k < end && isYAMLContinuation(lines[k]) {
			k++
		}
		out = append(out, lines[:keyLine]...)
		out = append(out, replacement)
		out = append(out, lines[k:]...)
	} else {
		out = append(out, lines[:end]...)
		out = append(out, replacement)
		out = append(out, lines[end:]...)
	}
	nc := strings.Join(out, "")
	return nc, nc != content
}

func matchTopLevelKey(line, key string) bool {
	t := strings.TrimRight(line, "\r\n")
	return t == key || strings.HasPrefix(t, key+":")
}

// appendRelationsBodyItem appends a "- <item>" markdown line to the target
// note's body `## Relations` section, after its last top-level list item (or
// directly under the heading when the section has none). When the section is
// absent it is created at the very end of the file, separated by a blank line.
// Existing content is preserved byte for byte; the bool reports whether
// content changed.
func appendRelationsBodyItem(content, item string) (string, bool) {
	line := "- " + item
	lines := strings.SplitAfter(content, "\n")
	heading := -1
	for i, l := range lines {
		if relationsHeadingRe.MatchString(strings.TrimSpace(l)) {
			heading = i
			break
		}
	}
	if heading < 0 {
		return appendRelationsSection(content, line), true
	}
	end := len(lines)
	for i := heading + 1; i < len(lines); i++ {
		if h2StartRe.MatchString(strings.TrimSpace(lines[i])) {
			end = i
			break
		}
	}
	insert := heading + 1
	for i := heading + 1; i < end; i++ {
		if listItemRe.MatchString(strings.TrimSpace(lines[i])) {
			insert = i + 1
		}
	}
	var b strings.Builder
	for i := 0; i < insert; i++ {
		b.WriteString(lines[i])
	}
	if insert > 0 && !strings.HasSuffix(lines[insert-1], "\n") {
		b.WriteString("\n")
	}
	b.WriteString(line + "\n")
	for i := insert; i < len(lines); i++ {
		b.WriteString(lines[i])
	}
	nc := b.String()
	return nc, nc != content
}

// appendRelationsSection creates the `## Relations` section at the end of a
// note with a blank-line separation from the existing content.
func appendRelationsSection(content, line string) string {
	var b strings.Builder
	b.WriteString(content)
	switch {
	case content == "":
	case strings.HasSuffix(content, "\n"):
		b.WriteString("\n")
	default:
		b.WriteString("\n\n")
	}
	b.WriteString("## Relations\n\n")
	b.WriteString(line + "\n")
	return b.String()
}

func isYAMLContinuation(line string) bool {
	t := strings.TrimRight(line, "\r\n")
	if t == "" {
		return false
	}
	if strings.HasPrefix(t, " ") || strings.HasPrefix(t, "\t") {
		return true
	}
	return t == "-" || strings.HasPrefix(t, "- ")
}
