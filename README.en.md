# agent-skills-manager

[中文](./README.md) | [English](./README.en.md)

**Agent skills home & governance hub** (alias **`asm`**).

Discover skills on this machine / in projects, classify them, unify under `~/.agents/skills/`, and wire Claude / Cursor / Grok / Codex / Gemini (and similar) with symlinks. For skills with **proven provenance**, check (and optionally apply) upstream updates.

- **In scope**: multi-tool inventory, canonical home, conflict handling, doctor, outdated, self-update, enable/disable gates  
- **Out of scope**: replacing [`npx skills`](https://github.com/vercel-labs/skills) as a marketplace installer  

> Current releases ship **macOS only** — one universal zip (Apple Silicon + Intel).

## Install (simplest)

1. Open [Releases](https://github.com/anjun/agent-skills-manager/releases)  
2. Download **`asm-macos.zip`** (universal)  
3. Unzip → double-click **`Install.command`**  
   - If blocked: right-click → Open  
4. Open a **new** terminal: `asm version`

Upgrade later:

```bash
asm upgrade
```

### CLI install (optional)

With `gh` logged in:

```bash
gh api -H 'Accept: application/vnd.github.raw' \
  repos/anjun/agent-skills-manager/contents/scripts/install.sh | bash
```

Or after clone: `bash scripts/install.sh`

Private repos need `gh auth login` or `GITHUB_TOKEN`.

### From source

```bash
go build -o bin/asm ./cmd/agent-skills-manager
```

## Release

```bash
make release                 # auto patch bump + tag + push → CI builds asm-macos.zip only
make release BUMP=minor
make release VERSION=0.4.0
```

Pushing `main` runs tests only — **no release**.

## Quick start

```bash
# Daily unify (recommended)
asm sync                          # rescan → plan preview → migrate + wire (same conflict rules as apply)
asm sync -y                       # non-interactive (use --strategy on conflicts)

# Step-by-step (terraform-style: look before write)
asm scan                          # discover + classify (no writes)
asm plan                          # preview unify into ~/.agents/skills + tool symlinks (no writes)
asm apply                         # execute: move content + symlinks under tool dirs
asm undo last                     # roll back last apply

# Update / remove / health
asm outdated                      # check whether skills have upstream updates
asm outdated --all                # include unknown + tips on how to get provenance
asm outdated --update             # check then try auto-update
asm outdated --update -y          # skip confirmation on update
asm outdated --update dws         # update one skill only
asm remove write-a-skill          # global uninstall (canonical + tool links)
asm remove foo bar -y             # multiple, skip confirm
asm doctor                        # health (broken links, missing truth, provenance, stale lock)
asm upgrade --check               # check for a newer asm binary

# Enable / disable skills (disabled tools will not see the skill)
asm disable grill-me              # global off (park content under state; remove all tool links)
asm enable grill-me               # global on + re-wire
asm disable grill-me --tool codex # off for Codex only
asm enable grill-me -t codex      # on for Codex only
asm gates                         # list disable state
```

### What `outdated --update` does

| Provenance | Action |
|------------|--------|
| skill-lock (`npx skills`) | `npx skills update <names> -g -y` |
| known (`knownsource.Catalog`) | run catalog `UpdateSpec` (currently dws: `upgrade -y` + `skill setup mono`) |
| git skill directory | `git pull --ff-only` |
| upstream deleted / no path | skip with message |

Self-maintained skills without skill-lock / non-git: add a Catalog entry in `internal/knownsource` (fingerprint + upstream path + update command) to join check + `--update`.

### Enable / disable (gates)

| Mode | Effect |
|------|--------|
| **Global off** | Remove wire links for all tools; move `~/.agents/skills/<name>` → `~/.local/state/agent-skills-manager/disabled-skills/<name>` |
| **Per-tool off** | Remove only that tool’s symlink (e.g. `~/.codex/skills/<name>`); keep agents content |
| **plan / sync / apply** | Respect `skill-gates.json` — will **not** re-wire a disabled skill |

State file: `~/.local/state/agent-skills-manager/skill-gates.json`

UI language: default Chinese; use `--en` / `--zh` or `ASM_LANG=en`.

## License & commercial use

This project is licensed under the [MIT License](./LICENSE) (Copyright © 2026 [anjun](https://github.com/anjun)).

Personal use, learning, evaluation, and non-profit use are **free**.  
**Commercial use** (including SaaS, client delivery, white-label, redistribution, embedding in closed-source products, and training-data use) is **also free under MIT**. You must retain the copyright notice and license text in distributions.

For commercial support, custom development, or partnership: open a [GitHub Issue](https://github.com/anjun/agent-skills-manager/issues) with the subject “Commercial licensing”, or contact the maintainer via the repository.

[Docs](./README.en.md) · [Issues](https://github.com/anjun/agent-skills-manager/issues) · [License](./LICENSE)

Made by [anjun](https://github.com/anjun)
