package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

var (
	// todoRowRe matches an sb-todos rolled-up row:
	//   - [ ] <text> — [[<full entry path>]] · <event_id>
	todoRowRe = regexp.MustCompile(`^-\s*\[[ xX]\]\s*(.*?)\s*—\s*\[\[([^\]]+)\]\](?:\s*·\s*(\S+))?\s*$`)
	// inlineTaskRe matches an inline markdown task item, capturing the checkbox
	// state and the task text.
	inlineTaskRe = regexp.MustCompile(`(?m)^[ \t]*-[ \t]+\[([ xX])\][ \t]*(.*?)[ \t]*$`)
)

// rule21: todos.md open rows and open inline entry tasks reconcile
// bidirectionally (ADR 0004 rule 21, warning). Dedup key is (event_id,
// normalized task text). Only open inline tasks must have a row; a row must
// resolve to an existing entry whose event_id matches the row.
func rule21(c *Context) {
	todos, ok := c.Vault.ByPath["todos.md"]
	if !ok {
		return // absence is rule 18's job; do not double-report
	}
	type taskKey struct{ eventID, text string }
	matched := map[taskKey]bool{}
	for _, line := range strings.Split(bodyOf(todos.Content), "\n") {
		m := todoRowRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		text := normalizeTaskText(m[1])
		target, rowEID := m[2], m[3]
		resolved := resolveWiki(target, c.Vault)
		if resolved == "" {
			c.Add(21, vault.SeverityWarning, todos.Path,
				fmt.Sprintf("todos row references a file that does not exist: [[%s]]", target))
			continue
		}
		fileEID, _ := fmString(c.Vault.ByPath[resolved].Frontmatter, "event_id")
		if rowEID == "" || fileEID == "" || rowEID != fileEID {
			c.Add(21, vault.SeverityWarning, todos.Path,
				fmt.Sprintf("todos row event_id %q does not match %s", rowEID, resolved))
			continue
		}
		matched[taskKey{rowEID, text}] = true
	}
	for _, ei := range c.Entries {
		eid, _ := fmString(ei.File.Frontmatter, "event_id")
		for _, m := range inlineTaskRe.FindAllStringSubmatch(bodyOf(ei.File.Content), -1) {
			if m[1] != " " {
				continue // closed inline tasks do not need a row
			}
			text := normalizeTaskText(m[2])
			if !matched[taskKey{eid, text}] {
				c.Add(21, vault.SeverityWarning, ei.File.Path,
					fmt.Sprintf("open inline task %q has no matching todos.md row", text))
			}
		}
	}
}

// normalizeTaskText trims and collapses whitespace for dedup comparison.
func normalizeTaskText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
