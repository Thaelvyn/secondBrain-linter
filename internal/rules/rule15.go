package rules

import (
	"fmt"
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule15: path depth (file path segments relative to the vault root) must
// not exceed 8. Soft rule, warning only. Folder notes count all segments
// like any other file.
func rule15(c *Context) {
	for _, f := range c.Vault.Files {
		n := len(strings.Split(f.Path, "/"))
		if n > 8 {
			c.Add(15, vault.SeverityWarning, f.Path, fmt.Sprintf("path depth %d exceeds the 8-segment soft cap", n))
		}
	}
}
