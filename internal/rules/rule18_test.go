package rules

import (
	"reflect"
	"testing"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

func TestRule18(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  []vault.Finding
	}{
		{
			name: "both present",
			files: map[string]string{
				"todos.md":        "x\n",
				"inbox/review.md": "x\n",
			},
			want: nil,
		},
		{
			name: "both missing",
			files: map[string]string{
				"README.md": "x\n",
			},
			want: []vault.Finding{
				{Rule: 18, Severity: vault.SeverityError, Path: "inbox/review.md", Message: "inbox/review.md must exist"},
				{Rule: 18, Severity: vault.SeverityError, Path: "todos.md", Message: "todos.md must exist at the vault root"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := byRule(lintVault(t, tc.files, emptyTax()), 18)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("R18 findings mismatch:\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}
