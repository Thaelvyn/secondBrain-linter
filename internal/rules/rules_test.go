package rules

import (
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
		{Rule: 16, Severity: vault.SeverityError, Path: "personal/people/jane-doe/whatever/2026/2026-01-01-misplaced2.md", Message: "folder above the year must be an event type from the taxonomy catalog (got \"whatever\")"},
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
