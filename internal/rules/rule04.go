package rules

import (
	"fmt"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule04: category folder names exist in _system/taxonomy.yaml
// (scopes -> categories). Work company dirs must be in the scope's companies
// list; _shared is allowed. Structural top-levels (_system, daily-logs,
// calendar, inbox, attachments) are exempt. Root files todos.md/README.md and
// scope roots themselves are not folders and never reach this rule.
func rule04(c *Context) {
	for _, d := range c.Vault.Dirs {
		segs := splitPath(d)
		switch len(segs) {
		case 1:
			if structuralSet[segs[0]] {
				continue
			}
			if _, ok := c.Tax.Scopes[segs[0]]; ok {
				continue
			}
			c.Add(4, vault.SeverityError, d, fmt.Sprintf("top-level folder %q is not a taxonomy scope or a structural folder", segs[0]))
		case 2:
			scope, ok := c.Tax.Scopes[segs[0]]
			if !ok {
				continue // top-level already flagged
			}
			if segs[0] == "work" {
				if segs[1] == "_shared" || contains(scope.Companies, segs[1]) {
					continue
				}
				c.Add(4, vault.SeverityError, d, fmt.Sprintf("company folder %q is not in the companies list of _system/taxonomy.yaml", segs[1]))
				continue
			}
			if _, ok := scope.Categories[segs[1]]; ok {
				continue
			}
			c.Add(4, vault.SeverityError, d, fmt.Sprintf("category folder %q is not in taxonomy scopes.%s.categories", segs[1], segs[0]))
		case 3:
			if segs[0] != "work" {
				continue
			}
			scope := c.Tax.Scopes[segs[0]]
			if segs[1] == "_shared" || !contains(scope.Companies, segs[1]) {
				continue // exempt or already flagged at depth 2
			}
			if _, ok := scope.Categories[segs[2]]; ok {
				continue
			}
			c.Add(4, vault.SeverityError, d, fmt.Sprintf("category folder %q is not in taxonomy scopes.work.categories", segs[2]))
		}
	}
}

func splitPath(d string) []string {
	out := []string{}
	start := 0
	for i := 0; i <= len(d); i++ {
		if i == len(d) || d[i] == '/' {
			out = append(out, d[start:i])
			start = i + 1
		}
	}
	return out
}
