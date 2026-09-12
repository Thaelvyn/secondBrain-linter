package rules

import (
	"regexp"
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// relatedHeadingRe matches the optional `## Related` body heading; a trailing
// inline comment (e.g. `<!-- optional -->`) is tolerated on the heading text.
var relatedHeadingRe = regexp.MustCompile(`^##\s+Related(?:\s|$)`)

// h2StartRe matches the start of any H2 heading, terminating a body section.
var h2StartRe = regexp.MustCompile(`^##\s`)

// listItemRe matches a markdown list item, capturing its text.
var listItemRe = regexp.MustCompile(`^\s*[-*]\s+(.*)$`)

// rule11: related (entries and folder notes) contains only wikilinks;
// entities and source contain only plain paths (no brackets); the optional
// body `## Related` section contains only wikilinks.
func rule11(c *Context) {
	for _, ei := range c.Entries {
		checkRelated(c, ei.File.Frontmatter, ei.File.Path)
		for _, item := range stringList(ei.File.Frontmatter, "entities") {
			if strings.ContainsAny(item, "[]") {
				c.Add(11, vault.SeverityError, ei.File.Path, "frontmatter entities must contain only plain paths, no brackets")
				break
			}
		}
		for _, item := range stringList(ei.File.Frontmatter, "source") {
			if strings.ContainsAny(item, "[]") {
				c.Add(11, vault.SeverityError, ei.File.Path, "frontmatter source must contain only plain paths, no brackets")
				break
			}
		}
		for _, item := range relatedBodyItems(ei.File.Content) {
			if !isWikiLink(item) {
				c.Add(11, vault.SeverityError, ei.File.Path, "body Related section must contain only wikilinks ([[full/path]])")
				break
			}
		}
	}
	for _, dir := range sortedKeys(c.FolderNotes) {
		f := c.FolderNotes[dir]
		checkRelated(c, f.Frontmatter, f.Path)
	}
}

// relatedBodyItems returns the trimmed text of every top-level list item in the
// optional `## Related` body section, or nil when the heading is absent. The
// section runs from the heading line to the next H2 heading or EOF. The heading
// line itself and non-list lines (blank lines, prose) are ignored.
func relatedBodyItems(content string) []string {
	lines := strings.Split(bodyOf(content), "\n")
	start := -1
	for i, line := range lines {
		if relatedHeadingRe.MatchString(strings.TrimSpace(line)) {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return nil
	}
	var out []string
	for _, line := range lines[start:] {
		if h2StartRe.MatchString(line) {
			break
		}
		if m := listItemRe.FindStringSubmatch(line); m != nil {
			out = append(out, strings.TrimSpace(m[1]))
		}
	}
	return out
}

func checkRelated(c *Context, fm map[string]any, path string) {
	for _, item := range stringList(fm, "related") {
		if !isWikiLink(item) {
			c.Add(11, vault.SeverityError, path, "frontmatter related must contain only wikilinks ([[full/path]])")
			return
		}
	}
}
