package rules

import (
	"fmt"
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule23: every entity named in an entry's `entities` frontmatter that has a
// folder note must be inline-linked (`[[...]]`) from the entry body, using the
// folder-note FILE path (`{entity}/{basename}`), since Obsidian wikilinks
// resolve to files only. Links may sit anywhere in the body, including the
// optional `## Related` section; the frontmatter `related` field does not
// count. The severity follows the folder note's kind: a missing link to a
// `type: entity` note is an error, to a `type: collection` note a warning
// (missing/unknown kind counts as entity). Daily-log and `_system` files are
// never linted as entries.
func rule23(c *Context) {
	for _, ei := range c.Entries {
		if isDailyLogOrSystemPath(ei.File.Path) {
			continue
		}
		targets := bodyWikiTargets(bodyOf(ei.File.Content), c.Vault)
		for _, item := range stringList(ei.File.Frontmatter, "entities") {
			entity := strings.TrimSpace(item)
			if entity == "" || strings.ContainsAny(entity, "[]") {
				continue
			}
			note := c.folderNoteFor(entity)
			if note == nil {
				continue
			}
			notePath := strings.TrimSuffix(note.Path, ".md")
			if targets[notePath] || targets[notePath+".md"] || targets[resolveWiki(notePath, c.Vault)] {
				continue
			}
			sev := vault.SeverityError
			if t, _ := fmString(note.Frontmatter, "type"); t == "collection" {
				sev = vault.SeverityWarning
			}
			c.Add(23, sev, ei.File.Path,
				fmt.Sprintf("entry body is missing a wikilink to entity %q", entity))
		}
	}
}

// isDailyLogOrSystemPath reports whether a path lives under the daily-logs
// (including the daily-logs/in drop zone) or _system top-levels, which are
// never subject to the entry-body rules.
func isDailyLogOrSystemPath(path string) bool {
	return strings.HasPrefix(path, "daily-logs/") || strings.HasPrefix(path, "_system/")
}

// folderNoteFor returns the folder note backing an entities path: the
// `{dir}/{basename}.md` note when the path is a directory, or a plain
// `{path}.md` file at that path. Returns nil when neither exists.
func (c *Context) folderNoteFor(entity string) *vault.File {
	if f, ok := c.FolderNotes[entity]; ok {
		return f
	}
	return c.Vault.ByPath[entity+".md"]
}

// bodyWikiTargets collects every inline wikilink in a body, keyed by its
// stripped raw target (alias `|` and anchor `#` removed, trailing "/" trimmed)
// and by its resolved canonical path. Embeds (![[...]]) are skipped, as in
// rule 10.
func bodyWikiTargets(body string, v *vault.Vault) map[string]bool {
	out := map[string]bool{}
	for _, m := range wikiRe.FindAllStringSubmatchIndex(body, -1) {
		start := m[0]
		if start > 0 && body[start-1] == '!' {
			continue
		}
		t := wikiTarget(body[m[2]:m[3]])
		t = strings.TrimSuffix(t, "/")
		if t == "" {
			continue
		}
		out[t] = true
		if r := resolveWiki(t, v); r != "" {
			out[r] = true
		}
	}
	return out
}
