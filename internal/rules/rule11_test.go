package rules

import (
	"reflect"
	"testing"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

const relatedEntryFields = `event_id: 2026-01-01-abcdef123456
date: 2026-01-01
type: meeting
perspective: personal
privacy: private
entities:
  - personal/people/jane-doe
origin: generated
source: manual
summary: x
`

// relatedEntryWithFM builds an entry with `extraFM` appended to its frontmatter.
func relatedEntryWithFM(extraFM, body string) string {
	return "---\n" + relatedEntryFields + extraFM + "---\n" + body
}

func relatedEntry(body string) string { return relatedEntryWithFM("", body) }

const relatedEntryPath = "personal/people/jane-doe/meeting/2026/2026-01-01-x.md"

func TestRule11BodyRelated(t *testing.T) {
	bodyErr := []vault.Finding{{
		Rule:     11,
		Severity: vault.SeverityError,
		Path:     relatedEntryPath,
		Message:  "body Related section must contain only wikilinks ([[full/path]])",
	}}
	cases := []struct {
		name string
		body string
		want []vault.Finding
	}{
		{
			name: "no body Related section is no finding",
			body: "\n\nBody.\n\n## Context\n\n## Discussion\n",
		},
		{
			name: "all wikilink items are clean",
			body: "\n\n## Related\n\n- [[personal/people/jane-doe]]\n- [[personal/people/jane-doe/meeting/2026/2026-01-01-y]]\n",
		},
		{
			name: "bare path item is an error",
			body: "\n\n## Related\n\n- personal/people/jane-doe\n",
			want: bodyErr,
		},
		{
			name: "Related at EOF without trailing newline is detected",
			body: "\n\n## Related\n\n- personal/people/jane-doe",
			want: bodyErr,
		},
		{
			name: "items after the next heading are not scanned",
			body: "\n\n## Related\n\n- [[personal/people/jane-doe]]\n\n## Context\n\n- not-a-link\n",
		},
		{
			name: "asterisk and indented items are checked",
			body: "\n\n## Related\n\n* [[personal/people/jane-doe]]\n  - personal/people/jane-doe\n",
			want: bodyErr,
		},
		{
			name: "optional marker suffix still matches the heading",
			body: "\n\n## Related <!-- optional -->\n\n- personal/people/jane-doe\n",
			want: bodyErr,
		},
		{
			name: "one finding per entry even with several bare items",
			body: "\n\n## Related\n\n- personal/people/a\n- personal/people/b\n",
			want: bodyErr,
		},
		{
			name: "heading with different text is not the Related section",
			body: "\n\n## Context\n\n- personal/people/jane-doe\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{
				"todos.md":        "x\n",
				"inbox/review.md": "x\n",
				relatedEntryPath:  relatedEntry(tc.body),
			}
			got := byRule(lintVault(t, files, emptyTax()), 11)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("R11 findings mismatch:\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}

func TestRule11FrontmatterRelatedUnchanged(t *testing.T) {
	cases := []struct {
		name string
		fm   string
		want []vault.Finding
	}{
		{
			name: "bare frontmatter related still errors",
			fm:   "related:\n  - personal/people/jane-doe\n",
			want: []vault.Finding{{
				Rule:     11,
				Severity: vault.SeverityError,
				Path:     relatedEntryPath,
				Message:  "frontmatter related must contain only wikilinks ([[full/path]])",
			}},
		},
		{
			name: "wikilink frontmatter related is clean",
			fm:   "related:\n  - \"[[personal/people/jane-doe]]\"\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{
				"todos.md":        "x\n",
				"inbox/review.md": "x\n",
				relatedEntryPath:  relatedEntryWithFM(tc.fm, "\n\nBody.\n"),
			}
			got := byRule(lintVault(t, files, emptyTax()), 11)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("R11 findings mismatch:\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}

func TestRule11BodyRelatedIgnoresTemplates(t *testing.T) {
	files := map[string]string{
		"todos.md":                     "x\n",
		"inbox/review.md":              "x\n",
		"_system/templates/meeting.md": "---\ntype: meeting\nevent_id:\n---\n\n## Related <!-- optional -->\n\n- personal/people/jane-doe\n",
	}
	got := byRule(lintVault(t, files, emptyTax()), 11)
	if got != nil {
		t.Fatalf("templates must not be entries; got R11 findings: %+v", got)
	}
}

func TestRelatedBodyItems(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{name: "absent heading", content: "---\nx: 1\n---\n\nBody\n", want: nil},
		{
			name:    "entries until next H2",
			content: "---\nx: 1\n---\n\n## Related\n\n- [[a]]\n* [[b]]\n\n## Next\n\n- [[c]]\n",
			want:    []string{"[[a]]", "[[b]]"},
		},
		{
			name:    "EOF section",
			content: "---\nx: 1\n---\n\n## Related\n\n- bare",
			want:    []string{"bare"},
		},
		{
			name:    "marker suffix",
			content: "---\nx: 1\n---\n\n## Related <!-- optional -->\n\n- [[a]]\n",
			want:    []string{"[[a]]"},
		},
		{
			name:    "blank lines and prose ignored",
			content: "---\nx: 1\n---\n\n## Related\n\nSome prose.\n\n- [[a]]\n\n- bare\n",
			want:    []string{"[[a]]", "bare"},
		},
		{
			name:    "no frontmatter still scans body",
			content: "## Related\n\n- [[a]]\n",
			want:    []string{"[[a]]"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := relatedBodyItems(tc.content)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("relatedBodyItems mismatch:\n got: %v\nwant: %v", got, tc.want)
			}
		})
	}
}

func TestRelatedBodyItemsTrimsItems(t *testing.T) {
	content := "---\nx: 1\n---\n\n## Related\n\n-    [[a]]   \n"
	got := relatedBodyItems(content)
	if len(got) != 1 || got[0] != "[[a]]" {
		t.Fatalf("expected trimmed item [[a]], got %q", got)
	}
}
