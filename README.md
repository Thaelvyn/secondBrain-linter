# sblint — secondBrain vault linter

`gopkg.in/yaml.v3`. Module `github.com/Thaelvyn/secondBrain-linter`.

## Usage

```
sblint [flags] <vault-path>
  --json      machine-readable JSON output to stdout
  --report    write timestamped report to <vault>/_system/status/lint/sblint-YYYYmmdd-HHMMSS.txt (console format)
  --fix       apply best-effort deterministic fixes (R19 children_counts, R3 missing folder notes, R11 related wikilinks), then re-lint and report residuals
  --recount   recompute + rewrite folder-note children_counts only (the R19 fix), print a summary of notes updated; implies fix; idempotent
  --changed   report/fix only files changed vs git (fallback: mtime state file); the scan is still full
  --version   print version (ldflags-injected, default v0.2.3)
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

## Rules (ADR 0004, rules 1–23)

| # | Severity | Rule |
|---|----------|------|
| 1 | error | Folder and file slugs lowercase, kebab-case, ASCII, no spaces (exempt: dotfiles/dirs, README.md, `_`-prefixed structural names) |
| 2 | error | Entry filenames (and daily-log files) match `{YYYY-MM-DD}-{slug}.md` (folder notes and `daily-logs/in/` drop-zone files exempt) |
| 3 | error | Every entity folder (dir whose subpaths contain entries) contains its `{folder}/{folder}.md` folder note |
| 4 | error | Category folder names exist in `_system/taxonomy.yaml`; work company dirs in the `companies` list, `_shared` allowed |
| 5 | error | Entry required frontmatter present: event_id, date, type, perspective, privacy, entities, origin, source, summary |
| 6 | error | `type` in event_types catalog; `privacy` in {public, private, secret}; `origin` in {generated, human}; perspective/date/summary non-empty; date valid YYYY-MM-DD; ADR 0006 enums when present: decision `status` in {proposed, accepted, superseded}, fact/observation `confidence` in {high, medium, low}, decision `supersedes`/`superseded_by` are `[[...]]` wikilinks |
| 7 | error | Folder-note frontmatter valid: type in {entity, collection}, name, scope, status in {active, dormant}, created |
| 8 | error | `event_id` matches `{YYYY-MM-DD}-{12-hex}` |
| 9 | error | Duplicate `event_id` in ≥2 entries with no mutual related cross-link |
| 10 | error | All wikilinks `[[...]]` resolve to existing files (alias/anchor stripped; target, target+".md", folder note) |
| 11 | error | `related` contains only wikilinks; `entities`/`source` plain paths, no brackets; the optional body `## Related` section also contains only wikilinks (`[[full/path]]`) |
| 12 | error | No embeds anywhere (`![[` forbidden) |
| 13 | error | Perspective groups (same event_id, ≥2 entries) full directed mesh via `related` |
| 14 | retired (v0.2.2) | Orphan entity folder warning — retired: stubs are auto-created on first mention (ADR 0001), an empty entity note is by design; the missing-folder-note case is R3. See ADR 0004 addendum 5 |
| 15 | warning | Path depth > 8 segments (relative to vault root) |
| 16 | error | Entry leaves: parent is 4-digit year dir, above it an event type or collection category from taxonomy, filename starts with frontmatter date |
| 17 | error | Files outside known top-level scopes (root files allowed: todos.md, README.md) |
| 18 | error | `todos.md` and `inbox/review.md` exist |
| 19 | warning | Folder-note `children_counts` matches the actual direct-child counts (see scoping + definitions below) |
| 20 | error | Entry frontmatter `source` resolves to an existing vault file (`manual` and empty exempt) |
| 21 | warning | `todos.md` rows reconcile with open inline entry tasks, both directions |
| 22 | error | Entry body contains the required `##` headings for its `type`, read from the vault template (ADR 0006) |
| 23 | error / warning | Each `entities` path with a folder note is inline-linked (`[[...]]`) from the entry body; error for `type: entity` notes, warning for `type: collection` notes (daily-log and `_system` files exempt) |

## Behavior notes / decisions

- **Entry detection**: a markdown file is an entry iff its frontmatter has a non-empty `event_id` (templates under `_system/templates/` have an empty one and are never entries).
- **Folder note detection**: file `{dir}/{basename}.md` at any depth, including structural notes (`_system.md`, `personal.md`, …).
- **R1 exemption**: `_`-prefixed names (`_system`, `_shared`) are ADR-sanctioned structural conventions and are exempt like dotfiles.
- **R3 scoping**: a folder-note is required for every ancestor dir of an entry except 4-digit year dirs, dirs under the structural top-levels, and event-type dirs (`{type}` or `{type}s`).
- **R9 vs R13**: a group with no mutual (bidirectional) link pair is reported once by R9; groups with at least one mutual pair are mesh-checked by R13 (one finding per group listing missing directed links).
- **R10 resolution order**: target → target+".md" → `{target}/{basename}.md` (folder note). Embeds `![[` are skipped by R10 and reported by R12 only.
- **R16** uses the exact `event_types` keys from the taxonomy (`meeting`, `interaction`, …) or `kind: collection` category names (`journal`, `decisions`, …) for the dir above the year.
- **R17** reports stray root files; unknown root *folders* are reported by R4 (no double-reporting).
- **R19 scoping**: only folder notes whose dir is a taxonomy category (kind `entity` or `collection`) or an entity dir (direct child of an entity-kind category) are checked. Scope roots (`personal/`, `work/`, `goals/`, `reference/`), company dirs (`work/emi/`), and structural top-levels (`_system`, `daily-logs`, `calendar`, `inbox`, `attachments`) are skipped — they carry `children_counts` but are not taxonomically countable.
- **R19 entity counts**: for an entity folder note at `E`, keys are the event-type dir names directly under `E` (any pluralization actually present, e.g. `meeting` or `meetings`), value = number of entries (non-empty `event_id`) anywhere beneath `E/{type-dir}/` (all years).
- **R19 collection counts**: for a collection folder note at `C`, keys are the 4-digit year dirs directly under `C`, value = number of entries directly under `C/{year}/`.
- **R19 drift**: one finding per mismatching key (missing key counts as 0 actual vs stored; an extra stored key that no longer exists is a mismatch too). A note lacking `children_counts` gets one finding expecting `{}` or the actual map. Stored year-like keys are compared as strings; the fix renders them quoted (`{"2026": 1}`) because unquoted numeric YAML keys do not round-trip.
- **R20 resolution**: the `source` value is tried as-is relative to the vault root, with a leading `./` stripped, and `<path>.md` when the path has no extension. `manual` (case-insensitive) and empty values are skipped (`/sb-add` entries, ADR 0002 addendum).
- **R21**: dedup key is `(event_id, normalized text)` (trimmed, whitespace collapsed). The `sb-todos` row format is `- [ ] <text> — [[<full entry path>]] · <event_id>`. Only open inline tasks (`- [ ]`) must have a row; closed (`- [x]`) are fine without one. A row whose `[[path]]` does not resolve, or whose `event_id` does not match the resolved file, is a finding. An absent `todos.md` is R18's job; R21 stays silent. A row pointing at a non-existent file is additionally reported by R10 (both findings are intentional).
- **R22**: required headings come from `_system/templates/{type}.md` read at runtime — never hardcoded (ADR 0006: templates are vault data). Every `## <Heading>` line is a heading; a line carrying the inline marker `<!-- optional -->` is optional, everything else is required (compared case-sensitively, trimmed). A required heading whose section contains only `n/a` is accepted (the heading is present). A template missing/unreadable emits one warning per type ("template missing for type …: cannot validate body headings") and skips that type. The template path prefers the `event_types.{type}.template` registration in `_system/taxonomy.yaml`, falling back to `_system/templates/{type}.md`. `## Actions` is required wherever the template marks it required — the template is the contract.
- **R23**: every `entities` frontmatter path that has a folder note must be inline-linked from the entry body. A folder note exists when `<entities-path>` is a dir containing `<basename>.md` (Obsidian folder-note convention) or when the plain file `<entities-path>.md` exists; entities without a note are ignored (stubs are created on demand, ADR 0001). The body is searched for inline wikilinks only (embeds `![[` skipped, as in R10); alias `|…` and anchor `#…` are stripped and the target/resolved path/target+".md" all match, so `[[path|Name]]`, `[[path#anchor]]` and `[[path.md]]` count. The frontmatter `related` field does not count — but a `## Related` body list item does. One finding per unlinked entity: **error** when the folder note is `type: entity` (or its kind is missing/unreadable), **warning** when it is `type: collection`. Rationale: the julien vault's entity notes are overwhelmingly `type: entity` (people, projects), where a missing inline link is a real content defect; collection paths (`personal/health`, `personal/home`, `personal/decisions`) have folder notes too, and flagging them as hard errors would be dishonest — they are legitimate broad-context references. Daily-log files (including `daily-logs/in/`) and `_system/` files are exempt.
- **Fix scope** (`--fix`): (a) R19 — every linted folder note's `children_counts` is rewritten to the actual map (added if missing; the rest of the frontmatter is kept byte-identical); (b) R3 — a missing folder note `{dir}/{dir}.md` is created (type entity/collection per taxonomy, name = title-cased dir, scope, `status: active`, `created: today`, actual `children_counts`); (c) R11 — bare-path `related` items are wrapped in `[[…]]`. Slug renames stay report-only: renaming files is not a content-safe deterministic operation (links, entities and source paths would silently break), so R1 stayed unfixed. All fixes are idempotent — a second `--fix` changes nothing — and never overwrite content outside the three fields above. After fixing, sblint re-scans, re-runs the rules and reports residual findings.
- **`--recount`** is the focused R19 fix: it rewrites `children_counts` only, prints which notes were updated ("recounted N file(s)"), and is idempotent. `--fix --recount` behaves exactly like `--recount` (a subset of `--fix`).
- **`--changed`** reports (and, with `--fix`/`--recount`, fixes) only files changed vs git: `git -C <vault> diff --name-only HEAD` plus `git -C <vault> ls-files --others --exclude-standard` (paths relative to the vault). If git is unavailable or the vault is not a repo, it falls back to an mtime state file at `<vault>/_system/status/lint/.sblint-changed.json` (first run reports everything and records a baseline; later runs report files whose mtime moved past the recorded value). Caveat: context assembly still requires a full scan — `--changed` filters which findings are **reported** (and which files are **fixed**), it does not shorten the scan. Folder-level findings (R3 reports on a dir; R19 reports the folder note) also fire when any changed file lives under that dir. With no changed files the output is clean and the exit code is 0.
- Rules 19–22 and the three fix modes were **shipped in v0.2.0** (ADR 0006 adds rule 22; the ADR 0004 consistency group is now fully implemented). v0.2.1 extends rule 11 to also validate the optional body `## Related` section (wikilinks only). v0.2.2 retires rule 14 (empty-entity-stub warning) per ADR 0004 addendum 5; rule numbering unchanged (R15–R22 still emit). v0.2.3 adds rule 23 (entry body must inline-link each entity with a folder note; ADR 0004), report-only (no `--fix`).
- Frontmatter with malformed YAML (including duplicate keys) is treated as absent; the file is not analyzed as an entry. `date:`-style values parsed by yaml.v3 as timestamps are normalized to YYYY-MM-DD.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/Thaelvyn/secondBrain-linter/main/scripts/install.sh | bash
# or pinned:
curl -fsSL .../install.sh | bash -s -- --version v0.2.3
```

The script downloads the release binary for `GOOS/GOARCH` (darwin arm64 / linux amd64) into `~/scripts/bin/sblint`. Idempotent.

## Development

```sh
go test ./...    # unit + golden fixture tests (testdata/clean-fixture, testdata/violations-fixture)
go vet ./...
```

Single walk, frontmatter parsed only for `.md`, taxonomy cached per run, maps everywhere (no O(n²)). `--changed` invokes the git CLI (when available) instead of walking the vault.