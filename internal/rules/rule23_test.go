package rules

import (
	"reflect"
	"testing"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

const rule23EntryPath = "personal/people/jane-doe/meeting/2026/2026-01-01-x.md"

// rule23Entry builds an entry whose `entities` frontmatter is `entities`
// (already indented as a YAML list) and whose body is `body`.
func rule23Entry(eid, entities, body string) string {
	return rule23EntryWithFM(eid, entities, "", body)
}

// rule23EntryWithFM is rule23Entry with additional frontmatter lines inserted
// after `entities` and before the closing `---`.
func rule23EntryWithFM(eid, entities, extraFM, body string) string {
	return "---\nevent_id: " + eid + "\ndate: 2026-01-01\ntype: meeting\nperspective: personal\nprivacy: private\nentities:\n" +
		entities + extraFM + "origin: generated\nsource: manual\nsummary: x\n---\n" + body
}

func TestRule23(t *testing.T) {
	const entityFM = "  - personal/people/jane-doe\n"
	const collectionFM = "  - personal/health\n"
	folderNotes := map[string]string{
		"personal/people/jane-doe/jane-doe.md": folderNoteMD("entity", "jane-doe", "personal"),
		"personal/health/health.md":            folderNoteMD("collection", "health", "personal"),
	}
	cases := []struct {
		name  string
		files map[string]string
		want  []vault.Finding
	}{
		{
			name: "all entities linked in prose",
			files: mergeFiles(folderNotes, map[string]string{
				"todos.md":        "x\n",
				"inbox/review.md": "x\n",
				rule23EntryPath:   rule23Entry("2026-01-01-abcdef123456", entityFM, "\nSpoke with [[personal/people/jane-doe]] today.\n"),
			}),
		},
		{
			name: "missing link to entity folder note is an error",
			files: mergeFiles(folderNotes, map[string]string{
				"todos.md":        "x\n",
				"inbox/review.md": "x\n",
				rule23EntryPath:   rule23Entry("2026-01-01-abcdef123456", entityFM, "\nNo link here.\n"),
			}),
			want: []vault.Finding{{
				Rule:     23,
				Severity: vault.SeverityError,
				Path:     rule23EntryPath,
				Message:  `entry body is missing a wikilink to entity "personal/people/jane-doe"`,
			}},
		},
		{
			name: "missing link to collection folder note is a warning",
			files: mergeFiles(folderNotes, map[string]string{
				"todos.md":        "x\n",
				"inbox/review.md": "x\n",
				rule23EntryPath:   rule23Entry("2026-01-01-abcdef123456", collectionFM, "\nNo link here.\n"),
			}),
			want: []vault.Finding{{
				Rule:     23,
				Severity: vault.SeverityWarning,
				Path:     rule23EntryPath,
				Message:  `entry body is missing a wikilink to entity "personal/health"`,
			}},
		},
		{
			name: "frontmatter related does not count",
			files: mergeFiles(folderNotes, map[string]string{
				"todos.md":        "x\n",
				"inbox/review.md": "x\n",
				rule23EntryPath: rule23EntryWithFM("2026-01-01-abcdef123456", entityFM,
					"related:\n  - \"[[personal/people/jane-doe]]\"\n", "\nNo link here.\n"),
			}),
			want: []vault.Finding{{
				Rule:     23,
				Severity: vault.SeverityError,
				Path:     rule23EntryPath,
				Message:  `entry body is missing a wikilink to entity "personal/people/jane-doe"`,
			}},
		},
		{
			name: "entity path without a folder note is ignored",
			files: map[string]string{
				"todos.md":        "x\n",
				"inbox/review.md": "x\n",
				rule23EntryPath:   rule23Entry("2026-01-01-abcdef123456", "  - personal/people/ghost\n", "\nNo link here.\n"),
			},
		},
		{
			name: "daily-log entry is exempt",
			files: mergeFiles(folderNotes, map[string]string{
				"todos.md":                           "x\n",
				"inbox/review.md":                    "x\n",
				"daily-logs/2026/01/2026-01-01-x.md": rule23Entry("2026-01-01-abcdef123456", entityFM, "\nNo link here.\n"),
			}),
		},
		{
			name: "system status entry is exempt",
			files: mergeFiles(folderNotes, map[string]string{
				"todos.md":                            "x\n",
				"inbox/review.md":                     "x\n",
				"_system/status/lint/2026-01-01-x.md": rule23Entry("2026-01-01-abcdef123456", entityFM, "\nNo link here.\n"),
			}),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := byRule(lintVault(t, tc.files, emptyTax()), 23)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("R23 findings mismatch:\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}

func TestRule23LinkForms(t *testing.T) {
	const entityFM = "  - personal/people/jane-doe\n"
	folderNotes := map[string]string{
		"personal/people/jane-doe/jane-doe.md": folderNoteMD("entity", "jane-doe", "personal"),
	}
	cases := []struct {
		name string
		body string
	}{
		{name: "plain path", body: "\nSee [[personal/people/jane-doe]].\n"},
		{name: "alias stripped", body: "\nSee [[personal/people/jane-doe|Jane Doe]].\n"},
		{name: "anchor stripped", body: "\nSee [[personal/people/jane-doe#Context]].\n"},
		{name: "md suffix", body: "\nSee [[personal/people/jane-doe.md]].\n"},
		{name: "body Related section", body: "\nLede.\n\n## Related\n\n- [[personal/people/jane-doe]]\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := mergeFiles(folderNotes, map[string]string{
				"todos.md":        "x\n",
				"inbox/review.md": "x\n",
				rule23EntryPath:   rule23Entry("2026-01-01-abcdef123456", entityFM, tc.body),
			})
			if got := byRule(lintVault(t, files, emptyTax()), 23); got != nil {
				t.Fatalf("expected no R23 findings, got: %+v", got)
			}
		})
	}
}

// mergeFiles returns a copy of base with extra's entries added.
func mergeFiles(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
