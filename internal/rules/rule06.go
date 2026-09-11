package rules

import (
	"fmt"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule06: type is in the event_types catalog; privacy in
// {public, private, secret}; origin in {generated, human}; perspective/date/
// summary non-empty; date is a valid YYYY-MM-DD.
func rule06(c *Context) {
	for _, ei := range c.Entries {
		fm := ei.File.Frontmatter
		if t, ok := fmString(fm, "type"); ok {
			if _, in := c.Tax.EventTypes[t]; !in {
				c.Add(6, vault.SeverityError, ei.File.Path, fmt.Sprintf("type %q is not in the event_types catalog of _system/taxonomy.yaml", t))
			}
		}
		if p, ok := fmString(fm, "privacy"); ok && p != "public" && p != "private" && p != "secret" {
			c.Add(6, vault.SeverityError, ei.File.Path, fmt.Sprintf("privacy %q not in {public, private, secret}", p))
		}
		if o, ok := fmString(fm, "origin"); ok && o != "generated" && o != "human" {
			c.Add(6, vault.SeverityError, ei.File.Path, fmt.Sprintf("origin %q not in {generated, human}", o))
		}
		for _, k := range []string{"perspective", "summary"} {
			if s, ok := fmString(fm, k); ok && s == "" {
				c.Add(6, vault.SeverityError, ei.File.Path, fmt.Sprintf("frontmatter field %q must be non-empty", k))
			}
		}
		if d, ok := fmString(fm, "date"); ok && !validDate(d) {
			c.Add(6, vault.SeverityError, ei.File.Path, fmt.Sprintf("date %q is not a valid YYYY-MM-DD", d))
		}
	}
}
