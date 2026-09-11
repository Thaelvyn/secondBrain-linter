package rules

import (
	"fmt"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule06: type is in the event_types catalog; privacy in
// {public, private, secret}; origin in {generated, human}; perspective/date/
// summary non-empty; date is a valid YYYY-MM-DD. ADR 0006 adds optional enums:
// decision status, fact/observation confidence, and decision supersedes/
// superseded_by wikilinks (validated only when present).
func rule06(c *Context) {
	for _, ei := range c.Entries {
		fm := ei.File.Frontmatter
		if t, ok := fmString(fm, "type"); ok {
			if _, in := c.Tax.EventTypes[t]; !in {
				c.Add(6, vault.SeverityError, ei.File.Path, fmt.Sprintf("type %q is not in the event_types catalog of _system/taxonomy.yaml", t))
			}
			checkTypeEnums(c, ei.File.Path, t, fm)
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

// checkTypeEnums validates the ADR 0006 optional type-specific enums.
func checkTypeEnums(c *Context, path, typ string, fm map[string]any) {
	switch typ {
	case "decision":
		if s, ok := fmString(fm, "status"); ok && s != "proposed" && s != "accepted" && s != "superseded" {
			c.Add(6, vault.SeverityError, path, fmt.Sprintf("decision status %q not in {proposed, accepted, superseded}", s))
		}
		for _, k := range []string{"supersedes", "superseded_by"} {
			for _, item := range stringList(fm, k) {
				if item != "" && !isWikiLink(item) {
					c.Add(6, vault.SeverityError, path, fmt.Sprintf("decision frontmatter %s must be a wikilink ([[full/path]])", k))
					break
				}
			}
		}
	case "fact", "observation":
		if cv, ok := fmString(fm, "confidence"); ok && cv != "high" && cv != "medium" && cv != "low" {
			c.Add(6, vault.SeverityError, path, fmt.Sprintf("%s confidence %q not in {high, medium, low}", typ, cv))
		}
	}
}
