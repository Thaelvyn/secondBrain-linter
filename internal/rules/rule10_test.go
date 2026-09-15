package rules

import (
	"reflect"
	"testing"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// TestRule10FolderPathIsBroken pins that a bare folder path is a broken
// wikilink for R10 (Obsidian resolves wikilinks to files only), while the
// folder-note file path resolves.
func TestRule10FolderPathIsBroken(t *testing.T) {
	cases := []struct {
		name string
		link string
		want []vault.Finding
	}{
		{
			name: "bare folder path is broken",
			link: "[[work/emi/people/rasa]]",
			want: []vault.Finding{{
				Rule:     10,
				Severity: vault.SeverityError,
				Path:     "notes.md",
				Message:  "wikilink [[work/emi/people/rasa]] does not resolve to an existing file",
			}},
		},
		{
			name: "folder-note file path resolves",
			link: "[[work/emi/people/rasa/rasa]]",
		},
		{
			name: "folder-note file path with alias resolves",
			link: "[[work/emi/people/rasa/rasa|Rasa]]",
		},
		{
			name: "folder-note file path with md suffix resolves",
			link: "[[work/emi/people/rasa/rasa.md]]",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{
				"notes.md":                     "x\n" + tc.link + "\n",
				"work/emi/people/rasa/rasa.md": folderNoteMD("entity", "rasa", "work"),
			}
			got := byRule(lintVault(t, files, emptyTax()), 10)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("R10 findings mismatch:\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}
