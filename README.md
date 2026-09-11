# sblint — secondBrain vault linter

`gopkg.in/yaml.v3`. Module `github.com/Thaelvyn/secondBrain-linter`.

## Usage

```
sblint [flags] <vault-path>
  --json      machine-readable JSON output to stdout
  --report    write timestamped report to <vault>/_system/status/lint/sblint-YYYYmmdd-HHMMSS.txt (console format)
  --version   print version (ldflags-injected, default v0.1.1)
  -h / --help
```

Exit codes: `0` = clean, `1` = violations found, `2` = internal error.

Console format, one line per finding then summary:

```
[error] R1 julien/This is a test.md: filename slug must be lowercase kebab-case ASCII without spaces
[error] R17 julien/This is a test.md: file outside known top-level scopes; allowed root files: todos.md, README.md
2 errors, 0 warnings
```

JSON:

```json
{"vault":"julien","findings":[{"rule":1,"severity":"error","path":"julien/This is a test.md","message":"filename slug must be lowercase kebab-case ASCII without spaces"}],"counts":{"error":1,"warning":0}}
```

## Rules (ADR 0004, rules 1–18)

| # | Severity | Rule |
|---|----------|------|
| 1 | error | Folder and file slugs lowercase, kebab-case, ASCII, no spaces (exempt: dotfiles/dirs, README.md, `_`-prefixed structural names) |
| 2 | error | Entry filenames (and daily-log files) match `{YYYY-MM-DD}-{slug}.md` (folder notes exempt) |
| 3 | error | Every entity folder (dir whose subpaths contain entries) contains its `{folder}/{folder}.md` folder note |
| 4 | error | Category folder names exist in `_system/taxonomy.yaml`; work company dirs in the `companies` list, `_shared` allowed |
| 5 | error | Entry required frontmatter present: event_id, date, type, perspective, privacy, entities, origin, source, summary |
| 6 | error | `type` in event_types catalog; `privacy` in {public, private, secret}; `origin` in {generated, human}; perspective/date/summary non-empty; date valid YYYY-MM-DD |
| 7 | error | Folder-note frontmatter valid: type in {entity, collection}, name, scope, status in {active, dormant}, created |
| 8 | error | `event_id` matches `{YYYY-MM-DD}-{12-hex}` |
| 9 | error | Duplicate `event_id` in ≥2 entries with no mutual related cross-link |
| 10 | error | All wikilinks `[[...]]` resolve to existing files (alias/anchor stripped; target, target+".md", folder note) |
| 11 | error | `related` contains only wikilinks; `entities`/`source` plain paths, no brackets |
| 12 | error | No embeds anywhere (`![[` forbidden) |
| 13 | error | Perspective groups (same event_id, ≥2 entries) full directed mesh via `related` |
| 14 | warning | Orphan entity folder: entity-type note with no entries beneath (stub allowed, flagged) |
| 15 | warning | Path depth > 8 segments (relative to vault root) |
| 16 | error | Entry leaves: parent is 4-digit year dir, above it an event type or collection category from taxonomy, filename starts with frontmatter date |
| 17 | error | Files outside known top-level scopes (root files allowed: todos.md, README.md) |
| 18 | error | `todos.md` and `inbox/review.md` exist |

## Behavior notes / decisions

- **Entry detection**: a markdown file is an entry iff its frontmatter has a non-empty `event_id` (templates under `_system/templates/` have an empty one and are never entries).
- **Folder note detection**: file `{dir}/{basename}.md` at any depth, including structural notes (`_system.md`, `personal.md`, …).
- **R1 exemption**: `_`-prefixed names (`_system`, `_shared`) are ADR-sanctioned structural conventions and are exempt like dotfiles.
- **R3 scoping**: a folder-note is required for every ancestor dir of an entry except 4-digit year dirs, dirs under the structural top-levels, and event-type dirs (`{type}` or `{type}s`).
- **R9 vs R13**: a group with no mutual (bidirectional) link pair is reported once by R9; groups with at least one mutual pair are mesh-checked by R13 (one finding per group listing missing directed links).
- **R10 resolution order**: target → target+".md" → `{target}/{basename}.md` (folder note). Embeds `![[` are skipped by R10 and reported by R12 only.
- **R16** uses the exact `event_types` keys from the taxonomy (`meeting`, `interaction`, …) or `kind: collection` category names (`journal`, `decisions`, …) for the dir above the year.
- **R17** reports stray root files; unknown root *folders* are reported by R4 (no double-reporting).
- Frontmatter with malformed YAML (including duplicate keys) is treated as absent; the file is not analyzed as an entry. `date:`-style values parsed by yaml.v3 as timestamps are normalized to YYYY-MM-DD.
- Rules 19–22 (children_counts drift, source existence, todo reconciliation, body-heading validation) are **not** part of v1 (ADR 0006 adds rule 22; deferred).

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/Thaelvyn/secondBrain-linter/main/scripts/install.sh | bash
# or pinned:
curl -fsSL .../install.sh | bash -s -- --version v0.1.1
```

The script downloads the release binary for `GOOS/GOARCH` (darwin arm64 / linux amd64) into `~/scripts/bin/sblint`. Idempotent.

## Development

```sh
go test ./...    # unit + golden fixture tests (testdata/clean-fixture, testdata/violations-fixture)
go vet ./...
```

Single walk, frontmatter parsed only for `.md`, taxonomy cached per run, maps everywhere (no O(n²)).