# agent-skills-manager

[中文](./README.md) | [English](./README.en.md)

**Agent Skills 归口与治理中枢**（别名 **`asm`**）。

发现本机 / 项目中的 skills，分类后统一到 `~/.agents/skills/`，在 Claude / Cursor / Grok / Codex / Gemini 等目录用符号链接接线；并支持对**可证明有上游**的 skill 做**只读**更新检查。

- **做强**：多工具盘点、真源归口、冲突处理、doctor、outdated、工具自更新  
- **不做**：替代 [`npx skills`](https://github.com/vercel-labs/skills) 的 marketplace 安装  

> 当前 Release **仅提供 macOS 通用包**（Intel + Apple 芯片二合一）。

## 安装（最简单）

1. 打开 [Releases](https://github.com/anjun/agent-skills-manager/releases)  
2. 下载 **`asm-macos.zip`**（Intel / Apple 芯片通用）  
3. 双击解压 → 双击 **`Install.command`**  
   - 若提示无法打开：右键 → 打开  
4. **新开终端**输入：`asm version`

以后升级：

```bash
asm upgrade
```

### 命令行安装（可选）

已登录 `gh` 时：

```bash
gh api -H 'Accept: application/vnd.github.raw' \
  repos/anjun/agent-skills-manager/contents/scripts/install.sh | bash
```

或 clone 后：`bash scripts/install.sh`

私有仓需要 `gh auth login` 或 `GITHUB_TOKEN`。

### 从源码

```bash
go build -o bin/asm ./cmd/agent-skills-manager
```

## 发版

```bash
make release                 # 自动 patch 升版 + 打 tag + push → CI 只打 asm-macos.zip
make release BUMP=minor
make release VERSION=0.4.0
```

push `main` 只跑测试，**不发版**。

## 快速开始

```bash
asm scan
asm plan
asm apply
asm outdated                      # 检查 skill 是否有上游更新
asm outdated --update             # 检查并尝试自动更新
asm outdated --update -y          # 更新时跳过确认
asm outdated --update dws         # 只更新指定 skill
asm remove write-a-skill          # 全局卸载指定 skill（真源 + 各工具链接）
asm remove foo bar -y             # 多个，跳过确认
asm doctor
asm upgrade --check               # 更新 asm 自身
asm undo last
```

### `outdated --update` 会做什么

| 来源 | 动作 |
|------|------|
| skill-lock（npx skills） | `npx skills update <names> -g -y` |
| known（dws） | `dws skill setup --mode mono --target all --yes`（失败则 `dws upgrade -y`） |
| git skill 目录 | `git pull --ff-only` |
| 上游已删除 / 无路径 | 跳过并提示 |

## 许可证

MIT
