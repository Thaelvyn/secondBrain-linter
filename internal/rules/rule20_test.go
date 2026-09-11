package rules

import (
	"reflect"
	"testing"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

func TestRule20(t *testing.T) {
	entry := func(source string) string {
		return "---\nevent_id: 2026-01-01-abcdef123456\ndate: 2026-01-01\ntype: meeting\nperspective: personal\nprivacy: private\nentities:\n  - personal/people/jane-doe\norigin: generated\nsource: " + source + "\nsummary: x\n---\n"
	}
	cases := []struct {
		name  string
		files map[string]string
		want  []vault.Finding
	}{
		{
			name: "missing source file is an error",
			files: map[string]string{
				"todos.md":        "x\n",
				"inbox/review.md": "x\n",
				"personal/people/jane-doe/meeting/2026/2026-01-01-x.md": entry("daily-logs/2026/01/2026-01-01-x.md"),
			},
			want: []vault.Finding{
				{Rule: 20, Severity: vault.SeverityError, Path: "personal/people/jane-doe/meeting/2026/2026-01-01-x.md", Message: "source file does not exist: daily-logs/2026/01/2026-01-01-x.md"},
			},
		},
		{
			name: "manual, empty and existing sources are skipped",
			files: map[string]string{
				"todos.md":                           "x\n",
				"inbox/review.md":                    "x\n",
				"daily-logs/2026/01/2026-01-01-x.md": "x\n",
				"personal/people/jane-doe/meeting/2026/2026-01-01-a.md": entry("manual"),
				"personal/people/jane-doe/meeting/2026/2026-01-01-b.md": entry("./daily-logs/2026/01/2026-01-01-x.md"),
				"personal/people/jane-doe/meeting/2026/2026-01-01-c.md": entry("daily-logs/2026/01/2026-01-01-x"),
				"personal/people/jane-doe/meeting/2026/2026-01-01-d.md": entry("Manual"),
				"personal/people/jane-doe/meeting/2026/2026-01-01-e.md": "---\nevent_id: 2026-01-01-abcdef123456\ndate: 2026-01-01\ntype: meeting\nperspective: personal\nprivacy: private\nentities:\n  - personal/people/jane-doe\norigin: generated\nsummary: x\n---\n",
			},
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tax := scopedTax("personal", nil, map[string]string{"people": "entity"}, []string{"meeting"})
			got := byRule(lintVault(t, tc.files, tax), 20)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("R20 findings mismatch:\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}
