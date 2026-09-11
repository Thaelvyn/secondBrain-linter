package rules

import (
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule18: todos.md and inbox/review.md exist.
func rule18(c *Context) {
	if _, ok := c.Vault.ByPath["todos.md"]; !ok {
		c.Add(18, vault.SeverityError, "todos.md", "todos.md must exist at the vault root")
	}
	if _, ok := c.Vault.ByPath["inbox/review.md"]; !ok {
		c.Add(18, vault.SeverityError, "inbox/review.md", "inbox/review.md must exist")
	}
}

// topLevelDir returns the first path segment, or "" for a root-level file.
func topLevelDir(p string) string {
	i := strings.IndexByte(p, '/')
	if i < 0 {
		return ""
	}
	return p[:i]
}
