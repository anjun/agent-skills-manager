# agent-skills-manager — Design

**Agent skills home & governance hub** (`asm`). Discover skills on known roots, classify them, unify user-owned skills under `~/.agents/skills/` with tool symlinks, doctor health, and **check-only** update status for skills with proven provenance.

Complements [`npx skills`](https://github.com/vercel-labs/skills) (marketplace install/update). This tool is the **local multi-agent skill housekeeper**, not a second registry client.

## Decisions

| # | Topic | Choice |
|---|--------|--------|
| 1 | Scan boundary | Allowlist global roots + project skill dirs (`--project-root` repeatable). No full-disk scan. |
| 2 | Canonical source | `~/.agents/skills/<name>/` is content home; tool dirs get symlinks. Vendor never moved. |
| 3 | Name conflicts | Interactive default; hash merge; `--strategy keep-agents\|keep-newest\|skip`. |
| 4 | Vendor detection | Path globs + **Grok `inspect` probe** + static denylist fallback for other tools. |
| 5 | Project skills | Never migrate; inventory + **outdated check** allowed. |
| 6 | Symlink wire | claude, cursor, grok, codex, gemini user roots. |
| 7 | Stack | Go single binary (`agent-skills-manager` + alias `asm`) |
| 8 | Rollback | Transaction log + `undo` |
| 9 | Product name | **agent-skills-manager** / **asm** |
| 10 | Product scope | **归口 + 治理中枢** — not full marketplace CLI |
| 11 | Skill updates | proven provenance only; `outdated` check + optional `--update` (skill-lock / git / knownsource Catalog) |
| 12 | Tool self-update | GitHub Releases (`asm upgrade`); default repo `anjun/agent-skills-manager`; private needs token |
| 13 | Daily unify entry | `asm sync` = rescan → plan → apply; conflicts same as apply; personal primary tools Claude/Cursor/Grok/Codex |
| 14 | Skill gates | `disable`/`enable` global or per-tool; global parks content under state `disabled-skills/`; plan/sync respect gates |

## Classification

Each discovered skill directory (folder containing `SKILL.md`):

| Class | Meaning | Default action |
|-------|---------|----------------|
| `canonical` | Already under `~/.agents/skills` | keep; may rewire others → it |
| `user` | User-global tool dir, not vendor | **migrate** → agents + symlink |
| `vendor` | Tool/bundled denylist or vendor path | leave |
| `project` | Under a git work tree skill root | leave |
| `conflict` | Same `name`, different content hash | interactive / strategy |

## CLI surface

```text
asm scan                 # discover + classify
asm plan                 # proposed moves/symlinks (no writes)
asm apply [--yes] [--strategy ...]
asm sync                 # rescan → plan → apply (daily entry)
asm undo [id|last]
asm doctor               # health + local provenance / stale-lock hints
asm outdated [--all] [--update]
asm upgrade [--check]    # self-update from GitHub Releases
```

### outdated

- Scope: canonical + user + project (not vendor); de-dupe by content path then skill name.
- Provenance order:
  1. skill dir is its own git repo with `origin` → compare HEAD to origin
  2. `~/.agents/.skill-lock.json` GitHub entry → GitHub Trees API
  3. `knownsource.Catalog` fingerprint (e.g. dws) → release or HEAD tree
  4. frontmatter `source`/`version` → `unknown` (actionable tip)
  5. project `skills-lock.json` nearby → `unknown`
  6. no provenance → `unknown` (hidden unless `--all`)
- `--update`: skill-lock → `npx skills update`; known → Catalog `UpdateSpec`; git → `git pull --ff-only`.

## Layout

```text
~/.agents/skills/<name>/SKILL.md     # truth
~/.claude/skills/<name> -> ../../.agents/skills/<name>
~/.cursor/skills/<name> -> ...
~/.grok/skills/<name>   -> ...       # only if user-owned; vendor entries stay real dirs
~/.codex/skills/<name>  -> ...       # user skills only; NEVER migrate/link over .system/
~/.gemini/skills/<name> -> ...       # official Gemini user root
# Note: Gemini already treats ~/.agents/skills as an alias (precedence over ~/.gemini/skills
# within the same tier). After unify, Gemini may need no symlink at all if it only reads agents —
# still wire ~/.gemini/skills for tools/docs that only look there, and for skills already living under config/skills.
```

### Known user-level skill roots (scan allowlist, v1)

| Tool | Path | Notes |
|------|------|--------|
| Canonical | `~/.agents/skills` | truth home |
| Claude Code | `~/.claude/skills` | user |
| Cursor | `~/.cursor/skills` | user |
| Grok | `~/.grok/skills` | user; `~/.grok/bundled/skills` = vendor |
| Codex | `~/.codex/skills` | user; **`~/.codex/skills/.system`** = vendor (plan, skill-creator, …) |
| Codex (extra) | `~/.codex/.agents/skills`, `~/.codex/vendor_imports/**` | vendor_imports = vendor |
| **Gemini CLI** | **`~/.gemini/skills`** | official user root ([docs](https://geminicli.com/docs/cli/skills/)); also natively aliases **`~/.agents/skills`** (agents 优先于 `.gemini/skills`) |
| Gemini (extra) | `~/.gemini/config/skills` | 部分安装把用户 skill 放这里 → **scan + 可 migrate** |
| Gemini (vendor) | `~/.gemini/config/plugins/**/skills`, `~/.gemini/antigravity*/**` | extension / backup → vendor / ignore |

**Codex caveat:** some builds historically ignored *symlinked* skill trees ([openai/codex#9365](https://github.com/openai/codex/issues/9365)). Symlink first; `doctor` warn; optional `--codex-mode copy|symlink`.

**Gemini note:** 官方已把 `~/.agents/skills` 当 alias 且同层优先于 `~/.gemini/skills`。归口后 Gemini 往往**直接吃 agents 真源**即可；仍接线 `~/.gemini/skills`（及 migrate `config/skills` 里的用户 skill）以覆盖只认 `.gemini` 路径的场景。

State:

```text
~/.local/state/agent-skills-manager/transactions/<id>.json
~/.config/agent-skills-manager/config.toml   # optional overrides
```

## Interaction (frozen)

**UI client: none (option A).** Polish = good tables, confirmations, conflict prompts, doctor — not a GUI/TUI app.

### Flow (terraform-like)

```text
scan  → classify only (no writes)
plan  → proposed actions (no writes; optional --out plan.json)
apply → execute (writes; transaction + undo)
undo  → reverse last/id transaction
doctor → health check (no writes)
```

### Defaults

| Topic | Choice |
|-------|--------|
| plan vs apply | **A** — `apply` re-scans/plans by default; optional `--plan file.json` freezes a plan |
| confirm | **A** — TTY: summary + list of **moves** + `Proceed? [y/N]`; require `--yes` if non-TTY |
| conflict rename | **A** — primary keeps `{name}`; other becomes `{name}@{tool}` |
| conflict timing | Ask at **apply** only; plan marks `CONFLICT` |
| same content hash | Merge without asking |
| non-TTY + no `--strategy` on conflicts | Fail fast |
| step failure | Stop; leave completed steps; print `undo` (no implicit full rollback) |
| v1 non-goals UI | No desktop app, no web dashboard, no full-screen TUI |

### Conflict prompt (TTY)

```text
(k)eep 1  (K)eep 2  (r)ename both  (s)kip  (d)iff  (q)uit
```

### Non-interactive

```bash
apply --yes --strategy skip|keep-agents|keep-newest
```

## Non-goals (v1)

- Full-disk / `find /` scanning
- Replacing `npx skills` marketplace install
- Editing SKILL.md content
- GUI / desktop / local web UI / full-screen TUI
- Windows junction edge-cases beyond basic symlink support (best-effort)
