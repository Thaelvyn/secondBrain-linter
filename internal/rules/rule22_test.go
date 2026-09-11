package rules

import (
	"reflect"
	"testing"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

func TestRule22(t *testing.T) {
	entry := func(eid, body string) string {
		return "---\nevent_id: " + eid + "\ndate: 2026-01-01\ntype: meeting\nperspective: personal\nprivacy: private\nentities:\n  - personal/people/jane-doe\norigin: generated\nsource: manual\nsummary: x\n---\n" + body
	}
	templates := map[string]string{
		"meeting": "---\ntype: meeting\n---\n\n## Context\n\n## Discussion\n\n## Related <!-- optional -->\n",
	}
	cases := []struct {
		name  string
		files map[string]string
		want  []vault.Finding
	}{
		{
			name: "missing required headings are errors",
			files: map[string]string{
				"_system/templates/meeting.md": templates["meeting"],
				"todos.md":                     "x\n",
				"inbox/review.md":              "x\n",
				"personal/people/jane-doe/meeting/2026/2026-01-01-x.md": entry("2026-01-01-abcdef123456", "\nLede.\n\n## Context\n"),
			},
			want: []vault.Finding{
				{Rule: 22, Severity: vault.SeverityError, Path: "personal/people/jane-doe/meeting/2026/2026-01-01-x.md", Message: "missing required heading \"## Discussion\" for type \"meeting\""},
			},
		},
		{
			name: "optional heading and n/a content are accepted",
			files: map[string]string{
				"_system/templates/meeting.md": templates["meeting"],
				"todos.md":                     "x\n",
				"inbox/review.md":              "x\n",
				"personal/people/jane-doe/meeting/2026/2026-01-01-x.md": entry("2026-01-01-abcdef123456", "\nLede.\n\n## Context\n\nn/a\n\n## Discussion\n\nn/a\n"),
			},
			want: nil,
		},
		{
			name: "missing template warns once per type and skips validation",
			files: map[string]string{
				"todos.md":        "x\n",
				"inbox/review.md": "x\n",
				"personal/people/jane-doe/meeting/2026/2026-01-01-a.md": entry("2026-01-01-abcdef123456", ""),
				"personal/people/jane-doe/meeting/2026/2026-01-01-b.md": entry("2026-01-01-abcdef123457", ""),
			},
			want: []vault.Finding{
				{Rule: 22, Severity: vault.SeverityWarning, Path: "_system/templates/meeting.md", Message: "template missing for type meeting: cannot validate body headings"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tax := scopedTax("personal", nil, map[string]string{"people": "entity"}, []string{"meeting"})
			got := byRule(lintVault(t, tc.files, tax), 22)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("R22 findings mismatch:\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}
