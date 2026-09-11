package rules

import (
	"fmt"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule10: all wikilinks [[...]] resolve to existing files. Aliases (|...)
// and anchors (#...) are stripped; targets are tried as-is and with ".md".
// Embeds (![[...]]) are skipped here and reported by rule 12.
func rule10(c *Context) {
	for _, f := range c.Vault.Files {
		if !f.Markdown {
			continue
		}
		for _, m := range wikiRe.FindAllStringSubmatchIndex(f.Content, -1) {
			start := m[0]
			if start > 0 && f.Content[start-1] == '!' {
				continue
			}
			raw := f.Content[m[2]:m[3]]
			target := raw
			if i := indexByteSlash(target, '|'); i >= 0 {
				target = target[:i]
			}
			if resolveWiki(target, c.Vault) == "" {
				c.Add(10, vault.SeverityError, f.Path, fmt.Sprintf("wikilink [[%s]] does not resolve to an existing file", raw))
			}
		}
	}
}

func indexByteSlash(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
