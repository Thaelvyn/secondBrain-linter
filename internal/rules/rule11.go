package rules

import (
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule11: related (entries and folder notes) contains only wikilinks;
// entities and source contain only plain paths (no brackets).
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
	}
	for _, dir := range sortedKeys(c.FolderNotes) {
		f := c.FolderNotes[dir]
		checkRelated(c, f.Frontmatter, f.Path)
	}
}

func checkRelated(c *Context, fm map[string]any, path string) {
	for _, item := range stringList(fm, "related") {
		if !isWikiLink(item) {
			c.Add(11, vault.SeverityError, path, "frontmatter related must contain only wikilinks ([[full/path]])")
			return
		}
	}
}
