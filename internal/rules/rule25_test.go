package rules

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Thaelvyn/secondBrain-linter/internal/taxonomy"
	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// personNoteMDCounts is personNoteMD with a children_counts line, so the R19
// fix stays silent in R25 fix tests. The body is appended after the
// frontmatter.
func personNoteMDCounts(name, extra, counts, body string) string {
	return "---\ntype: entity\nname: " + name + "\nscope: personal\n" + extra +
		"children_counts: " + counts + "\n---\n" + body
}

func TestRule25MissingMirrorWarning(t *testing.T) {
	files := personBaseFiles()
	files[aliceNote] = personBodyMD("Alice", "", relationsSection("partner [[personal/people/bob/bob]]"))
	files[bobNote] = personNoteMD("Bob", "")
	want := []vault.Finding{{
		Rule:     25,
		Severity: vault.SeverityWarning,
		Path:     "personal/people/bob/bob.md",
		Message:  "personal/people/bob/bob lacks 'partner [[personal/people/alice/alice]]'",
	}}
	got := byRule(lintVault(t, files, peopleTax()), 25)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("R25 findings mismatch:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestRule25InverseMirrorAndSymmetricPresent(t *testing.T) {
	cases := []struct {
		name  string
		alice []string
		bob   []string
		want  string // expected warning on bob, "" when clean
	}{
		{
			name:  "symmetric present is clean",
			alice: []string{"friend [[personal/people/bob/bob]]"},
			bob:   []string{"friend [[personal/people/alice/alice]]"},
		},
		{
			name:  "inverse mirror present is clean",
			alice: []string{"manager [[personal/people/bob/bob]]"},
			bob:   []string{"report [[personal/people/alice/alice]]"},
		},
		{
			name:  "inverse mirror missing",
			alice: []string{"manager [[personal/people/bob/bob]]"},
			bob:   nil,
			want:  "personal/people/bob/bob lacks 'report [[personal/people/alice/alice]]'",
		},
		{
			name:  "wrong mirror kind does not satisfy",
			alice: []string{"manager [[personal/people/bob/bob]]"},
			bob:   []string{"colleague [[personal/people/charlie/charlie]]"},
			want:  "personal/people/bob/bob lacks 'report [[personal/people/alice/alice]]'",
		},
		{
			name:  "mirror with prose still satisfies",
			alice: []string{"manager [[personal/people/bob/bob]]"},
			bob:   []string{"report [[personal/people/alice/alice]] — since 2020"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := personBaseFiles()
			files[aliceNote] = personBodyMD("Alice", "", relationsSection(tc.alice...))
			files[bobNote] = personBodyMD("Bob", "", relationsSection(tc.bob...))
			if strings.Contains(strings.Join(tc.bob, "\n"), "charlie") {
				files["personal/people/charlie/charlie.md"] = personBodyMD("Charlie", "",
					relationsSection("colleague [[personal/people/bob/bob]]"))
			}
			got := byRule(lintVault(t, files, peopleTax()), 25)
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("expected no R25 findings, got: %+v", got)
				}
				return
			}
			if len(got) != 1 || got[0].Path != bobNote || got[0].Message != tc.want {
				t.Fatalf("R25 findings mismatch:\n got: %+v\nwant one on bob: %s", got, tc.want)
			}
		})
	}
}

func TestRule25SelfLoopIsNotAnEdge(t *testing.T) {
	files := personBaseFiles()
	files[aliceNote] = personBodyMD("Alice", "", relationsSection("partner [[personal/people/alice/alice]]"))
	if got := byRule(lintVault(t, files, peopleTax()), 25); got != nil {
		t.Fatalf("a self-loop must not produce a mirror warning, got: %+v", got)
	}
}

func TestRule25NoRelationKindsNoOp(t *testing.T) {
	files := personBaseFiles()
	files[aliceNote] = personBodyMD("Alice", "", relationsSection("partner [[personal/people/bob/bob]]"))
	files[bobNote] = personNoteMD("Bob", "")
	got := byRule(lintVault(t, files, scopedTax("personal", nil, map[string]string{"people": "entity"}, nil)), 25)
	if got != nil {
		t.Fatalf("R25 must no-op without relation_kinds, got: %+v", got)
	}
}

func TestFixR25AppendsIntoExistingSectionAndIdempotent(t *testing.T) {
	files := personBaseFiles()
	files[aliceNote] = personNoteMDCounts("Alice", "", "{}",
		relationsSection("partner [[personal/people/bob/bob]]"))
	files[bobNote] = personNoteMDCounts("Bob", "", "{}",
		"\n## Relations\n\n- friend [[personal/people/charlie/charlie]]\n\n## Notes\n\nHuman notes.\n")
	files["personal/people/charlie/charlie.md"] = personNoteMDCounts("Charlie", "", "{}",
		relationsSection("friend [[personal/people/bob/bob]]"))

	first, second, v := fixSequence(t, files, peopleTax(), FixOptions{})
	want := []string{"personal/people/bob/bob.md"}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("first fix changed = %v, want %v", first, want)
	}
	if len(second) != 0 {
		t.Fatalf("second fix must be a no-op, changed = %v", second)
	}
	content := contentOf(v, bobNote)
	if !strings.Contains(content, "- partner [[personal/people/alice/alice]]\n") {
		t.Errorf("mirror not appended to bob:\n%s", content)
	}
	if !strings.Contains(content, "- friend [[personal/people/charlie/charlie]]\n") {
		t.Errorf("existing bob item must be preserved:\n%s", content)
	}
	// Appended before the next H2, not at the end of the file.
	if strings.Index(content, "partner [[personal/people/alice/alice]]") > strings.Index(content, "## Notes") {
		t.Errorf("mirror must land inside the Relations section:\n%s", content)
	}
	if !strings.Contains(content, "Human notes.\n") {
		t.Errorf("other content must be preserved:\n%s", content)
	}
	if v2 := contentOf(v, aliceNote); !strings.Contains(v2, "partner [[personal/people/bob/bob]]") {
		t.Errorf("alice must be untouched:\n%s", v2)
	}
}

func TestFixR25CreatesSectionWhenAbsent(t *testing.T) {
	files := personBaseFiles()
	files[aliceNote] = personNoteMDCounts("Alice", "", "{}",
		relationsSection("manager [[personal/people/bob/bob]]"))
	files[bobNote] = personNoteMDCounts("Bob", "", "{}", "\nSome body text.\n")

	_, _, v := fixSequence(t, files, peopleTax(), FixOptions{})
	content := contentOf(v, bobNote)
	if !strings.Contains(content, "\n## Relations\n\n- report [[personal/people/alice/alice]]\n") {
		t.Errorf("Relations section must be created at the end of bob:\n%s", content)
	}
	if !strings.HasSuffix(content, "- report [[personal/people/alice/alice]]\n") {
		t.Errorf("new section must be at the end of the file:\n%s", content)
	}
	if !strings.Contains(content, "\nSome body text.\n") {
		t.Errorf("existing body must be preserved:\n%s", content)
	}
}

func TestFixR25CreatesSectionAfterTrailingNewline(t *testing.T) {
	files := personBaseFiles()
	files[aliceNote] = personNoteMDCounts("Alice", "", "{}",
		relationsSection("manager [[personal/people/bob/bob]]"))
	// Body ends with a newline: no extra blank line before the new heading.
	files[bobNote] = personNoteMDCounts("Bob", "", "{}", "\nSome body text.\n")

	_, _, v := fixSequence(t, files, peopleTax(), FixOptions{})
	content := contentOf(v, bobNote)
	if strings.Contains(content, "Some body text.\n\n\n## Relations") {
		t.Errorf("at most one blank line must separate the new section:\n%q", content)
	}
}

func TestFixR25EmptyRelationsSectionGetsItem(t *testing.T) {
	files := personBaseFiles()
	files[aliceNote] = personNoteMDCounts("Alice", "", "{}",
		relationsSection("manager [[personal/people/bob/bob]]"))
	files[bobNote] = personNoteMDCounts("Bob", "", "{}", "\n## Relations\n\n## Notes\n\nText.\n")

	first, second, v := fixSequence(t, files, peopleTax(), FixOptions{})
	if !reflect.DeepEqual(first, []string{"personal/people/bob/bob.md"}) {
		t.Fatalf("first fix changed = %v", first)
	}
	if len(second) != 0 {
		t.Fatalf("second fix must be a no-op, changed = %v", second)
	}
	content := contentOf(v, bobNote)
	if !strings.Contains(content, "## Relations\n- report [[personal/people/alice/alice]]\n\n## Notes") {
		t.Errorf("item must be inserted inside the empty section:\n%s", content)
	}
}

func TestFixR25AppendsInsideSection(t *testing.T) {
	files := personBaseFiles()
	files[aliceNote] = personNoteMDCounts("Alice", "", "{}",
		relationsSection("manager [[personal/people/bob/bob]]"))
	files[bobNote] = personNoteMDCounts("Bob", "", "{}",
		"\n## Relations\n\n- friend [[personal/people/charlie/charlie]]\n  - a nested note\n\n## Notes\n\nText.\n")

	_, _, v := fixSequence(t, files, peopleTax(), FixOptions{})
	content := contentOf(v, bobNote)
	idxItem := strings.Index(content, "- report [[personal/people/alice/alice]]")
	idxNested := strings.Index(content, "  - a nested note")
	idxNotes := strings.Index(content, "## Notes")
	if idxItem < 0 || idxItem < idxNested || idxItem > idxNotes {
		t.Errorf("mirror must be appended after the section's last top-level item:\n%s", content)
	}
}

func TestFixR25RecountOnlySkipsRelations(t *testing.T) {
	tax := scopedTax("personal", nil, map[string]string{"people": "entity"}, []string{"fact"})
	tax.RelationKinds = testRelationKinds
	files := personBaseFiles()
	files[aliceNote] = personNoteMDCounts("Alice", "", "{}",
		relationsSection("partner [[personal/people/bob/bob]]"))
	files[bobNote] = personNoteMDCounts("Bob", "", "{}", "")
	files["personal/people/charlie/charlie.md"] = personNoteMDCounts("Charlie", "", "{}", "")
	files["personal/people/charlie/facts/2026/2026-01-01-a.md"] = entryMD("2026-01-01-abcdef123456", "2026-01-01", "fact")

	first, second, v := fixSequence(t, files, tax, FixOptions{RecountOnly: true})
	if !reflect.DeepEqual(first, []string{"personal/people/charlie/charlie.md"}) {
		t.Fatalf("recount must only touch children_counts, changed = %v", first)
	}
	if len(second) != 0 {
		t.Fatalf("second recount must be a no-op, changed = %v", second)
	}
	if c := contentOf(v, bobNote); strings.Contains(c, "## Relations") {
		t.Errorf("recount must not append a Relations section to bob:\n%s", c)
	}
}

func TestRule25IgnoresInvalidEdges(t *testing.T) {
	// Only the well-formed edge may trigger a mirror warning; every other kind
	// of broken item is R24's business and must be skipped silently here.
	files := personBaseFiles()
	files[aliceNote] = personBodyMD("Alice", "", relationsSection(
		"ok",                                   // malformed: no space, no wikilink
		"soulmate [[personal/people/bob/bob]]", // unknown kind
		"partner [[personal/people/ghost/ghost]]", // unresolvable target
		"partner [[personal/people/alice/alice]]", // self-loop
		"partner [[personal/health/health]]",      // target not a person note
		"\"partner [[personal/people/bob/bob]]\"", // quoted: malformed body item
		"partner [[personal/people/bob/bob]]",     // the only valid edge
	))
	files[bobNote] = personNoteMD("Bob", "")
	files["personal/health/health.md"] = "---\ntype: collection\nname: health\nscope: personal\n---\n"
	want := []vault.Finding{{
		Rule:     25,
		Severity: vault.SeverityWarning,
		Path:     "personal/people/bob/bob.md",
		Message:  "personal/people/bob/bob lacks 'partner [[personal/people/alice/alice]]'",
	}}
	got := byRule(lintVault(t, files, peopleTax()), 25)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("R25 findings mismatch:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestRelationHelpersDirect(t *testing.T) {
	c := &Context{Tax: nil}
	if got := c.missingMirrors(); len(got) != 0 {
		t.Fatalf("missingMirrors without taxonomy = %v, want empty", got)
	}
	if got := c.relationEdges(nil); got != nil {
		t.Fatalf("relationEdges(nil) = %v", got)
	}
	c.Tax = &taxonomy.Taxonomy{RelationKinds: testRelationKinds}
	if _, ok := c.mirrorKind("nope"); ok {
		t.Fatal("unknown kind must have no mirror")
	}
	if got := relationsBodyItems("no heading here\n"); got != nil {
		t.Fatalf("relationsBodyItems without heading = %v", got)
	}
}

func TestRule25KindWithoutMirrorDefinition(t *testing.T) {
	tax := scopedTax("personal", nil, map[string]string{"people": "entity"}, nil)
	tax.RelationKinds = map[string]taxonomy.RelationKind{
		"plotter": {}, // neither symmetric nor inverse: no mirror can exist
	}
	files := personBaseFiles()
	files[aliceNote] = personBodyMD("Alice", "", relationsSection("plotter [[personal/people/bob/bob]]"))
	files[bobNote] = personNoteMD("Bob", "")
	if got := byRule(lintVault(t, files, tax), 25); got != nil {
		t.Fatalf("expected no R25 findings, got: %+v", got)
	}
}

func TestAppendRelationsBodyItemEdgeCases(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "create section at end of frontmatter-only note",
			content: "---\ntype: entity\n---\n",
			want:    "---\ntype: entity\n---\n\n## Relations\n\n- a [[x]]\n",
		},
		{
			name:    "create section after body without trailing newline",
			content: "---\ntype: entity\n---\nBody",
			want:    "---\ntype: entity\n---\nBody\n\n## Relations\n\n- a [[x]]\n",
		},
		{
			name:    "append to existing list",
			content: "---\ntype: entity\n---\n\n## Relations\n\n- old [[y]]\n",
			want:    "---\ntype: entity\n---\n\n## Relations\n\n- old [[y]]\n- a [[x]]\n",
		},
		{
			name:    "insert into empty section before next heading",
			content: "---\ntype: entity\n---\n\n## Relations\n\n## Next\n\ntext\n",
			want:    "---\ntype: entity\n---\n\n## Relations\n- a [[x]]\n\n## Next\n\ntext\n",
		},
		{
			name:    "last item at EOF without trailing newline",
			content: "---\ntype: entity\n---\n\n## Relations\n\n- old [[y]]",
			want:    "---\ntype: entity\n---\n\n## Relations\n\n- old [[y]]\n- a [[x]]\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, changed := appendRelationsBodyItem(tc.content, "a [[x]]")
			if !changed {
				t.Fatal("changed = false, want true")
			}
			if got != tc.want {
				t.Fatalf("content mismatch:\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

func TestFixR25ChangedFilter(t *testing.T) {
	files := personBaseFiles()
	files[aliceNote] = personNoteMDCounts("Alice", "", "{}",
		relationsSection("partner [[personal/people/bob/bob]]"))
	files[bobNote] = personNoteMDCounts("Bob", "", "{}", "")

	changed := map[string]bool{bobNote: true}
	_, _, v := fixSequence(t, files, peopleTax(), FixOptions{Changed: changed})
	if c := contentOf(v, bobNote); !strings.Contains(c, "- partner [[personal/people/alice/alice]]") {
		t.Errorf("bob in changed set must be fixed:\n%s", c)
	}

	// A changed source alone does not touch the untouched target.
	files2 := personBaseFiles()
	files2[aliceNote] = personNoteMDCounts("Alice", "", "{}",
		relationsSection("partner [[personal/people/bob/bob]]"))
	files2[bobNote] = personNoteMDCounts("Bob", "", "{}", "")
	changed2 := map[string]bool{aliceNote: true}
	_, _, v2 := fixSequence(t, files2, peopleTax(), FixOptions{Changed: changed2})
	if c := contentOf(v2, bobNote); strings.Contains(c, "## Relations") {
		t.Errorf("unchanged bob must not be fixed when only alice changed:\n%s", c)
	}
}
