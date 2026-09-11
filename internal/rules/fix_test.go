package rules

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Thaelvyn/secondBrain-linter/internal/taxonomy"
	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// personalBase returns the ancestor folder notes every personal entry needs so
// the R3 fix does not create unrelated scope/category notes in the tests.
func personalBase() map[string]string {
	return map[string]string{
		"personal/personal.md":      folderNoteMD("collection", "personal", "personal"),
		"personal/people/people.md": folderNoteMD("entity", "people", "personal"),
	}
}

// fixSequence scans files, applies the fix, re-scans and applies it again. It
// returns the changed paths of both runs; the second must be empty for
// idempotency. The final vault is returned for content assertions.
func fixSequence(t *testing.T, files map[string]string, tax *taxonomy.Taxonomy, opts FixOptions) (first, second []string, v *vault.Vault) {
	t.Helper()
	root, err := scanTestVaultDir(t, files)
	if err != nil {
		t.Fatal(err)
	}
	v1, err := vault.Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	c1 := New(v1, tax)
	first, ferr := c1.Fix(opts)
	if ferr != nil {
		t.Fatal(ferr)
	}
	v, err = vault.Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	c2 := New(v, tax)
	second, ferr = c2.Fix(opts)
	if ferr != nil {
		t.Fatal(ferr)
	}
	return first, second, v
}

func contentOf(v *vault.Vault, path string) string {
	if f := v.ByPath[path]; f != nil {
		return f.Content
	}
	return ""
}

func TestFixR19RewriteAndIdempotency(t *testing.T) {
	files := personalBase()
	files["personal/people/jane-doe/jane-doe.md"] = folderNoteMD("entity", "jane-doe", "personal")
	files["personal/people/jane-doe/facts/2026/2026-01-01-a.md"] = entryMD("2026-01-01-abcdef123456", "2026-01-01", "fact")
	files["personal/people/jane-doe/facts/2026/2026-01-01-b.md"] = entryMD("2026-01-01-abcdef123457", "2026-01-01", "fact")
	tax := scopedTax("personal", nil, map[string]string{"people": "entity"}, []string{"fact"})

	first, second, v := fixSequence(t, files, tax, FixOptions{})
	if !reflect.DeepEqual(first, []string{"personal/people/jane-doe/jane-doe.md"}) {
		t.Fatalf("first fix changed = %v", first)
	}
	if len(second) != 0 {
		t.Fatalf("second fix must be a no-op, changed = %v", second)
	}
	if !strings.Contains(contentOf(v, "personal/people/jane-doe/jane-doe.md"), "children_counts: {facts: 2}") {
		t.Errorf("counts not rewritten: %q", contentOf(v, "personal/people/jane-doe/jane-doe.md"))
	}
}

func TestFixR19CollectionYearRendering(t *testing.T) {
	files := personalBase()
	files["personal/health/health.md"] = "---\ntype: collection\nname: health\nscope: personal\nstatus: active\ncreated: 2026-09-11\n---\n"
	files["personal/health/2026/2026-01-01-x.md"] = entryMD("2026-01-01-abcdef123456", "2026-01-01", "fact")
	tax := scopedTax("personal", nil, map[string]string{"people": "entity", "health": "collection"}, []string{"fact"})

	first, second, v := fixSequence(t, files, tax, FixOptions{})
	if !reflect.DeepEqual(first, []string{"personal/health/health.md"}) {
		t.Fatalf("first fix changed = %v", first)
	}
	if len(second) != 0 {
		t.Fatalf("second fix must be a no-op, changed = %v", second)
	}
	if !strings.Contains(contentOf(v, "personal/health/health.md"), `children_counts: {"2026": 1}`) {
		t.Errorf("year key must be quoted: %q", contentOf(v, "personal/health/health.md"))
	}
}

func TestFixR3CreatesFolderNoteAndIdempotency(t *testing.T) {
	files := personalBase()
	files["personal/people/jane-doe/facts/2026/2026-01-01-a.md"] = entryMD("2026-01-01-abcdef123456", "2026-01-01", "fact")
	tax := scopedTax("personal", nil, map[string]string{"people": "entity"}, []string{"fact"})

	first, second, v := fixSequence(t, files, tax, FixOptions{})
	want := []string{"personal/people/jane-doe/jane-doe.md"}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("first fix changed = %v, want %v", first, want)
	}
	if len(second) != 0 {
		t.Fatalf("second fix must be a no-op, changed = %v", second)
	}
	note := contentOf(v, "personal/people/jane-doe/jane-doe.md")
	for _, part := range []string{
		"type: entity\n",
		"name: Jane Doe\n",
		"scope: personal\n",
		"status: active\n",
		"children_counts: {facts: 1}\n",
	} {
		if !strings.Contains(note, part) {
			t.Errorf("created note missing %q:\n%s", part, note)
		}
	}
}

func TestFixR11NormalizesRelatedAndIdempotency(t *testing.T) {
	files := personalBase()
	files["personal/people/jane-doe/jane-doe.md"] = folderNoteMD("entity", "jane-doe", "personal")
	files["personal/people/jane-doe/meeting/2026/2026-01-01-x.md"] = `---
event_id: 2026-01-01-abcdef123456
date: 2026-01-01
type: meeting
perspective: personal
privacy: private
entities:
  - personal/people/jane-doe
origin: generated
source: manual
summary: x
related:
  - personal/people/jane-doe
---
`
	tax := scopedTax("personal", nil, map[string]string{"people": "entity"}, []string{"meeting"})

	first, second, v := fixSequence(t, files, tax, FixOptions{})
	want := []string{
		"personal/people/jane-doe/jane-doe.md",
		"personal/people/jane-doe/meeting/2026/2026-01-01-x.md",
	}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("first fix changed = %v, want %v", first, want)
	}
	if len(second) != 0 {
		t.Fatalf("second fix must be a no-op, changed = %v", second)
	}
	if !strings.Contains(contentOf(v, "personal/people/jane-doe/meeting/2026/2026-01-01-x.md"), `  - "[[personal/people/jane-doe]]"`) {
		t.Errorf("related not normalized: %q", contentOf(v, "personal/people/jane-doe/meeting/2026/2026-01-01-x.md"))
	}
}

func TestFixRecountOnly(t *testing.T) {
	files := personalBase()
	files["personal/people/jane-doe/jane-doe.md"] = folderNoteMD("entity", "jane-doe", "personal")
	files["personal/people/jane-doe/facts/2026/2026-01-01-a.md"] = entryMD("2026-01-01-abcdef123456", "2026-01-01", "fact")
	files["personal/people/jane-doe/meeting/2026/2026-01-01-b.md"] = `---
event_id: 2026-01-01-abcdef123457
date: 2026-01-01
type: meeting
perspective: personal
privacy: private
entities:
  - personal/people/jane-doe
origin: generated
source: manual
summary: x
related:
  - personal/people/jane-doe
---
`
	tax := scopedTax("personal", nil, map[string]string{"people": "entity"}, []string{"fact", "meeting"})

	first, second, v := fixSequence(t, files, tax, FixOptions{RecountOnly: true})
	want := []string{"personal/people/jane-doe/jane-doe.md"}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("recount must only touch the children_counts note, changed = %v", first)
	}
	if len(second) != 0 {
		t.Fatalf("second recount must be a no-op, changed = %v", second)
	}
	if !strings.Contains(contentOf(v, "personal/people/jane-doe/jane-doe.md"), "children_counts: {facts: 1, meeting: 1}") {
		t.Errorf("recount did not rewrite counts: %q", contentOf(v, "personal/people/jane-doe/jane-doe.md"))
	}
	// R11 must not have run: the bare related path is untouched.
	if c := contentOf(v, "personal/people/jane-doe/meeting/2026/2026-01-01-b.md"); !strings.Contains(c, "- personal/people/jane-doe") {
		t.Errorf("recount must not normalize related: %q", c)
	}
}

func TestFixChangedFilter(t *testing.T) {
	files := personalBase()
	files["personal/people/jane-doe/jane-doe.md"] = folderNoteMD("entity", "jane-doe", "personal")
	files["personal/people/jane-doe/facts/2026/2026-01-01-a.md"] = entryMD("2026-01-01-abcdef123456", "2026-01-01", "fact")
	files["personal/health/health.md"] = folderNoteMD("collection", "health", "personal")
	files["personal/health/2026/2026-01-01-b.md"] = entryMD("2026-01-01-abcdef123457", "2026-01-01", "fact")
	tax := scopedTax("personal", nil, map[string]string{"people": "entity", "health": "collection"}, []string{"fact"})

	changed := map[string]bool{"personal/people/jane-doe/facts/2026/2026-01-01-a.md": true}
	_, _, v := fixSequence(t, files, tax, FixOptions{Changed: changed})
	if c := contentOf(v, "personal/people/jane-doe/jane-doe.md"); !strings.Contains(c, "children_counts: {facts: 1}") {
		t.Errorf("affected note must be recounted: %q", c)
	}
	if c := contentOf(v, "personal/health/health.md"); !strings.Contains(c, "children_counts: {}") {
		t.Errorf("unaffected health note must keep its empty counts: %q", c)
	}
}
