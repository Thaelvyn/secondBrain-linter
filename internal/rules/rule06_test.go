package rules

import (
	"reflect"
	"testing"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

func TestRule06TypeEnums(t *testing.T) {
	entry := func(fields map[string]string) string {
		b := "---\nevent_id: 2026-01-01-abcdef123456\ndate: 2026-01-01\nperspective: personal\nprivacy: private\nentities:\n  - personal/people/jane-doe\norigin: generated\nsource: manual\nsummary: x\n"
		for k, v := range fields {
			b += k + ": " + v + "\n"
		}
		return b + "---\n"
	}
	cases := []struct {
		name  string
		files map[string]string
		want  []vault.Finding
	}{
		{
			name: "decision status enum",
			files: map[string]string{
				"personal/people/jane-doe/decisions/2026/2026-01-01-x.md": entry(map[string]string{"type": "decision", "status": "bogus"}),
			},
			want: []vault.Finding{
				{Rule: 6, Severity: vault.SeverityError, Path: "personal/people/jane-doe/decisions/2026/2026-01-01-x.md", Message: "decision status \"bogus\" not in {proposed, accepted, superseded}"},
			},
		},
		{
			name: "decision supersedes/superseded_by must be wikilinks",
			files: map[string]string{
				"personal/people/jane-doe/decisions/2026/2026-01-01-x.md": entry(map[string]string{"type": "decision", "supersedes": "personal/people/jane-doe/decisions/2026/2026-01-01-y.md"}),
			},
			want: []vault.Finding{
				{Rule: 6, Severity: vault.SeverityError, Path: "personal/people/jane-doe/decisions/2026/2026-01-01-x.md", Message: "decision frontmatter supersedes must be a wikilink ([[full/path]])"},
			},
		},
		{
			name: "fact and observation confidence enums",
			files: map[string]string{
				"personal/people/jane-doe/facts/2026/2026-01-01-a.md":        entry(map[string]string{"type": "fact", "confidence": "medium"}),
				"personal/people/jane-doe/observations/2026/2026-01-01-b.md": entry(map[string]string{"type": "observation", "confidence": "sure"}),
				"personal/people/jane-doe/decisions/2026/2026-01-01-c.md":    entry(map[string]string{"type": "decision", "status": "accepted", "superseded_by": "[[personal/people/jane-doe/decisions/2026/2026-01-01-d]]"}),
			},
			want: []vault.Finding{
				{Rule: 6, Severity: vault.SeverityError, Path: "personal/people/jane-doe/observations/2026/2026-01-01-b.md", Message: "observation confidence \"sure\" not in {high, medium, low}"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tax := scopedTax("personal", nil, map[string]string{"people": "entity"}, []string{"decision", "fact", "observation"})
			got := byRule(lintVault(t, tc.files, tax), 6)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("R6 findings mismatch:\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}
