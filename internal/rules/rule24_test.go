package rules

import (
	"reflect"
	"testing"

	"github.com/Thaelvyn/secondBrain-linter/internal/taxonomy"
	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// testRelationKinds mirrors the relation_kinds catalog of ADR 0004 addendum 8.
var testRelationKinds = map[string]taxonomy.RelationKind{
	"partner":      {Symmetric: true},
	"spouse":       {Symmetric: true},
	"ex_spouse":    {Symmetric: true},
	"parent":       {Inverse: "child"},
	"child":        {Inverse: "parent"},
	"sibling":      {Symmetric: true},
	"friend":       {Symmetric: true},
	"acquaintance": {Symmetric: true},
	"colleague":    {Symmetric: true},
	"peer":         {Symmetric: true},
	"manager":      {Inverse: "report"},
	"report":       {Inverse: "manager"},
}

// peopleTax returns a personal-scope taxonomy with the people entity category
// and the relation_kinds catalog.
func peopleTax() *taxonomy.Taxonomy {
	tax := scopedTax("personal", nil, map[string]string{"people": "entity"}, nil)
	tax.RelationKinds = testRelationKinds
	return tax
}

// personNoteMD builds a folder note for a person entity with extra frontmatter
// lines inserted before the closing ---.
func personNoteMD(name, extra string) string {
	return "---\ntype: entity\nname: " + name + "\nscope: personal\n" + extra + "---\n"
}

// personBodyMD is personNoteMD with a body appended after the frontmatter.
func personBodyMD(name, extra, body string) string {
	return personNoteMD(name, extra) + body
}

// relationsSection renders a body `## Relations` section from raw item lines.
func relationsSection(items ...string) string {
	s := "\n## Relations\n\n"
	for _, it := range items {
		s += "- " + it + "\n"
	}
	return s
}

const aliceNote = "personal/people/alice/alice.md"
const bobNote = "personal/people/bob/bob.md"

// personBaseFiles are the ancestor folder notes and support files the R24/R25
// fixtures need.
func personBaseFiles() map[string]string {
	return map[string]string{
		"todos.md":        "x\n",
		"inbox/review.md": "x\n",
	}
}

func TestRule24WellFormedRelations(t *testing.T) {
	files := personBaseFiles()
	files[aliceNote] = personBodyMD("Alice", "met: true\n",
		relationsSection(
			"partner [[personal/people/bob/bob]]",
			"parent [[personal/people/bob/bob]]",
		))
	files[bobNote] = personBodyMD("Bob", "",
		relationsSection("partner [[personal/people/alice/alice]]"))
	if got := byRule(lintVault(t, files, peopleTax()), 24); got != nil {
		t.Fatalf("expected no R24 findings, got: %+v", got)
	}
}

func TestRule24TrailingProseIgnored(t *testing.T) {
	cases := []string{
		"partner [[personal/people/bob/bob]] — estranged",
		"parent [[personal/people/bob/bob]] since 1998",
		"spouse [[personal/people/bob/bob]] (see [[personal/people/bob/bob]])",
	}
	for _, item := range cases {
		t.Run(item, func(t *testing.T) {
			files := personBaseFiles()
			files[aliceNote] = personBodyMD("Alice", "", relationsSection(item))
			files[bobNote] = personNoteMD("Bob", "")
			if got := byRule(lintVault(t, files, peopleTax()), 24); got != nil {
				t.Fatalf("prose after the link must be ignored, got: %+v", got)
			}
		})
	}
}

func TestRule24RelationsErrors(t *testing.T) {
	cases := []struct {
		name string
		item string
		want string
	}{
		{
			name: "malformed missing whitespace",
			item: "partner[[personal/people/bob/bob]]",
			want: `malformed relation "partner[[personal/people/bob/bob]]": expected '<kind> [[<target>]]'`,
		},
		{
			name: "malformed no link",
			item: "partner personal/people/bob/bob",
			want: `malformed relation "partner personal/people/bob/bob": expected '<kind> [[<target>]]'`,
		},
		{
			name: "malformed bare wikilink",
			item: "[[personal/people/bob/bob]]",
			want: `malformed relation "[[personal/people/bob/bob]]": expected '<kind> [[<target>]]'`,
		},
		{
			name: "malformed non-wikilink",
			item: "partner (personal/people/bob/bob)",
			want: `malformed relation "partner (personal/people/bob/bob)": expected '<kind> [[<target>]]'`,
		},
		{
			name: "unknown kind",
			item: "soulmate [[personal/people/bob/bob]]",
			want: `unknown relation kind "soulmate"`,
		},
		{
			name: "unresolvable target",
			item: "partner [[personal/people/ghost/ghost]]",
			want: `relation target "personal/people/ghost/ghost" does not resolve to an existing file`,
		},
		{
			name: "target not a folder note",
			item: "friend [[personal/people/bob/stray]]",
			want: `relation target "personal/people/bob/stray" must resolve to a person folder note (type: entity)`,
		},
		{
			name: "target not entity",
			item: "friend [[personal/health/health]]",
			want: `relation target "personal/health/health" must resolve to a person folder note (type: entity)`,
		},
		{
			name: "self loop",
			item: "partner [[personal/people/alice/alice]]",
			want: `relation "partner [[personal/people/alice/alice]]" is a self-loop`,
		},
		{
			name: "self loop with prose",
			item: "partner [[personal/people/alice/alice]] — me",
			want: `relation "partner [[personal/people/alice/alice]]" is a self-loop`,
		},
		{
			name: "duplicate pair",
			item: "partner [[personal/people/bob/bob]]",
			want: `duplicate relation "partner [[personal/people/bob/bob]]"`,
		},
		{
			name: "duplicate via alias still duplicate",
			item: "partner [[personal/people/bob/bob|Bob]]",
			want: `duplicate relation "partner [[personal/people/bob/bob|Bob]]"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := personBaseFiles()
			items := []string{tc.item}
			switch tc.name {
			case "duplicate pair":
				items = append([]string{"partner [[personal/people/bob/bob]]"}, items...)
			case "duplicate via alias still duplicate":
				items = append([]string{"partner [[personal/people/bob/bob]]"}, items...)
			}
			files[aliceNote] = personBodyMD("Alice", "", relationsSection(items...))
			files[bobNote] = personNoteMD("Bob", "")
			files["personal/people/bob/stray.md"] = "x\n"
			files["personal/health/health.md"] = "---\ntype: collection\nname: health\nscope: personal\n---\n"
			got := byRule(lintVault(t, files, peopleTax()), 24)
			if len(got) != 1 || got[0].Message != tc.want {
				t.Fatalf("R24 findings mismatch:\n got: %+v\nwant message: %s", got, tc.want)
			}
		})
	}
}

func TestRule24IgnoresStaleFrontmatterRelations(t *testing.T) {
	// The frontmatter `relations:` field no longer exists in the schema: a
	// stale field is ignored (not R24's business) in any shape.
	stale := []string{
		"relations:\n  - junk\n",
		`relations: "partner [[personal/people/bob/bob]]"` + "\n",
		"relations:\n  - { kind: partner, target: x }\n",
		"relations: []\n",
	}
	for _, extra := range stale {
		files := personBaseFiles()
		files[aliceNote] = personNoteMD("Alice", extra)
		files[bobNote] = personNoteMD("Bob", "")
		if got := byRule(lintVault(t, files, peopleTax()), 24); got != nil {
			t.Fatalf("stale frontmatter relations must be ignored (%q), got: %+v", extra, got)
		}
	}
}

func TestRule24RelatedHeadingIsNotRelations(t *testing.T) {
	files := personBaseFiles()
	files[aliceNote] = personBodyMD("Alice", "",
		"\n## Related\n\n- not-a-link\n- partner [[personal/people/bob/bob]]\n")
	files[bobNote] = personNoteMD("Bob", "")
	if got := byRule(lintVault(t, files, peopleTax()), 24); got != nil {
		t.Fatalf("`## Related` must not be parsed as Relations, got: %+v", got)
	}
}

func TestRule24RelationsHeadingAndList(t *testing.T) {
	files := personBaseFiles()
	files[aliceNote] = personBodyMD("Alice", "",
		"\n## Relations <!-- optional -->\n\nSome prose.\n\n- partner [[personal/people/bob/bob]]\n\n## Next\n\n- junk\n")
	files[bobNote] = personNoteMD("Bob", "")
	if got := byRule(lintVault(t, files, peopleTax()), 24); got != nil {
		t.Fatalf("heading marker, prose and sections after the next H2 must be handled, got: %+v", got)
	}
}

func TestRule24WrongKindNoteCarriesMetSelf(t *testing.T) {
	files := personBaseFiles()
	files["personal/health/health.md"] = "---\ntype: collection\nname: health\nscope: personal\nmet: true\n---\n"
	files["personal/projects/apollo/apollo.md"] = personNoteMD("Apollo", "self: true\n")
	want := []vault.Finding{
		{Rule: 24, Severity: vault.SeverityError, Path: "personal/health/health.md", Message: "met is only allowed on person folder notes"},
		{Rule: 24, Severity: vault.SeverityError, Path: "personal/projects/apollo/apollo.md", Message: "self is only allowed on person folder notes"},
	}
	got := byRule(lintVault(t, files, peopleTax()), 24)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("R24 findings mismatch:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestRule24MetSelfMustBeBoolean(t *testing.T) {
	files := personBaseFiles()
	files[aliceNote] = personNoteMD("Alice", "met: yes\nself: 1\n")
	want := []vault.Finding{
		{Rule: 24, Severity: vault.SeverityError, Path: aliceNote, Message: "met must be a boolean"},
		{Rule: 24, Severity: vault.SeverityError, Path: aliceNote, Message: "self must be a boolean"},
	}
	got := byRule(lintVault(t, files, peopleTax()), 24)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("R24 findings mismatch:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestRule24SelfOwner(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  []vault.Finding
	}{
		{
			name: "single self note is clean",
			files: map[string]string{
				aliceNote: personNoteMD("Julien", "self: true\n"),
			},
		},
		{
			name: "two self notes same name cross-linked is clean",
			files: map[string]string{
				aliceNote: personNoteMD("Julien", "self: true\nrelated:\n  - \"[[personal/people/bob/bob]]\"\n"),
				bobNote:   personNoteMD("Julien", "self: true\nrelated:\n  - \"[[personal/people/alice/alice]]\"\n"),
			},
		},
		{
			name: "different names are two owners",
			files: map[string]string{
				aliceNote: personNoteMD("Julien", "self: true\nrelated:\n  - \"[[personal/people/bob/bob]]\"\n"),
				bobNote:   personNoteMD("Jeanne", "self: true\nrelated:\n  - \"[[personal/people/alice/alice]]\"\n"),
			},
			want: []vault.Finding{
				{Rule: 24, Severity: vault.SeverityError, Path: "personal/people/bob/bob.md", Message: `self note name "Jeanne" does not match owner name "Julien" (at most one owner)`},
			},
		},
		{
			name: "missing related cross-link",
			files: map[string]string{
				aliceNote: personNoteMD("Julien", "self: true\nrelated: []\n"),
				bobNote:   personNoteMD("Julien", "self: true\nrelated:\n  - \"[[personal/people/alice/alice]]\"\n"),
			},
			want: []vault.Finding{
				{Rule: 24, Severity: vault.SeverityError, Path: "personal/people/alice/alice.md", Message: "self note must list every other self note in related: missing personal/people/bob/bob.md"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := personBaseFiles()
			for k, v := range tc.files {
				files[k] = v
			}
			got := byRule(lintVault(t, files, peopleTax()), 24)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("R24 findings mismatch:\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}

func TestRule24WorkCompanyTree(t *testing.T) {
	tax := scopedTax("work", []string{"emi"}, map[string]string{"people": "entity"}, nil)
	tax.RelationKinds = testRelationKinds
	files := personBaseFiles()
	files["work/work.md"] = folderNoteMD("collection", "work", "work")
	files["work/emi/emi.md"] = folderNoteMD("collection", "emi", "work")
	files["work/emi/people/people.md"] = folderNoteMD("entity", "people", "work")
	files["work/emi/people/rasa/rasa.md"] = personBodyMD("Rasa", "",
		relationsSection("colleague [[work/emi/people/ruta/ruta]]"))
	files["work/emi/people/ruta/ruta.md"] = personBodyMD("Ruta", "",
		relationsSection("colleague [[work/emi/people/rasa/rasa]]"))
	if got := byRule(lintVault(t, files, tax), 24); got != nil {
		t.Fatalf("expected no R24 findings, got: %+v", got)
	}
}

func TestRule24NoRelationKindsNoOp(t *testing.T) {
	files := personBaseFiles()
	files[aliceNote] = personBodyMD("Alice", "met: true\nself: true\n", relationsSection("junk"))
	got := byRule(lintVault(t, files, scopedTax("personal", nil, map[string]string{"people": "entity"}, nil)), 24)
	if got != nil {
		t.Fatalf("R24 must no-op without relation_kinds, got: %+v", got)
	}
}

func TestRelationsBodyItems(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{name: "absent heading", content: "---\nx: 1\n---\n\nBody\n", want: nil},
		{
			name:    "entries until next H2",
			content: "---\nx: 1\n---\n\n## Relations\n\n- partner [[a]]\n* child [[b]]\n\n## Next\n\n- junk\n",
			want:    []string{"partner [[a]]", "child [[b]]"},
		},
		{
			name:    "EOF section",
			content: "---\nx: 1\n---\n\n## Relations\n\n- partner [[a]]",
			want:    []string{"partner [[a]]"},
		},
		{
			name:    "marker suffix",
			content: "---\nx: 1\n---\n\n## Relations <!-- optional -->\n\n- partner [[a]]\n",
			want:    []string{"partner [[a]]"},
		},
		{
			name:    "Related heading is not Relations",
			content: "---\nx: 1\n---\n\n## Related\n\n- [[a]]\n",
			want:    nil,
		},
		{
			name:    "blank lines and prose ignored",
			content: "---\nx: 1\n---\n\n## Relations\n\nSome prose.\n\n- partner [[a]] — prose\n",
			want:    []string{"partner [[a]] — prose"},
		},
		{
			name:    "no frontmatter still scans body",
			content: "## Relations\n\n- partner [[a]]\n",
			want:    []string{"partner [[a]]"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := relationsBodyItems(tc.content)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("relationsBodyItems mismatch:\n got: %v\nwant: %v", got, tc.want)
			}
		})
	}
}

func TestRelationItemRe(t *testing.T) {
	cases := []struct {
		in     string
		kind   string
		target string
		ok     bool
	}{
		{in: "partner [[a/b]]", kind: "partner", target: "a/b", ok: true},
		{in: "ex_spouse [[a/b]] — note", kind: "ex_spouse", target: "a/b", ok: true},
		{in: "partner [[a/b|Alias]]", kind: "partner", target: "a/b|Alias", ok: true},
		{in: "partner [[a/b]] x [[c/d]]", kind: "partner", target: "a/b", ok: true},
		{in: "Partner [[a/b]]", ok: false},
		{in: "partner[[a/b]]", ok: false},
		{in: "partner a/b", ok: false},
		{in: "partner", ok: false},
	}
	for _, tc := range cases {
		m := relationItemRe.FindStringSubmatch(tc.in)
		if !tc.ok {
			if m != nil {
				t.Errorf("relationItemRe(%q) = %v, want no match", tc.in, m)
			}
			continue
		}
		if m == nil || m[1] != tc.kind || m[2] != tc.target {
			t.Errorf("relationItemRe(%q) = %v, want %s %s", tc.in, m, tc.kind, tc.target)
		}
	}
}
