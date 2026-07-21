# agent-skills-manager

[中文](./README.md) | [English](./README.en.md)

**Agent Skills 归口与治理中枢**（别名 **`asm`**）。

发现本机 / 项目中的 skills，分类后统一到 `~/.agents/skills/`，在 Claude / Cursor / Grok / Codex / Gemini 等目录用符号链接接线；并支持对**可证明有上游**的 skill 做**只读**更新检查。

- **做强**：多工具盘点、真源归口、冲突处理、doctor、outdated  
- **不做**：替代 [`npx skills`](https://github.com/vercel-labs/skills) 的 marketplace 安装（安装 / 有 lock 的更新仍用它）

> 原名 `agentskills-migrate`。状态目录会从 `~/.local/state/agentskills-migrate` **自动迁移**到 `~/.local/state/agent-skills-manager`。

## 安装

```bash
go build -o bin/agent-skills-manager ./cmd/agent-skills-manager
ln -sf agent-skills-manager bin/asm
./bin/asm version

# 可选：放入 PATH
cp bin/agent-skills-manager bin/asm ~/bin/
```

## 自测

```bash
./scripts/selftest.sh
```

## 快速开始

```bash
asm scan                          # 我有哪些 skills？
asm plan                          # 归口预览（不写盘）
asm apply                         # 迁移 + 符号链接
asm outdated                      # 可溯源 skill 是否有更新（只读）
asm outdated --all                # 连 unknown 也列出
asm doctor
asm undo last
```

## 分类

| 类型 | 含义 |
|------|------|
| **canonical** | 已在 `~/.agents/skills` |
| **user** | 工具全局目录中的用户 skill，可归口 |
| **project** | 仓库内 skill（可 `--project-root`）；不 migrate |
| **vendor** | 内置/市场；Grok 用 `grok inspect` 探针 |

## 更新检查（outdated）

仅检查**有 provenance** 的 skill（**不 pull / 不改 lock**）：

1. skill 目录本身是带 `origin` 的 git 仓库 → 对比 `HEAD` 与 `origin/<default>`  
2. `~/.agents/.skill-lock.json`（`npx skills` 全局 lock）中有 GitHub 条目 → 用 Trees API 对比 `skillFolderHash`  
3. 内置 known source（如 `dws`：指纹识别钉钉官方 skill）→ 对比 GitHub `skills/mono` 下文件的 git blob  
4. `SKILL.md` 声明了 source/version → 标记 unknown  
5. 项目内 `skills-lock.json` → 标记 detected（内容 hash 对比尚未实现）  
6. 其余手动 skill → unknown（默认隐藏，`--all` 可见）

## 常用参数

```text
--json  --all  --yes  --no-probe  --no-project
--project-root PATH   # 可重复
--lang zh|en  --zh  --en
--extra-root PATH
```

环境变量：`ASM_LANG=en`

## 许可证

MIT
