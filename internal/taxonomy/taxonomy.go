// Package taxonomy loads the vault taxonomy registry (_system/taxonomy.yaml,
// ADR 0002) once per run and exposes scope, category, company and event-type
// catalogs to the rules.
package taxonomy

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Category is a single category entry; Kind is entity or collection.
type Category struct {
	Kind string `yaml:"kind"`
}

// Scope groups categories; the work scope additionally carries its companies
// list (company dirs must appear there, ADR 0004 rule 4).
type Scope struct {
	Description string              `yaml:"description"`
	Companies   []string            `yaml:"companies"`
	Categories  map[string]Category `yaml:"categories"`
}

// EventType maps an event type key to its body template path.
type EventType struct {
	Template string `yaml:"template"`
}

// RelationKind describes a typed people relationship: either symmetric
// (`partner`, `sibling`) or carrying the inverse kind (`parent` <-> `child`).
type RelationKind struct {
	Symmetric bool   `yaml:"symmetric"`
	Inverse   string `yaml:"inverse"`
}

// Taxonomy is the parsed registry.
type Taxonomy struct {
	Scopes        map[string]Scope        `yaml:"scopes"`
	EventTypes    map[string]EventType    `yaml:"event_types"`
	RelationKinds map[string]RelationKind `yaml:"relation_kinds"`
}

// Load parses <root>/_system/taxonomy.yaml. A missing or malformed file
// returns an error; the caller decides how to surface it (rule 4 finding).
func Load(root string) (*Taxonomy, error) {
	data, err := os.ReadFile(filepath.Join(root, "_system", "taxonomy.yaml"))
	if err != nil {
		return nil, err
	}
	var t Taxonomy
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	if t.Scopes == nil {
		t.Scopes = map[string]Scope{}
	}
	if t.EventTypes == nil {
		t.EventTypes = map[string]EventType{}
	}
	if t.RelationKinds == nil {
		t.RelationKinds = map[string]RelationKind{}
	}
	for k, s := range t.Scopes {
		if s.Categories == nil {
			s.Categories = map[string]Category{}
			t.Scopes[k] = s
		}
	}
	return &t, nil
}
