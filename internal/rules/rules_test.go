package rules

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/Thaelvyn/secondBrain-linter/internal/taxonomy"
	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

func lintFixture(t *testing.T, dir string) []vault.Finding {
	t.Helper()
	v, err := vault.Scan(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	tax, err := taxonomy.Load(dir)
	if err != nil {
		t.Fatalf("taxonomy: %v", err)
	}
	c := New(v, tax)
	Run(c)
	return sortedFindings(c.Findings)
}

func sortedFindings(fs []vault.Finding) []vault.Finding {
	out := append([]vault.Finding(nil), fs...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && lesser(out[j], out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func lesser(a, b vault.Finding) bool {
	if a.Rule != b.Rule {
		return a.Rule < b.Rule
	}
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	return a.Message < b.Message
}

func TestCleanFixture(t *testing.T) {
	fs := lintFixture(t, "../../testdata/clean-fixture")
	if len(fs) != 0 {
		t.Fatalf("clean fixture produced %d findings: %+v", len(fs), fs)
	}
}

func TestViolationsFixtureGolden(t *testing.T) {
	fs := lintFixture(t, "../../testdata/violations-fixture")
	want := []vault.Finding{
		{Rule: 1, Severity: vault.SeverityError, Path: "personal/Hello World.md", Message: "filename slug must be lowercase kebab-case ASCII without spaces"},
		{Rule: 2, Severity: vault.SeverityError, Path: "daily-logs/2026/01/notes.md", Message: "daily-log filename must match {YYYY-MM-DD}-{slug}.md"},
		{Rule: 3, Severity: vault.SeverityError, Path: "personal/people/jane-doe/misc", Message: "entity folder \"personal/people/jane-doe/misc\" is missing its folder note \"personal/people/jane-doe/misc.md\""},
		{Rule: 3, Severity: vault.SeverityError, Path: "personal/people/jane-doe/whatever", Message: "entity folder \"personal/people/jane-doe/whatever\" is missing its folder note \"personal/people/jane-doe/whatever.md\""},
		{Rule: 3, Severity: vault.SeverityError, Path: "personal/projects/apollo-missing", Message: "entity folder \"personal/projects/apollo-missing\" is missing its folder note \"personal/projects/apollo-missing.md\""},
		{Rule: 4, Severity: vault.SeverityError, Path: "work/ghostco", Message: "company folder \"ghostco\" is not in the companies list of _system/taxonomy.yaml"},
		{Rule: 5, Severity: vault.SeverityError, Path: "personal/people/jane-doe/meeting/2026/2026-01-01-nosummary.md", Message: "entry missing required frontmatter field(s): summary"},
		{Rule: 6, Severity: vault.SeverityError, Path: "personal/people/jane-doe/meeting/2026/2026-01-01-badprivacy.md", Message: "privacy \"top-secret\" not in {public, private, secret}"},
		{Rule: 6, Severity: vault.SeverityError, Path: "personal/people/jane-doe/meeting/2026/2026-01-01-notype.md", Message: "type \"hologram\" is not in the event_types catalog of _system/taxonomy.yaml"},
		{Rule: 6, Severity: vault.SeverityError, Path: "personal/people/jane-doe/meeting/2026/2026-13-01-baddate.md", Message: "date \"2026-13-01\" is not a valid YYYY-MM-DD"},
		{Rule: 7, Severity: vault.SeverityError, Path: "personal/admin/admin.md", Message: "invalid folder-note frontmatter: type \"banana\" not in {entity, collection}"},
		{Rule: 8, Severity: vault.SeverityError, Path: "personal/people/jane-doe/meeting/2026/2026-01-01-badid.md", Message: "event_id \"2026-01-01-nothexatall\" does not match {YYYY-MM-DD}-{12-hex}"},
		{Rule: 9, Severity: vault.SeverityError, Path: "personal/people/jane-doe/meeting/2026/2026-01-02-dupe-a.md", Message: "duplicate event_id 2026-01-02-cafebabe0001 across 2 entries with no mutual related links between them"},
		{Rule: 10, Severity: vault.SeverityError, Path: "personal/people/jane-doe/meeting/2026/2026-01-01-badlink.md", Message: "wikilink [[personal/people/zelda]] does not resolve to an existing file"},
		{Rule: 11, Severity: vault.SeverityError, Path: "personal/people/jane-doe/meeting/2026/2026-01-01-plainrelated.md", Message: "frontmatter entities must contain only plain paths, no brackets"},
		{Rule: 11, Severity: vault.SeverityError, Path: "personal/people/jane-doe/meeting/2026/2026-01-01-plainrelated.md", Message: "frontmatter related must contain only wikilinks ([[full/path]])"},
		{Rule: 11, Severity: vault.SeverityError, Path: "personal/people/jane-doe/meeting/2026/2026-01-01-plainrelated.md", Message: "frontmatter source must contain only plain paths, no brackets"},
		{Rule: 12, Severity: vault.SeverityError, Path: "personal/people/jane-doe/meeting/2026/2026-01-01-embed.md", Message: "embeds are forbidden: found ![[...]]"},
		{Rule: 13, Severity: vault.SeverityError, Path: "personal/people/jane-doe/meeting/2026/2026-01-01-triple-a.md", Message: "perspective group 2026-01-01-deadbeef0001 must cross-reference every other member (full mesh); missing related: personal/people/jane-doe/meeting/2026/2026-01-01-triple-a.md -> personal/people/jane-doe/meeting/2026/2026-01-01-triple-c.md; personal/people/jane-doe/meeting/2026/2026-01-01-triple-b.md -> personal/people/jane-doe/meeting/2026/2026-01-01-triple-c.md; personal/people/jane-doe/meeting/2026/2026-01-01-triple-c.md -> personal/people/jane-doe/meeting/2026/2026-01-01-triple-a.md; personal/people/jane-doe/meeting/2026/2026-01-01-triple-c.md -> personal/people/jane-doe/meeting/2026/2026-01-01-triple-b.md"},
		{Rule: 14, Severity: vault.SeverityWarning, Path: "personal/people/ghost", Message: "orphan entity folder \"personal/people/ghost\": entity folder note present but no entries beneath it (stub allowed, flagged)"},
		{Rule: 15, Severity: vault.SeverityWarning, Path: "_system/status/lint/deep/1/2/3/4/x.txt", Message: "path depth 9 exceeds the 8-segment soft cap"},
		{Rule: 16, Severity: vault.SeverityError, Path: "personal/people/jane-doe/misc/2026-01-01-misplaced.md", Message: "entry parent folder must be a 4-digit year folder (got \"misc\")"},
		{Rule: 16, Severity: vault.SeverityError, Path: "personal/people/jane-doe/whatever/2026/2026-01-01-misplaced2.md", Message: "folder above the year must be an event type or a collection category from the taxonomy catalog (got \"whatever\")"},
		{Rule: 17, Severity: vault.SeverityError, Path: "loose.md", Message: "file outside known top-level scopes; allowed root files: todos.md, README.md"},
		{Rule: 18, Severity: vault.SeverityError, Path: "todos.md", Message: "todos.md must exist at the vault root"},
	}
	if len(fs) != len(want) {
		t.Fatalf("got %d findings, want %d:\n%+v", len(fs), len(want), fs)
	}
	for i := range want {
		if !reflect.DeepEqual(fs[i], want[i]) {
			t.Errorf("finding %d mismatch:\n got: %+v\nwant: %+v", i, fs[i], want[i])
		}
	}
}

func TestCounts(t *testing.T) {
	v, err := vault.Scan("../../testdata/violations-fixture")
	if err != nil {
		t.Fatal(err)
	}
	tax, err := taxonomy.Load("../../testdata/violations-fixture")
	if err != nil {
		t.Fatal(err)
	}
	c := New(v, tax)
	Run(c)
	e, w := 0, 0
	for _, f := range c.Findings {
		switch f.Severity {
		case vault.SeverityError:
			e++
		case vault.SeverityWarning:
			w++
		}
	}
	if e != 23 || w != 2 {
		t.Fatalf("counts = %d errors, %d warnings; want 23, 2", e, w)
	}
}

func TestRule16CollectionCategories(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		tax   *taxonomy.Taxonomy
		want  []vault.Finding
	}{
		{
			name: "collection category above year is allowed",
			files: map[string]string{
				"todos.md":                    "x\n",
				"inbox/review.md":             "x\n",
				"work/work.md":                folderNoteMD("collection", "work", "work"),
				"work/emi/emi.md":             folderNoteMD("collection", "emi", "work"),
				"work/emi/journal/journal.md": folderNoteMD("collection", "journal", "work"),
				"work/emi/journal/2026/2026-09-11-ruta-review.md": entryMD("2026-09-11-abcdef123456", "2026-09-11", "meeting"),
			},
			tax:  scopedTax("work", []string{"webdex", "emi"}, map[string]string{"journal": "collection", "people": "entity"}, []string{"meeting"}),
			want: nil,
		},
		{
			name: "entity kind above year is flagged",
			files: map[string]string{
				"todos.md":                             "x\n",
				"inbox/review.md":                      "x\n",
				"work/work.md":                         folderNoteMD("collection", "work", "work"),
				"work/emi/emi.md":                      folderNoteMD("collection", "emi", "work"),
				"work/emi/people/people.md":            folderNoteMD("entity", "people", "work"),
				"work/emi/people/2026/2026-09-11-x.md": entryMD("2026-09-11-abcdef123456", "2026-09-11", "meeting"),
			},
			tax: scopedTax("work", []string{"webdex", "emi"}, map[string]string{"journal": "collection", "people": "entity"}, []string{"meeting"}),
			want: []vault.Finding{
				{Rule: 16, Severity: vault.SeverityError, Path: "work/emi/people/2026/2026-09-11-x.md", Message: `folder above the year must be an event type or a collection category from the taxonomy catalog (got "people")`},
			},
		},
		{
			name: "event type dir above year still allowed",
			files: map[string]string{
				"todos.md":                             "x\n",
				"inbox/review.md":                      "x\n",
				"personal/personal.md":                 folderNoteMD("collection", "personal", "personal"),
				"personal/people/people.md":            folderNoteMD("collection", "people", "personal"),
				"personal/people/jane-doe/jane-doe.md": folderNoteMD("entity", "jane-doe", "personal"),
				"personal/people/jane-doe/meeting/2026/2026-09-11-x.md": entryMD("2026-09-11-abcdef123456", "2026-09-11", "meeting"),
			},
			tax:  scopedTax("personal", nil, map[string]string{"people": "entity"}, []string{"meeting"}),
			want: nil,
		},
		{
			name: "unknown dir above year still flagged",
			files: map[string]string{
				"todos.md":                                               "x\n",
				"inbox/review.md":                                        "x\n",
				"personal/personal.md":                                   folderNoteMD("collection", "personal", "personal"),
				"personal/people/people.md":                              folderNoteMD("collection", "people", "personal"),
				"personal/people/jane-doe/jane-doe.md":                   folderNoteMD("entity", "jane-doe", "personal"),
				"personal/people/jane-doe/whatever/whatever.md":          folderNoteMD("collection", "whatever", "personal"),
				"personal/people/jane-doe/whatever/2026/2026-09-11-x.md": entryMD("2026-09-11-abcdef123456", "2026-09-11", "meeting"),
			},
			tax: scopedTax("personal", nil, map[string]string{"people": "entity"}, []string{"meeting"}),
			want: []vault.Finding{
				{Rule: 16, Severity: vault.SeverityError, Path: "personal/people/jane-doe/whatever/2026/2026-09-11-x.md", Message: `folder above the year must be an event type or a collection category from the taxonomy catalog (got "whatever")`},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := scanTestVault(t, tc.files)
			if err != nil {
				t.Fatal(err)
			}
			c := New(v, tc.tax)
			Run(c)
			got := sortedFindings(c.Findings)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("findings mismatch:\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}

func scopedTax(scope string, companies []string, categories map[string]string, eventTypes []string) *taxonomy.Taxonomy {
	cm := map[string]taxonomy.Category{}
	for name, kind := range categories {
		cm[name] = taxonomy.Category{Kind: kind}
	}
	em := map[string]taxonomy.EventType{}
	for _, e := range eventTypes {
		em[e] = taxonomy.EventType{Template: "_system/templates/" + e + ".md"}
	}
	return &taxonomy.Taxonomy{
		Scopes: map[string]taxonomy.Scope{
			scope: {Companies: companies, Categories: cm},
		},
		EventTypes: em,
	}
}

func entryMD(eventID, date, typ string) string {
	return fmt.Sprintf(`---
event_id: %s
date: %s
type: %s
perspective: personal
privacy: private
entities:
  - work/emi/ruta
origin: generated
source: daily-logs/2026/09/2026-09-10-1432.md
summary: review notes
---
`, eventID, date, typ)
}

func folderNoteMD(kind, name, scope string) string {
	return fmt.Sprintf(`---
type: %s
name: %s
scope: %s
description: ""
keywords: []
aliases: []
related: []
status: active
created: 2026-09-11
children_counts: {}
---
`, kind, name, scope)
}
