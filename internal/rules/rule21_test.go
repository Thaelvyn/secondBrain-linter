package rules

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

const kickoffBody = `---
event_id: 2026-01-01-abcdef123456
date: 2026-01-01
type: meeting
perspective: personal
privacy: private
entities:
  - personal/people/jane-doe
origin: generated
source: manual
summary: Kickoff
---

Body.

## Actions

- [ ] Send the deck
- [x] Already done
`

func TestRule21(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  []vault.Finding
	}{
		{
			name: "open task and broken row are both drift",
			files: map[string]string{
				"todos.md":                             "# Todos\n\n- [ ] Send the deck — [[personal/people/jane-doe/meeting/2026/2026-01-01-kickoff.md]] · 2026-01-01-abcdef123456\n- [ ] Ghost — [[personal/people/jane-doe/meeting/2026/2026-01-01-nope]] · 2026-01-01-000000000000\n- [ ] Mismatch — [[personal/people/jane-doe/meeting/2026/2026-01-01-kickoff.md]] · 2026-01-01-badbadbadbad\n",
				"inbox/review.md":                      "x\n",
				"personal/people/jane-doe/jane-doe.md": folderNoteMD("entity", "jane-doe", "personal"),
				"personal/people/jane-doe/meeting/2026/2026-01-01-kickoff.md": kickoffBody,
				"personal/people/jane-doe/meeting/2026/2026-01-01-other.md":   strings.Replace(kickoffBody, "Send the deck", "Book the room", 1),
			},
			want: []vault.Finding{
				{Rule: 21, Severity: vault.SeverityWarning, Path: "personal/people/jane-doe/meeting/2026/2026-01-01-other.md", Message: "open inline task \"Book the room\" has no matching todos.md row"},
				{Rule: 21, Severity: vault.SeverityWarning, Path: "todos.md", Message: "todos row event_id \"2026-01-01-badbadbadbad\" does not match personal/people/jane-doe/meeting/2026/2026-01-01-kickoff.md"},
				{Rule: 21, Severity: vault.SeverityWarning, Path: "todos.md", Message: "todos row references a file that does not exist: [[personal/people/jane-doe/meeting/2026/2026-01-01-nope]]"},
			},
		},
		{
			name: "empty todos with no inline tasks is clean",
			files: map[string]string{
				"todos.md":                             "# Todos\n",
				"inbox/review.md":                      "x\n",
				"personal/people/jane-doe/jane-doe.md": folderNoteMD("entity", "jane-doe", "personal"),
				"personal/people/jane-doe/meeting/2026/2026-01-01-kickoff.md": "---\nevent_id: 2026-01-01-abcdef123456\ndate: 2026-01-01\ntype: meeting\nperspective: personal\nprivacy: private\nentities:\n  - personal/people/jane-doe\norigin: generated\nsource: manual\nsummary: Kickoff\n---\n\nBody.\n\n## Actions\n\nNothing to do.\n",
			},
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tax := scopedTax("personal", nil, map[string]string{"people": "entity"}, []string{"meeting"})
			got := byRule(lintVault(t, tc.files, tax), 21)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("R21 findings mismatch:\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}
