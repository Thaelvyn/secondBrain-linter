package rules

import (
	"os"
	"path/filepath"

	"github.com/Thaelvyn/secondBrain-linter/internal/taxonomy"
)

func emptyTax() *taxonomy.Taxonomy {
	return &taxonomy.Taxonomy{
		Scopes: map[string]taxonomy.Scope{
			"personal": {Categories: map[string]taxonomy.Category{}},
			"work":     {Categories: map[string]taxonomy.Category{}},
			"goals":    {Categories: map[string]taxonomy.Category{}},
			"reference": {Categories: map[string]taxonomy.Category{}},
		},
		EventTypes: map[string]taxonomy.EventType{},
	}
}

func mkdirAll(p string) error {
	dir := filepath.Dir(p)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func writeFile(p, content string) error {
	return os.WriteFile(p, []byte(content), 0o644)
}
