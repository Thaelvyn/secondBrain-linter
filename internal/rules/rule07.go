package rules

import (
	"fmt"
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule07: folder-note frontmatter is valid: type in {entity, collection},
// name, scope, status in {active, dormant}, created present.
func rule07(c *Context) {
	for _, dir := range sortedKeys(c.FolderNotes) {
		f := c.FolderNotes[dir]
		fm := f.Frontmatter
		var issues []string
		if !f.HasFM {
			issues = append(issues, "no frontmatter")
		} else {
			if t, ok := fmString(fm, "type"); ok && t != "entity" && t != "collection" {
				issues = append(issues, fmt.Sprintf("type %q not in {entity, collection}", t))
			}
			for _, k := range []string{"name", "scope", "created"} {
				if _, ok := fm[k]; !ok {
					issues = append(issues, fmt.Sprintf("missing field %q", k))
				}
			}
			if s, ok := fmString(fm, "status"); ok && s != "active" && s != "dormant" {
				issues = append(issues, fmt.Sprintf("status %q not in {active, dormant}", s))
			}
		}
		if len(issues) > 0 {
			c.Add(7, vault.SeverityError, f.Path, "invalid folder-note frontmatter: "+strings.Join(issues, "; "))
		}
	}
}
