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
asm outdated
asm doctor
asm upgrade --check
asm undo last
```

## 许可证

MIT
