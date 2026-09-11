package rules

import (
	"fmt"
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule16: entry leaves live under {entity}/{type}/{year}/: the parent folder
// must be a 4-digit year dir, the folder above it must be one of the event
// types from the taxonomy catalog, and the filename must start with the
// frontmatter date (YYYY-MM-DD prefix match).
func rule16(c *Context) {
	for _, ei := range c.Entries {
		parent := lastSegment(ei.Dir)
		if !yearDirRe.MatchString(parent) {
			c.Add(16, vault.SeverityError, ei.File.Path, fmt.Sprintf("entry parent folder must be a 4-digit year folder (got %q)", parent))
		} else {
			gp := lastSegment(parentDir(ei.Dir))
			if gp != "" && !c.typeFamilies[gp] {
				c.Add(16, vault.SeverityError, ei.File.Path, fmt.Sprintf("folder above the year must be an event type from the taxonomy catalog (got %q)", gp))
			}
		}
		date, ok := fmString(ei.File.Frontmatter, "date")
		if ok && date != "" && !strings.HasPrefix(ei.File.Name, date) {
			c.Add(16, vault.SeverityError, ei.File.Path, fmt.Sprintf("filename must start with the frontmatter date %s", date))
		}
	}
}
