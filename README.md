# agent-skills-manager

[中文](./README.md) | [English](./README.en.md)

**Agent Skills 归口与治理中枢**（别名 **`asm`**）。

发现本机 / 项目中的 skills，分类后统一到 `~/.agents/skills/`，在 Claude / Cursor / Grok / Codex / Gemini 等目录用符号链接接线；并支持对**可证明有上游**的 skill 做**只读**更新检查。

- **做强**：多工具盘点、真源归口、冲突处理、doctor、outdated、工具自更新  
- **不做**：替代 [`npx skills`](https://github.com/vercel-labs/skills) 的 marketplace 安装（安装 / 有 lock 的更新仍用它）

## 安装

**从 Release 安装（macOS Apple Silicon）：**

```bash
# 推荐下 .tar.gz（里面是可执行的 asm）
tar -xzf agent-skills-manager-darwin-arm64.tar.gz
mkdir -p ~/bin && mv asm ~/bin/
export PATH="$HOME/bin:$PATH"
asm version
```

若下载的是无扩展名裸文件 `agent-skills-manager-darwin-arm64`，浏览器会去掉可执行位，Finder 显示成 Document，需：

```bash
chmod +x agent-skills-manager-darwin-arm64
xattr -d com.apple.quarantine agent-skills-manager-darwin-arm64 2>/dev/null || true
mv agent-skills-manager-darwin-arm64 ~/bin/asm
```

**从源码安装：**

```bash
go build -o bin/agent-skills-manager ./cmd/agent-skills-manager
ln -sf agent-skills-manager bin/asm
./bin/asm version
# 可选：cp bin/asm ~/bin/
```

## 自更新（GitHub Releases）

```bash
asm upgrade --check     # 仅检查
asm upgrade             # 从 GitHub Releases 下载并替换当前二进制
asm upgrade -y          # 跳过确认
```

默认仓库：`anjun/agent-skills-manager`（可用环境变量 `ASM_GITHUB_REPO=owner/repo` 覆盖）。  
私有仓库需要 token：`GITHUB_TOKEN` / `GH_TOKEN`，或已 `gh auth login`。

## 发版（GitHub Actions）

| 触发 | 做什么 | 会发版吗？ |
|------|--------|------------|
| push `main` | **CI**：只 `go test` | ❌ 不打包、不发 Release |
| `make release VERSION=x.y.z` / 推 `v*` tag | **Release**：多平台构建 + 上传 | ✅ |

`dist/` 仅 CI 产物，**不要提交**。

```bash
make release VERSION=0.3.2   # 变更摘要 → 打 tag → push → 触发发版
make summary
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
asm upgrade --check               # 检查 asm 自身是否有新版本
asm upgrade                       # 在线更新 asm 自身
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
