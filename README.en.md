# agent-skills-manager

[中文](./README.md) | [English](./README.en.md)

**Agent skills home & governance hub** (alias **`asm`**).

> Releases currently ship **macOS only** — one zip for both Apple Silicon and Intel.

## Install

1. Open [Releases](https://github.com/anjun/agent-skills-manager/releases)  
2. Download **`asm-macos.zip`**  
3. Unzip → double-click **`Install.command`** (right-click → Open if blocked)  
4. Open a **new** terminal: `asm version`

Upgrade later: `asm upgrade`

CLI install (with `gh auth`):

```bash
gh api -H 'Accept: application/vnd.github.raw' \
  repos/anjun/agent-skills-manager/contents/scripts/install.sh | bash
```

## Release

```bash
make release
```

CI publishes a single `asm-macos.zip` (arm64 + amd64 + double-click installer).

## License

MIT
