# agent-skills-manager

[中文](./README.md) | [English](./README.en.md)

**Agent skills home & governance hub** (alias **`asm`**).

Discover skills on tool and project roots, classify them, unify under `~/.agents/skills/` with symlinks for Claude / Cursor / Grok / Codex / Gemini, and **check** skills that have proven provenance for updates (read-only).

- **Strengths**: multi-tool inventory, canonical home, conflicts, doctor, outdated, self-update  
- **Non-goals**: replacing [`npx skills`](https://github.com/vercel-labs/skills) marketplace install

## Install

**From Release (macOS Apple Silicon):**

```bash
tar -xzf agent-skills-manager-darwin-arm64.tar.gz
mkdir -p ~/bin && mv asm ~/bin/
export PATH="$HOME/bin:$PATH"
asm version
```

Bare binary downloads often lose `+x` (Finder shows “Document”); then:

```bash
chmod +x agent-skills-manager-darwin-arm64
xattr -d com.apple.quarantine agent-skills-manager-darwin-arm64 2>/dev/null || true
mv agent-skills-manager-darwin-arm64 ~/bin/asm
```

**From source:**

```bash
go build -o bin/agent-skills-manager ./cmd/agent-skills-manager
ln -sf agent-skills-manager bin/asm
./bin/asm version
```

## Self-update (GitHub Releases)

```bash
asm upgrade --check
asm upgrade
asm upgrade -y
```

Default repo: `anjun/agent-skills-manager` (`ASM_GITHUB_REPO=owner/repo` to override).  
Private repos need `GITHUB_TOKEN` / `GH_TOKEN` or `gh auth login`.

## Releasing (GitHub Actions)

| Trigger | What runs | Release? |
|---------|-----------|----------|
| push `main` | **CI**: `go test` only | no |
| `make release VERSION=x.y.z` / push `v*` tag | **Release**: multi-arch build + upload | yes |

```bash
make release VERSION=0.3.2
make summary
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
asm upgrade --check       # check asm itself
asm upgrade               # self-update from GitHub Releases
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
