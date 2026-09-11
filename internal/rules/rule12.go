package rules

import (
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule12: no embeds anywhere: "![[" is forbidden in markdown files.
func rule12(c *Context) {
	for _, f := range c.Vault.Files {
		if !f.Markdown {
			continue
		}
		if strings.Contains(f.Content, "![[") {
			c.Add(12, vault.SeverityError, f.Path, "embeds are forbidden: found ![[...]]")
		}
	}
}
