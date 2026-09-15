package taxonomy

import (
	"os"
	"path/filepath"
	"testing"
)

const sample = `
scopes:
  personal:
    description: Personal memory tree
    categories:
      people: { kind: entity }
      places: { kind: entity }
  work:
    description: Work memory tree
    companies: [webdex, emi]
    categories:
      people: { kind: entity }
      projects: { kind: entity }
  goals:
    categories: {}
event_types:
  meeting: { template: _system/templates/meeting.md }
  interaction: { template: _system/templates/interaction.md }
relation_kinds:
  partner:      { symmetric: true }
  spouse:       { symmetric: true }
  ex_spouse:    { symmetric: true }
  parent:       { inverse: child }
  child:        { inverse: parent }
  sibling:      { symmetric: true }
  friend:       { symmetric: true }
  acquaintance: { symmetric: true }
  colleague:    { symmetric: true }
  peer:         { symmetric: true }
  manager:      { inverse: report }
  report:       { inverse: manager }
`

func TestLoad(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "_system")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "taxonomy.yaml"), []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	tax, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(tax.Scopes) != 3 {
		t.Fatalf("scopes = %d, want 3", len(tax.Scopes))
	}
	work := tax.Scopes["work"]
	if len(work.Companies) != 2 || work.Companies[0] != "webdex" {
		t.Errorf("companies = %v", work.Companies)
	}
	if work.Categories["people"].Kind != "entity" {
		t.Errorf("people kind = %q", work.Categories["people"].Kind)
	}
	if _, ok := tax.EventTypes["meeting"]; !ok {
		t.Error("event type meeting missing")
	}
	if tax.EventTypes["meeting"].Template != "_system/templates/meeting.md" {
		t.Errorf("template = %q", tax.EventTypes["meeting"].Template)
	}
	if len(tax.RelationKinds) != 12 {
		t.Fatalf("relation_kinds = %d, want 12", len(tax.RelationKinds))
	}
	if !tax.RelationKinds["partner"].Symmetric {
		t.Error("partner must be symmetric")
	}
	if tax.RelationKinds["parent"].Inverse != "child" {
		t.Errorf("parent inverse = %q, want child", tax.RelationKinds["parent"].Inverse)
	}
	if tax.RelationKinds["manager"].Inverse != "report" {
		t.Errorf("manager inverse = %q, want report", tax.RelationKinds["manager"].Inverse)
	}
}

func TestLoadMissing(t *testing.T) {
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("expected error for missing taxonomy.yaml")
	}
}

func TestLoadEmptyCategories(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "_system")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "taxonomy.yaml"), []byte("scopes:\n  goals:\n    description: no-categories-key\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tax, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if tax.Scopes["goals"].Categories == nil {
		t.Fatal("categories map must be non-nil")
	}
}

func TestLoadMalformed(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "_system")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "taxonomy.yaml"), []byte("scopes: [unclosed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("expected error for malformed taxonomy.yaml")
	}
}

func TestLoadMissingMaps(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "_system")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "taxonomy.yaml"), []byte("slug_rules: { lowercase: true }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tax, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if tax.Scopes == nil || tax.EventTypes == nil || tax.RelationKinds == nil {
		t.Fatal("maps must be non-nil after Load")
	}
	if len(tax.RelationKinds) != 0 {
		t.Fatalf("relation_kinds must be empty when absent, got %d", len(tax.RelationKinds))
	}
}
