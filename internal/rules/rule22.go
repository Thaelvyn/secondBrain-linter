package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// headingRe matches an H2 heading line, capturing its text.
var headingRe = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)

// optionalMarker marks a template heading as optional (ADR 0006).
const optionalMarker = "<!-- optional -->"

// rule22: each entry contains the required H2 headings of its type, resolved at
// runtime from the vault template (ADR 0004 rule 22 / ADR 0006, error). The
// template is the contract: unmarked `## Heading` lines are required, lines
// carrying `<!-- optional -->` are not. A template marked heading whose section
// holds only `n/a` is accepted because the heading itself is present. A missing
// template yields one warning per type and skips that type.
func rule22(c *Context) {
	assessed := map[string]bool{}
	required := map[string][]string{}
	for _, ei := range c.Entries {
		typ, _ := fmString(ei.File.Frontmatter, "type")
		if typ == "" {
			continue
		}
		if !assessed[typ] {
			assessed[typ] = true
			tplPath := c.templatePath(typ)
			data, err := os.ReadFile(filepath.Join(c.Vault.Root, filepath.FromSlash(tplPath)))
			if err != nil {
				c.Add(22, vault.SeverityWarning, tplPath,
					fmt.Sprintf("template missing for type %s: cannot validate body headings", typ))
				required[typ] = nil
			} else {
				req, _ := parseTemplateHeadings(string(data))
				required[typ] = req
			}
		}
		present := entryHeadings(ei.File.Content)
		for _, h := range required[typ] {
			if !present[h] {
				c.Add(22, vault.SeverityError, ei.File.Path,
					fmt.Sprintf("missing required heading %q for type %q", "## "+h, typ))
			}
		}
	}
}

// templatePath resolves the body template path for a type, preferring the
// taxonomy registration (ADR 0002) and falling back to the conventional
// `_system/templates/{type}.md`.
func (c *Context) templatePath(typ string) string {
	if c.Tax != nil {
		if et, ok := c.Tax.EventTypes[typ]; ok && et.Template != "" {
			return et.Template
		}
	}
	return "_system/templates/" + typ + ".md"
}

// parseTemplateHeadings returns the required and optional H2 headings of a
// template body.
func parseTemplateHeadings(content string) (required, optional []string) {
	for _, m := range headingRe.FindAllStringSubmatch(content, -1) {
		h := strings.TrimSpace(m[1])
		if strings.Contains(h, optionalMarker) {
			h = strings.TrimSpace(strings.Replace(h, optionalMarker, "", 1))
			if h != "" {
				optional = append(optional, h)
			}
			continue
		}
		if h != "" {
			required = append(required, h)
		}
	}
	return required, optional
}

// entryHeadings returns the set of trimmed H2 heading texts in an entry body.
func entryHeadings(content string) map[string]bool {
	out := map[string]bool{}
	for _, m := range headingRe.FindAllStringSubmatch(bodyOf(content), -1) {
		out[strings.TrimSpace(m[1])] = true
	}
	return out
}
