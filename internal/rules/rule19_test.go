package rules

import (
	"reflect"
	"testing"

	"github.com/Thaelvyn/secondBrain-linter/internal/taxonomy"
	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

func TestRule19(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		tax   *taxonomy.Taxonomy
		want  []vault.Finding
	}{
		{
			name: "entity, collection and missing/extra keys",
			files: map[string]string{
				"todos.md":                             "x\n",
				"inbox/review.md":                      "x\n",
				"personal/personal.md":                 folderNoteMD("collection", "personal", "personal"),
				"personal/people/people.md":            folderNoteMDCounts("collection", "people", "personal", `{meeting: 3}`),
				"personal/people/jane-doe/jane-doe.md": folderNoteMD("entity", "jane-doe", "personal"),
				"personal/people/jane-doe/facts/2026/2026-01-01-a.md": entryMD("2026-01-01-abcdef123456", "2026-01-01", "fact"),
				"personal/people/jane-doe/facts/2026/2026-01-01-b.md": entryMD("2026-01-01-abcdef123457", "2026-01-01", "fact"),
				"personal/health/health.md":                           "---\ntype: collection\nname: health\nscope: personal\nstatus: active\ncreated: 2026-09-11\n---\n",
				"personal/health/2026/2026-01-01-x.md":                entryMD("2026-01-01-abcdef123458", "2026-01-01", "fact"),
			},
			tax: scopedTax("personal", nil, map[string]string{"people": "entity", "health": "collection"}, []string{"fact", "meeting"}),
			want: []vault.Finding{
				{Rule: 19, Severity: vault.SeverityWarning, Path: "personal/health/health.md", Message: "children_counts is missing; expected {\"2026\": 1}"},
				{Rule: 19, Severity: vault.SeverityWarning, Path: "personal/people/jane-doe/jane-doe.md", Message: "children_counts[\"facts\"]: stored 0, actual 2"},
				{Rule: 19, Severity: vault.SeverityWarning, Path: "personal/people/people.md", Message: "children_counts[\"meeting\"]: stored 3, actual 0"},
			},
		},
		{
			name: "plural type dirs and company/scope skips",
			files: map[string]string{
				"todos.md":                     "x\n",
				"inbox/review.md":              "x\n",
				"work/work.md":                 folderNoteMD("collection", "work", "work"),
				"work/emi/emi.md":              folderNoteMD("collection", "emi", "work"),
				"work/emi/people/people.md":    folderNoteMD("entity", "people", "work"),
				"work/emi/people/ruta/ruta.md": folderNoteMD("entity", "ruta", "work"),
				"work/emi/people/ruta/meetings/2026/2026-09-11-x.md": entryMD("2026-09-11-abcdef123456", "2026-09-11", "meeting"),
			},
			tax: scopedTax("work", []string{"webdex", "emi"}, map[string]string{"people": "entity"}, []string{"meeting"}),
			want: []vault.Finding{
				{Rule: 19, Severity: vault.SeverityWarning, Path: "work/emi/people/ruta/ruta.md", Message: "children_counts[\"meetings\"]: stored 0, actual 1"},
			},
		},
		{
			name: "correct counts produce no findings",
			files: map[string]string{
				"todos.md":                             "x\n",
				"inbox/review.md":                      "x\n",
				"personal/people/people.md":            folderNoteMD("collection", "people", "personal"),
				"personal/people/jane-doe/jane-doe.md": folderNoteMDCounts("entity", "jane-doe", "personal", `{facts: 2}`),
				"personal/people/jane-doe/facts/2026/2026-01-01-a.md": entryMD("2026-01-01-abcdef123456", "2026-01-01", "fact"),
				"personal/people/jane-doe/facts/2026/2026-01-01-b.md": entryMD("2026-01-01-abcdef123457", "2026-01-01", "fact"),
			},
			tax:  scopedTax("personal", nil, map[string]string{"people": "entity"}, []string{"fact"}),
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := byRule(lintVault(t, tc.files, tc.tax), 19)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("R19 findings mismatch:\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}
