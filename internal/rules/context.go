// Package rules implements the sblint rule set (ADR 0004 rules 1-22).
package rules

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
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
	yamlSafeKeyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*$`)
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
	EntriesDirect  map[string]int             // dir -> count of entries directly in it
	RelatedLinks   map[string]map[string]bool // file path -> resolved related targets

	typeFamilies       map[string]bool
	collectionFamilies map[string]bool
	childDirs          map[string][]string // dir -> immediate child dir basenames
	Findings           []vault.Finding
}

// New assembles the context from the scanned vault and taxonomy.
func New(v *vault.Vault, tax *taxonomy.Taxonomy) *Context {
	c := &Context{
		Vault: v, Tax: tax,
		FolderNotes:    map[string]*vault.File{},
		EventGroups:    map[string][]*EntryInfo{},
		EntriesBeneath: map[string]int{},
		EntriesDirect:  map[string]int{},
		RelatedLinks:   map[string]map[string]bool{},
		childDirs:      map[string][]string{},
	}
	if tax != nil {
		c.typeFamilies = map[string]bool{}
		for t := range tax.EventTypes {
			c.typeFamilies[t] = true
			c.typeFamilies[t+"s"] = true
		}
		c.collectionFamilies = map[string]bool{}
		for _, s := range tax.Scopes {
			for name, cat := range s.Categories {
				if cat.Kind == "collection" {
					c.collectionFamilies[name] = true
				}
			}
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
			c.EntriesDirect[ei.Dir]++
			for _, d := range ancestorDirs(f.Path) {
				c.EntriesBeneath[d]++
			}
			c.RelatedLinks[f.Path] = resolveRelated(f, v)
		}
	}
	for _, d := range v.Dirs {
		p := parentDir(d)
		if p == "" {
			continue
		}
		c.childDirs[p] = append(c.childDirs[p], lastSegment(d))
	}
	for _, children := range c.childDirs {
		sort.Strings(children)
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

// folderNoteKind classifies a folder-note dir for ADR 0004 rule 19 and the R3
// fix: "entity" (children are event-type dirs), "collection" (children are
// year dirs), or "" when the dir is not linted (scope roots, structural dirs,
// company dirs and unknown depth).
//
// Scoping (documented in the README): a dir is linted when it is itself a
// taxonomy category (kind entity or collection), or an entity dir (a direct
// child of an entity-kind category, at personal/{cat}/{entity} or
// work/{company}/{cat}/{entity}).
func (c *Context) folderNoteKind(dir string) string {
	if dir == "" || c.Tax == nil || c.isStructuralPath(dir) {
		return ""
	}
	segs := splitPath(dir)
	switch len(segs) {
	case 2:
		if scope, ok := c.Tax.Scopes[segs[0]]; ok {
			if cat, ok := scope.Categories[segs[1]]; ok {
				return cat.Kind
			}
		}
		return ""
	case 3:
		if segs[0] == "work" {
			if scope, ok := c.Tax.Scopes["work"]; ok {
				if segs[1] == "_shared" || contains(scope.Companies, segs[1]) {
					if cat, ok := scope.Categories[segs[2]]; ok {
						return cat.Kind
					}
				}
			}
			return ""
		}
		if scope, ok := c.Tax.Scopes[segs[0]]; ok {
			if cat, ok := scope.Categories[segs[1]]; ok && cat.Kind == "entity" {
				return "entity"
			}
		}
		return ""
	case 4:
		if segs[0] != "work" {
			return ""
		}
		scope, ok := c.Tax.Scopes["work"]
		if !ok {
			return ""
		}
		if segs[1] != "_shared" && !contains(scope.Companies, segs[1]) {
			return ""
		}
		if cat, ok := scope.Categories[segs[2]]; ok && cat.Kind == "entity" {
			return "entity"
		}
		return ""
	}
	return ""
}

// actualChildren computes the ADR 0004 rule 19 "actual" children_counts map for
// a linted folder-note dir. Entity dirs key on the event-type dirs directly
// under them (count = entries anywhere beneath, all years); collection dirs key
// on the 4-digit year dirs directly under them (count = entries directly in the
// year dir). Only existing child dirs are keys, so an empty type/year dir is
// stored as 0. The second result is false when the dir is not linted.
func (c *Context) actualChildren(dir string) (map[string]int, bool) {
	kind := c.folderNoteKind(dir)
	if kind == "" {
		return nil, false
	}
	out := map[string]int{}
	for _, child := range c.childDirs[dir] {
		switch kind {
		case "entity":
			if !c.typeFamilies[child] {
				continue
			}
			out[child] = c.EntriesBeneath[dir+"/"+child]
		case "collection":
			if !yearDirRe.MatchString(child) {
				continue
			}
			out[child] = c.EntriesDirect[dir+"/"+child]
		}
	}
	return out, true
}

// missingFolderNoteDirs returns the entity/taxonomy dirs that have no folder
// note, in sorted order. Shared by rule 3 and its --fix path.
func (c *Context) missingFolderNoteDirs() []string {
	seen := map[string]bool{}
	var out []string
	for _, ei := range c.Entries {
		for _, d := range ancestorDirs(ei.File.Path) {
			if yearDirRe.MatchString(lastSegment(d)) {
				continue
			}
			if c.isStructuralPath(d) {
				continue
			}
			if c.typeFamilies[lastSegment(d)] {
				continue
			}
			if _, ok := c.FolderNotes[d]; ok {
				continue
			}
			if seen[d] {
				continue
			}
			seen[d] = true
			out = append(out, d)
		}
	}
	sort.Strings(out)
	return out
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

// fmIntMap decodes a frontmatter mapping into an int map. YAML numbers may
// arrive as int, int64 or float64; strings holding bare integers are accepted
// too. Numeric-looking keys (unquoted YAML years) force map[any]any, whose keys
// are stringified back. The bool result is false when the field is
// absent/nil or not a mapping.
func fmIntMap(fm map[string]any, key string) (map[string]int, bool) {
	v, ok := fm[key]
	if !ok || v == nil {
		return nil, false
	}
	out := make(map[string]int)
	switch m := v.(type) {
	case map[string]any:
		for k, e := range m {
			if n, ok := intValue(e); ok {
				out[k] = n
			}
		}
	case map[any]any:
		for k, e := range m {
			if n, ok := intValue(e); ok {
				out[fmt.Sprint(k)] = n
			}
		}
	default:
		return nil, false
	}
	return out, true
}

func intValue(e any) (int, bool) {
	switch n := e.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
			return i, true
		}
	}
	return 0, false
}

// bodyOf returns the markdown body after a leading YAML frontmatter block, or
// the whole content when there is no frontmatter.
func bodyOf(content string) string {
	lines := strings.SplitAfter(content, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r\n") != "---" {
		return content
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r\n") == "---" {
			return strings.Join(lines[i+1:], "")
		}
	}
	return content
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
