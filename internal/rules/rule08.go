package rules

import (
	"fmt"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule08: event_id matches {YYYY-MM-DD}-{12-hex}.
func rule08(c *Context) {
	for _, ei := range c.Entries {
		eid, _ := fmString(ei.File.Frontmatter, "event_id")
		if !eventIDRe.MatchString(eid) {
			c.Add(8, vault.SeverityError, ei.File.Path, fmt.Sprintf("event_id %q does not match {YYYY-MM-DD}-{12-hex}", eid))
		}
	}
}
