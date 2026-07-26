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
# 交互菜单（推荐日常使用，类似 mo）
asm                               # 无参数进入菜单：归口 / 更新 / 体检 / 管理 / 升级

# 日常归口（命令行）
asm sync                          # 重扫 → 预览 → 归口接线（冲突规则同 apply）
asm sync -y                       # 非交互执行（遇冲突需加 --strategy）

# 分步（terraform 式：先看再改）
asm scan                          # 扫描并分类各工具下的 skill（不写盘）
asm plan                          # 预览将如何归口到 ~/.agents/skills 并接线（不写盘）
asm apply                         # 执行归口：迁移内容 + 各工具目录符号链接
asm undo last                     # 回滚最近一次 apply

# 更新 / 卸载 / 健康
asm outdated                      # 检查 skill 是否有上游更新
asm outdated --all                # 含 unknown，并提示如何纳入更新
asm outdated --update             # 检查并尝试自动更新
asm outdated --update -y          # 更新时跳过确认
asm outdated --update dws         # 只更新指定 skill
asm remove write-a-skill          # 全局卸载指定 skill（真源 + 各工具链接）
asm remove foo bar -y             # 多个，跳过确认
asm doctor                        # 健康检查（断链、缺真源、无 provenance、陈旧 lock）
asm upgrade --check               # 检查 asm 自身是否有新版本

# 启用 / 关闭 skill（关闭后对应工具扫描不到）
asm disable grill-me              # 全局关闭（内容挪到 state，拆掉所有工具链接）
asm enable grill-me               # 全局开启并重新接线
asm disable grill-me --tool codex # 仅对 Codex 关闭
asm enable grill-me -t codex      # 仅对 Codex 开启
asm gates                         # 列出关闭状态
```

### `outdated --update` 会做什么

| 来源 | 动作 |
|------|------|
| skill-lock（npx skills） | `npx skills update <names> -g -y` |
| known（`knownsource.Catalog`） | 按目录 `UpdateSpec` 执行（当前 dws：`upgrade -y` + `skill setup mono`） |
| git skill 目录 | `git pull --ff-only` |
| 上游已删除 / 无路径 | 跳过并提示 |

仓库自维护但无 skill-lock / 非 git 的 skill：在 `internal/knownsource` 增加 Catalog 条目（指纹 + 上游路径 + 更新命令）即可接入检查与 `--update`。

### 启用 / 关闭 skill（gates）

| 模式 | 效果 |
|------|------|
| **全局关** | 拆掉各工具链接；真源挪到 `~/.local/state/agent-skills-manager/disabled-skills/<name>` |
| **按工具关** | 只删该工具下的符号链接（如 `~/.codex/skills/<name>`）；真源仍在 agents |
| **plan / sync / apply** | 读取 `skill-gates.json`，**不会**把已关闭的 skill 重新接线 |

状态文件：`~/.local/state/agent-skills-manager/skill-gates.json`

界面语言默认中文；可用 `--en` / `--zh` 或环境变量 `ASM_LANG=en`。

## 许可与商业使用

本项目采用 [MIT License](./LICENSE)（版权 © 2026 [anjun](https://github.com/anjun)）。

个人使用、学习、评估、非营利使用**免费**。  
商业使用（含 SaaS、客户交付、白标、再分发、闭源产品集成、训练数据用途等）在 MIT 下**同样免费允许**，须在分发物中保留版权声明与许可全文。

商业支持、定制开发或合作：在 [Issues](https://github.com/anjun/agent-skills-manager/issues) 开单并注明「Commercial licensing / 商业合作」，或直接联系仓库维护者。

[文档](./README.md) · [Issues](https://github.com/anjun/agent-skills-manager/issues) · [License](./LICENSE)

Made by [anjun](https://github.com/anjun)
