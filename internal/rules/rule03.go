package rules

import (
	"fmt"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule03: every entity folder (a dir whose subpaths contain entries) contains
// its {folder}/{folder}.md folder note. Exempt: year dirs, dirs under the
// structural top-levels, and event-type family dirs ({type} / {type}s).
func rule03(c *Context) {
	for _, d := range c.missingFolderNoteDirs() {
		c.Add(3, vault.SeverityError, d, fmt.Sprintf("entity folder %q is missing its folder note %q", d, d+".md"))
	}
}
