package rules

import (
	"fmt"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule03: every entity folder (a dir whose subpaths contain entries) contains
// its {folder}/{folder}.md folder note. Exempt: year dirs, dirs under the
// structural top-levels, and event-type family dirs ({type} / {type}s).
func rule03(c *Context) {
	reported := map[string]bool{}
	for _, ei := range c.Entries {
		for _, d := range ancestorDirs(ei.File.Path) {
			base := lastSegment(d)
			if yearDirRe.MatchString(base) {
				continue
			}
			if c.isStructuralPath(d) {
				continue
			}
			if c.typeFamilies[base] {
				continue
			}
			if _, ok := c.FolderNotes[d]; ok {
				continue
			}
			if reported[d] {
				continue
			}
			reported[d] = true
			c.Add(3, vault.SeverityError, d, fmt.Sprintf("entity folder %q is missing its folder note %q", d, d+".md"))
		}
	}
}
