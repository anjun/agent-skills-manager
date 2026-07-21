# agent-skills-manager

[中文](./README.md) | [English](./README.en.md)

**Agent skills home & governance hub** (alias **`asm`**).

Discover skills on tool and project roots, classify them, unify under `~/.agents/skills/` with symlinks for Claude / Cursor / Grok / Codex / Gemini, and **check** skills that have proven provenance for updates (read-only).

- **Strengths**: multi-tool inventory, canonical home, conflicts, doctor, outdated  
- **Non-goals**: replacing [`npx skills`](https://github.com/vercel-labs/skills) marketplace install

> Formerly `agentskills-migrate`. State auto-migrates from `~/.local/state/agentskills-migrate` → `~/.local/state/agent-skills-manager`.

## Install

```bash
go build -o bin/agent-skills-manager ./cmd/agent-skills-manager
ln -sf agent-skills-manager bin/asm
./bin/asm version
```

## Self-test

```bash
./scripts/selftest.sh
```

## Quick start

```bash
asm scan
asm plan
asm apply
asm outdated              # check-only
asm outdated --all        # include unknown
asm doctor
asm undo last
```

## Classes

| Class | Meaning |
|-------|---------|
| **canonical** | under `~/.agents/skills` |
| **user** | user skills in tool dirs (migrate candidates) |
| **project** | repo-local (`--project-root`); never migrate |
| **vendor** | bundled/curated; Grok via `grok inspect` |

## outdated

Provenance only (check-only; no pull / no lock rewrite):

1. Skill dir is its own git repo with `origin` → compare HEAD to origin  
2. `~/.agents/.skill-lock.json` (`npx skills` global lock) GitHub entry → compare `skillFolderHash` via Trees API  
3. Frontmatter source/version → unknown  
4. Project `skills-lock.json` → detected (content-hash compare not yet)  
5. Else unknown (hidden unless `--all`)

## License

MIT
