// Package rules implements the sblint rule set (ADR 0004 rules 1-18).
package rules

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Thaelvyn/secondBrain-linter/internal/taxonomy"
	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

var (
	datePrefixRe  = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9][a-z0-9-]*\.md$`)
	yearDirRe     = regexp.MustCompile(`^[0-9]{4}$`)
	eventIDRe     = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}-[0-9a-f]{12}$`)
	dateRe        = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	wikiRe        = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	slugRe        = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*$`)
	structuralSet = map[string]bool{"_system": true, "daily-logs": true, "calendar": true, "inbox": true, "attachments": true}
	knownRootFile = map[string]bool{"todos.md": true, "README.md": true}
)

// RequiredEntryFields is ADR 0004 rule 5 / ADR 0002 decision 2.
var requiredEntryFields = []string{"event_id", "date", "type", "perspective", "privacy", "entities", "origin", "source", "summary"}

// EntryInfo couples an entry file with its parsed frontmatter.
type EntryInfo struct {
	File *vault.File
	Dir  string // parent dir relative to vault root
}

// Context carries everything the rules need. It is assembled once, then each
// registered rule reads from it and appends findings.
type Context struct {
	Vault *vault.Vault
	Tax   *taxonomy.Taxonomy

	Entries        []*EntryInfo
	FolderNotes    map[string]*vault.File // dir -> its folder note
	EventGroups    map[string][]*EntryInfo
	EntriesBeneath map[string]int             // dir -> count of entries in its subtree
	RelatedLinks   map[string]map[string]bool // file path -> resolved related targets

	typeFamilies map[string]bool
	Findings     []vault.Finding
}

// New assembles the context from the scanned vault and taxonomy.
func New(v *vault.Vault, tax *taxonomy.Taxonomy) *Context {
	c := &Context{
		Vault: v, Tax: tax,
		FolderNotes:    map[string]*vault.File{},
		EventGroups:    map[string][]*EntryInfo{},
		EntriesBeneath: map[string]int{},
		RelatedLinks:   map[string]map[string]bool{},
	}
	if tax != nil {
		c.typeFamilies = map[string]bool{}
		for t := range tax.EventTypes {
			c.typeFamilies[t] = true
			c.typeFamilies[t+"s"] = true
		}
	}
	for _, f := range v.Files {
		if isFolderNote(f) {
			c.FolderNotes[parentDir(f.Path)] = f
		}
		eid, ok := fmString(f.Frontmatter, "event_id")
		if f.Markdown && ok && eid != "" {
			ei := &EntryInfo{File: f, Dir: parentDir(f.Path)}
			c.Entries = append(c.Entries, ei)
			c.EventGroups[eid] = append(c.EventGroups[eid], ei)
			for _, d := range ancestorDirs(f.Path) {
				c.EntriesBeneath[d]++
			}
			c.RelatedLinks[f.Path] = resolveRelated(f, v)
		}
	}
	sort.Slice(c.Entries, func(i, j int) bool { return c.Entries[i].File.Path < c.Entries[j].File.Path })
	return c
}

// Add appends a finding.
func (c *Context) Add(rule int, sev vault.Severity, path, message string) {
	c.Findings = append(c.Findings, vault.Finding{Rule: rule, Severity: sev, Path: path, Message: message})
}

// isFolderNote reports whether f is a folder note: its base name equals its
// parent dir name plus ".md" (Obsidian convention, used across the vault).
func isFolderNote(f *vault.File) bool {
	if !f.Markdown {
		return false
	}
	dir := parentDir(f.Path)
	if dir == "" {
		return false
	}
	return f.Name == lastSegment(dir)+".md"
}

func parentDir(p string) string {
	i := strings.LastIndexByte(p, '/')
	if i < 0 {
		return ""
	}
	return p[:i]
}

func lastSegment(p string) string {
	i := strings.LastIndexByte(p, '/')
	if i < 0 {
		return p
	}
	return p[i+1:]
}

// ancestorDirs returns every ancestor dir of a file path, from the top-level
// down to the parent dir.
func ancestorDirs(p string) []string {
	dir := parentDir(p)
	if dir == "" {
		return nil
	}
	segs := strings.Split(dir, "/")
	out := make([]string, 0, len(segs))
	for i := 1; i <= len(segs); i++ {
		out = append(out, strings.Join(segs[:i], "/"))
	}
	return out
}

func (c *Context) isStructuralPath(dir string) bool {
	top := dir
	if i := strings.IndexByte(dir, '/'); i >= 0 {
		top = dir[:i]
	}
	return structuralSet[top]
}

// resolveRelated parses the related frontmatter of an entry (or folder note)
// and returns the resolved, existing targets (canonical file paths).
func resolveRelated(f *vault.File, v *vault.Vault) map[string]bool {
	out := map[string]bool{}
	for _, item := range stringList(f.Frontmatter, "related") {
		if !isWikiLink(item) {
			continue
		}
		if resolved := resolveWiki(wikiTarget(item), v); resolved != "" {
			out[resolved] = true
		}
	}
	return out
}

// isWikiLink reports whether s is a full wikilink.
func isWikiLink(s string) bool { return strings.HasPrefix(s, "[[") && strings.HasSuffix(s, "]]") }

// wikiTarget strips the surrounding brackets, alias (|...) and anchor
// (#...) from a raw wikilink.
func wikiTarget(s string) string {
	t := strings.TrimSpace(s)
	t = strings.TrimPrefix(t, "[[")
	t = strings.TrimSuffix(t, "]]")
	if i := strings.IndexByte(t, '|'); i >= 0 {
		t = t[:i]
	}
	if i := strings.IndexByte(t, '#'); i >= 0 {
		t = t[:i]
	}
	return t
}

// resolveWiki resolves a target relative to the vault root: tried as-is, with
// ".md" appended, and as a folder reference resolving to the folder's note
// ({target}/{basename}.md, per the ADR 0002 link conventions). Returns the
// canonical file path, or "".
func resolveWiki(target string, v *vault.Vault) string {
	t := strings.TrimSpace(target)
	if i := strings.IndexByte(t, '#'); i >= 0 {
		t = t[:i]
	}
	t = strings.TrimSuffix(t, "/")
	if t == "" {
		return ""
	}
	if _, ok := v.ByPath[t]; ok {
		return t
	}
	if _, ok := v.ByPath[t+".md"]; ok {
		return t + ".md"
	}
	if v.DirSet[t] {
		note := t + "/" + lastSegment(t) + ".md"
		if _, ok := v.ByPath[note]; ok {
			return note
		}
	}
	return ""
}

// validSlug enforces the ADR 0004 rule 1 charset: lowercase, ASCII,
// kebab-case (hyphens and dots allowed), no spaces, no underscores.
func validSlug(s string) bool { return slugRe.MatchString(s) }

func validDate(s string) bool {
	if !dateRe.MatchString(s) {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// fmString returns a scalar frontmatter value as a string. yaml.v3 resolves
// bare dates like 2026-09-10 into time.Time, which is normalized back to the
// YYYY-MM-DD form.
func fmString(fm map[string]any, key string) (string, bool) {
	v, ok := fm[key]
	if !ok || v == nil {
		return "", false
	}
	switch t := v.(type) {
	case string:
		return t, true
	case time.Time:
		return t.Format("2006-01-02"), true
	}
	return "", false
}

// stringList normalizes a frontmatter field that may be a single string or a
// YAML list of strings.
func stringList(fm map[string]any, key string) []string {
	v, ok := fm[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		return []string{t}
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}

// sortedKeys returns the sorted keys of a string-keyed map.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
