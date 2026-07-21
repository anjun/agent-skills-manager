package i18n

// Keys are stable English identifiers. Values are UI copy.

var zh = map[string]string{
	// tools / actions / severity
	"tool.agents":       "真源",
	"tool.project":      "项目",
	"tool.extra":        "额外",
	"action.migrate":    "迁移",
	"action.link":       "链接",
	"action.skip":       "跳过",
	"action.conflict":   "冲突",
	"sev.error":         "错误",
	"sev.warn":          "警告",
	"sev.info":          "信息",

	// scan
	"scan.summary_title":     "汇总",
	"scan.sum_canonical":     "  已归位(canonical)  %d  — 已在 ~/.agents/skills，内容真源",
	"scan.sum_user":          "  待迁移(user)       %d  — 各工具全局目录里的用户 skill，可搬回家",
	"scan.sum_project":       "  项目内(project)    %d  — 仓库本地 skill，只展示、不迁移",
	"scan.sum_vendor":        "  内置/市场(vendor)  %d  — 工具自带或缓存，永不改动",
	"scan.sum_total":         "  合计               %d",
	"scan.sec_canonical":     "已归位",
	"scan.sec_canonical_h":   "已在 ~/.agents/skills",
	"scan.sec_user":          "待迁移",
	"scan.sec_user_h":        "将迁入 ~/.agents/skills，并在各工具目录建符号链接",
	"scan.sec_project":       "项目内",
	"scan.sec_project_h":     "仓库本地；永不迁移",
	"scan.sec_vendor":        "内置/市场",
	"scan.sec_vendor_h":      "工具自带或 curated 缓存；永不改动",
	"scan.sec_header":        "── %s（%d）— %s",
	"scan.vendor_collapsed":  "  （已折叠）按工具统计：%s",
	"scan.vendor_tip":        "  提示：加 --all 可列出全部内置/市场 skill",
	"scan.col_name":          "名称",
	"scan.col_tool":          "工具",
	"scan.col_path":          "路径",
	"scan.next_plan":         "下一步：asm plan   # 预览将如何改动（不写盘）",
	"scan.project_note1":     "说明：项目内 skill 会扫描「当前目录所在 git 仓库」。",
	"scan.project_note2":     "      其他仓库请加：--project-root /path/to/repo",

	// plan
	"plan.wrote":             "已写入 %s",
	"plan.col_type":          "操作",
	"plan.col_name":          "名称",
	"plan.col_tool":          "工具",
	"plan.col_note":          "说明",
	"plan.col_source":        "来源",
	"plan.candidates":        "%d 个候选",
	"plan.summary":           "\n汇总：迁移=%d  链接=%d  跳过=%d  冲突=%d",
	"plan.no_change":         "未改动任何文件。确认后执行：agent-skills-manager apply",

	// notes (plan package emits English keys)
	"note.vendor":             "内置/市场",
	"note.project":            "项目内",
	"note.already linked to agents": "已链接到真源",
	"note.move content into ~/.agents/skills": "内容迁入 ~/.agents/skills",
	"note.replace duplicate real copy with symlink": "重复实目录改为符号链接",
	"note.same name, different content — resolve at apply": "同名不同内容，apply 时处理",
	"note.empty conflict":     "空冲突",
	"note.conflict skipped":   "冲突已跳过",
	"note.renamed from conflict": "冲突后重命名",
	"note.ensure_symlink":     "确保在 %s 建立符号链接",

	// doctor
	"doctor.ok":               "正常 — 未发现问题",
	"doctor.count":            "\n共 %d 个问题",
	"doctor.broken_no_target": "损坏的符号链接（无目标）",
	"doctor.broken_target":    "损坏的符号链接 → %s",
	"doctor.dup_reals":        "%d 份用户实目录副本 — 请运行 asm sync 或 plan/apply",
	"doctor.wire_missing":     "接线根目录尚不存在（sync/apply 时按需创建）",
	"doctor.agents_missing":   "真源目录 ~/.agents/skills 不存在",
	"doctor.lock_orphan":    "skill-lock 列出 %s 但磁盘无此 skill — 清理: asm remove %s -y",
	"doctor.lock_orphan_at": "skill-lock 列出 %s（文件 %s）但磁盘无此 skill — 清理: asm remove %s -y",

	// apply
	"apply.need_yes":          "非 TTY 环境拒绝执行 apply，请加 --yes",
	"apply.about":             "\n即将执行：迁移=%d  链接=%d（跳过项未列出）",
	"apply.moves":             "迁移列表：",
	"apply.proceed":           "确认执行？[y/N] ",
	"apply.aborted":           "已取消",
	"apply.step_migrate":      "[%d] 迁移 %s",
	"apply.step_link":         "[%d] 链接 %s（%s）",
	"apply.err_migrate":       "迁移 %s 失败（回滚: agent-skills-manager undo %s）",
	"apply.err_link":          "链接 %s 失败（回滚: agent-skills-manager undo %s）",
	"apply.done":              "\n完成。迁移=%d  链接=%d  跳过=%d\n回滚：agent-skills-manager undo %s",
	"apply.conflict_need":     "技能 %q 存在冲突：请在交互终端运行，或指定 --strategy",
	"apply.conflict_title":    "\n冲突  名称=%s",
	"apply.conflict_cand":     "  [%d] %s  工具=%s  hash=%s",
	"apply.conflict_prompt":   "  (k)保留第 # 个  (r)两者都保留并重命名  (s)跳过  (q)退出\n> ",
	"apply.conflict_abort":    "在冲突 %s 处已取消",
	"apply.unknown_strategy":  "未知冲突策略 %q",
	"apply.no_tx":             "没有可回滚的事务",
	"apply.undo_warn":         "警告: 回滚移动 %s <- %s: %v\n",
	"apply.undo_done":         "已回滚事务 %s",

	// main
	"probe.disabled":          "探针：已关闭（--no-probe）；Grok 使用内置名单兜底",
	"probe.grok_ok":           "探针：Grok inspect 成功，识别 %d 个 bundled skill",
	"probe.grok_fail":         "探针：Grok inspect 失败（%s）；Grok 使用内置名单兜底",

	"main.unknown_cmd":        "未知命令: %s\n",
	"main.error_prefix":       "错误:",
	"outdated.title":          "Skill 更新检查",
	"outdated.col_status":     "状态",
	"outdated.col_name":       "名称",
	"outdated.col_prov":       "来源类型",
	"outdated.col_local":      "本地",
	"outdated.col_remote":     "远端",
	"outdated.col_msg":        "说明",
	"outdated.status_outdated": "可更新",
	"outdated.status_current":  "已最新",
	"outdated.status_unknown":  "未知",
	"outdated.status_error":    "错误",
	"outdated.status_skipped":  "跳过",
	"outdated.summary":         "\n汇总：可更新=%d  已最新=%d  未知=%d  错误=%d  合计=%d",
	"outdated.note":            "默认只检查。加 --update 可尝试自动更新：skill-lock→npx skills；known(Catalog)→UpdateSpec；git skill→git pull。",
	"outdated.update_begin":    "\n── 自动更新 ──",
	"outdated.update_summary":  "\n更新结果：成功=%d  跳过=%d  失败=%d",
	"outdated.help": `asm outdated — 检查（并可选更新）可溯源 skill

用法:
  asm outdated [--all] [--update|-u] [-y] [name ...]

  --all       列出 unknown
  --update    检查后尝试更新 Status=outdated 的 skill
  -y          更新时跳过确认
  name        只更新指定 skill（可多个）

更新路由:
  skill-lock  → npx skills update … -g -y
  known       → knownsource.Catalog 的 UpdateSpec（如 dws CLI）
  git skill   → git pull --ff-only
  上游已删    → 跳过
`,
	"remove.need_name": "请指定要卸载的 skill 名称，例如: asm remove write-a-skill",
	"remove.done":      "卸载完成：删除=%d  失败=%d  更新 lock=%d",
	"remove.help": `asm remove — 全局卸载指定 skill

用法:
  asm remove <name> [name2...] [-y] [--keep-lock] [--project]

  -y            跳过确认
  --keep-lock   不修改 ~/.agents/.skill-lock.json
  --project     同时删除项目内 (project) skill（默认只动全局）

说明:
  删除各工具全局目录与 ~/.agents/skills 下的同名目录/符号链接。
  永不删除 vendor（工具内置）skill。
  别名: rm, uninstall
`,
	"upgrade.current":          "asm 已是最新版本 %s",
	"upgrade.available":        "发现新版本：%s → %s\n  资源：%s\n  仓库：%s",
	"upgrade.confirm":          "是否下载并替换当前二进制？[y/N] ",
	"upgrade.aborted":          "已取消更新",
	"upgrade.done":             "已更新：%s → %s\n重新打开终端或再执行 asm version 确认。",
	"upgrade.check_hint":       "执行 asm upgrade 进行在线更新。",
	"sync.banner":    "── asm sync：重扫 → 预览 → 归口接线 ──",
	"sync.plan_done": "以上为计划；接下来执行 apply（冲突规则与 apply 相同）。",
	"sync.nothing":   "无需迁移或链接，已是最新接线。",
	"sync.bad_flag":  "sync 不支持参数 %s（可用全局 -y / --strategy / --json）",
	"sync.help": `asm sync — 一次完成归口接线（重扫 → plan → apply）

用法:
  asm sync [-y] [--strategy keep-agents|keep-newest|skip]

  -y            跳过确认（非 TTY 必须）
  --strategy    冲突策略（与 apply 相同；非 TTY 遇冲突时需要）

说明:
  等价于先 plan 再 apply。冲突交互与 apply 一致。
  日常重装 Agent 后优先跑这条。
`,

	"main.usage": `agent-skills-manager (asm) %s — Agent Skills 归口与治理中枢

用法：
  asm scan | plan | apply | sync | undo | doctor | outdated | remove | upgrade
  agent-skills-manager <command>

命令：
  scan                  发现并分类 skill
  plan [--out f.json]   预览归口/链接（不写盘）
  apply                 执行迁移与符号链接
  sync                  重扫 → plan → apply（日常归口入口）
  undo [id|last]        回滚
  doctor                健康检查（含 provenance 提示）
  outdated [--update]   检查 skill 更新；--update 尝试自动更新
  remove <name...>      全局卸载指定 skill（别名 rm）
  upgrade [--check]     从 GitHub Releases 更新 asm 自身

参数：
  --json                机器可读 JSON（字段仍为英文）
  --all                 scan 时列出全部内置/市场 skill
  --update, -u          outdated：检查后更新可更新的 skill
  --yes, -y             跳过确认（非 TTY 必须）
  --strategy <name>     非交互冲突策略
  --only / --exclude    限制工具
  --extra-root PATH     额外用户扫描根
  --project-root PATH   项目 skill 根（可重复）
  --no-project          不自动扫当前 git 仓库
  --no-probe            跳过 grok inspect 探针
  --lang zh|en          界面语言（默认中文；环境变量 ASM_LANG）
  --zh / --en

定位：多工具 skill 归口到 ~/.agents/skills；安装/市场上游仍用 npx skills。
asm upgrade 从 GitHub Releases 自更新（私有仓需 GITHUB_TOKEN 或 gh auth）。
`,
}

var en = map[string]string{
	"tool.agents":     "agents",
	"tool.project":    "project",
	"tool.extra":      "extra",
	"action.migrate":  "migrate",
	"action.link":     "link",
	"action.skip":     "skip",
	"action.conflict": "conflict",
	"sev.error":       "error",
	"sev.warn":        "warn",
	"sev.info":        "info",

	"scan.summary_title":    "Summary",
	"scan.sum_canonical":    "  canonical  %d  — already under ~/.agents/skills (truth)",
	"scan.sum_user":         "  user       %d  — user skills in tool dirs; migrate candidates",
	"scan.sum_project":      "  project    %d  — repo-local skills; inventory only",
	"scan.sum_vendor":       "  vendor     %d  — bundled/curated; never touch",
	"scan.sum_total":        "  total      %d",
	"scan.sec_canonical":    "canonical",
	"scan.sec_canonical_h":  "already under ~/.agents/skills",
	"scan.sec_user":         "user",
	"scan.sec_user_h":       "migrate into ~/.agents/skills + symlinks in tool dirs",
	"scan.sec_project":      "project",
	"scan.sec_project_h":    "repo-local; never migrate",
	"scan.sec_vendor":       "vendor",
	"scan.sec_vendor_h":     "bundled / curated; never touch",
	"scan.sec_header":       "── %s (%d) — %s",
	"scan.vendor_collapsed": "  (hidden) by tool: %s",
	"scan.vendor_tip":       "  tip: re-run with --all to list every vendor skill",
	"scan.col_name":         "NAME",
	"scan.col_tool":         "TOOL",
	"scan.col_path":         "PATH",
	"scan.next_plan":        "Next: asm plan   # preview moves (no writes)",
	"scan.project_note1":    "Note: project skills are scanned from the current git repo.",
	"scan.project_note2":    "      Use --project-root /path/to/repo to include another project.",

	"plan.wrote":      "wrote %s",
	"plan.col_type":   "TYPE",
	"plan.col_name":   "NAME",
	"plan.col_tool":   "TOOL",
	"plan.col_note":   "NOTE",
	"plan.col_source": "SOURCE",
	"plan.candidates": "%d candidates",
	"plan.summary":    "\nSummary: migrate=%d  link=%d  skip=%d  conflict=%d",
	"plan.no_change":  "No files changed. Run: agent-skills-manager apply",

	"note.vendor":             "vendor",
	"note.project":            "project",
	"note.already linked to agents": "already linked to agents",
	"note.move content into ~/.agents/skills": "move content into ~/.agents/skills",
	"note.replace duplicate real copy with symlink": "replace duplicate real copy with symlink",
	"note.same name, different content — resolve at apply": "same name, different content — resolve at apply",
	"note.empty conflict":        "empty conflict",
	"note.conflict skipped":      "conflict skipped",
	"note.renamed from conflict": "renamed from conflict",
	"note.ensure_symlink":        "ensure symlink in %s",

	"doctor.ok":               "OK — no issues found",
	"doctor.count":            "\n%d issue(s)",
	"doctor.broken_no_target": "broken symlink (no target)",
	"doctor.broken_target":    "broken symlink → %s",
	"doctor.dup_reals":        "%d real (non-symlink) user copies — run asm sync or plan/apply",
	"doctor.wire_missing":     "wire root does not exist yet (created on sync/apply if needed)",
	"doctor.agents_missing":   "canonical agents skills dir missing",
	"doctor.lock_orphan":    "skill-lock lists %s but skill not on disk — clean: asm remove %s -y",
	"doctor.lock_orphan_at": "skill-lock lists %s (file %s) but skill not on disk — clean: asm remove %s -y",

	"apply.need_yes":         "refusing to apply without --yes when not a TTY",
	"apply.about":            "\nAbout to apply: migrate=%d  link=%d (skips omitted)",
	"apply.moves":            "Moves:",
	"apply.proceed":          "Proceed? [y/N] ",
	"apply.aborted":          "aborted",
	"apply.step_migrate":     "[%d] migrate %s",
	"apply.step_link":        "[%d] link %s (%s)",
	"apply.err_migrate":      "migrate %s failed (undo: agent-skills-manager undo %s)",
	"apply.err_link":         "link %s failed (undo: agent-skills-manager undo %s)",
	"apply.done":             "\nDone. migrated=%d linked=%d skipped=%d\nundo: agent-skills-manager undo %s",
	"apply.conflict_need":    "conflict on %q requires TTY or --strategy",
	"apply.conflict_title":   "\nCONFLICT  name=%s",
	"apply.conflict_cand":    "  [%d] %s  tool=%s  hash=%s",
	"apply.conflict_prompt":  "  (k)eep #  (r)ename both  (s)kip  (q)uit\n> ",
	"apply.conflict_abort":   "aborted at conflict %s",
	"apply.unknown_strategy": "unknown strategy %q",
	"apply.no_tx":            "no transactions",
	"apply.undo_warn":        "warn: undo move %s <- %s: %v\n",
	"apply.undo_done":        "Undo finished for %s",

	"probe.disabled": "Probe: disabled (--no-probe); Grok falls back to built-in denylist",
	"probe.grok_ok":  "Probe: grok inspect OK, %d bundled skills",
	"probe.grok_fail": "Probe: grok inspect failed (%s); Grok falls back to built-in denylist",

	"main.unknown_cmd":  "unknown command: %s\n",
	"main.error_prefix": "error:",
	"outdated.title":    "Skill update check",
	"outdated.col_status": "STATUS",
	"outdated.col_name":   "NAME",
	"outdated.col_prov":   "PROVENANCE",
	"outdated.col_local":  "LOCAL",
	"outdated.col_remote": "REMOTE",
	"outdated.col_msg":    "NOTE",
	"outdated.status_outdated": "outdated",
	"outdated.status_current":  "current",
	"outdated.status_unknown":  "unknown",
	"outdated.status_error":    "error",
	"outdated.status_skipped":  "skipped",
	"outdated.summary": "\nSummary: outdated=%d  current=%d  unknown=%d  error=%d  total=%d",
	"outdated.note":    "Check-only by default. Pass --update to refresh: skill-lock→npx skills; known(Catalog)→UpdateSpec; git skill→git pull.",
	"outdated.update_begin":   "\n── auto-update ──",
	"outdated.update_summary": "\nUpdate result: ok=%d  skipped=%d  failed=%d",
	"outdated.help": `asm outdated — check (and optionally update) skills with provenance

Usage:
  asm outdated [--all] [--update|-u] [-y] [name ...]

  --all       list unknown
  --update    after check, try to update Status=outdated skills
  -y          skip confirmation
  name        only update these skills

Routes:
  skill-lock  → npx skills update … -g -y
  known       → knownsource.Catalog UpdateSpec (e.g. dws CLI)
  git skill   → git pull --ff-only
  deleted     → skip
`,
	"remove.need_name": "specify skill name(s), e.g. asm remove write-a-skill",
	"remove.done":      "remove done: deleted=%d  failed=%d  lock_updated=%d",
	"remove.help": `asm remove — uninstall skill(s) from global roots

Usage:
  asm remove <name> [name2...] [-y] [--keep-lock] [--project]

  -y            skip confirmation
  --keep-lock   do not edit ~/.agents/.skill-lock.json
  --project     also remove project-class skills (default: global only)

Removes matching dirs/symlinks under tool skill roots and ~/.agents/skills.
Never removes vendor skills. Aliases: rm, uninstall
`,
	"upgrade.current":   "asm is up to date (%s)",
	"upgrade.available": "update available: %s → %s\n  asset: %s\n  repo: %s",
	"upgrade.confirm":   "Download and replace the current binary? [y/N] ",
	"upgrade.aborted":   "upgrade cancelled",
	"upgrade.done":      "updated: %s → %s\nre-open the shell or run asm version to confirm.",
	"upgrade.check_hint": "run: asm upgrade",
	"sync.banner":    "── asm sync: rescan → plan → apply ──",
	"sync.plan_done": "Plan above; applying next (same conflict rules as apply).",
	"sync.nothing":   "Nothing to migrate or link; wiring already current.",
	"sync.bad_flag":  "sync does not take %s (use global -y / --strategy / --json)",
	"sync.help": `asm sync — one-shot unify (rescan → plan → apply)

Usage:
  asm sync [-y] [--strategy keep-agents|keep-newest|skip]

  -y            skip confirmation (required non-TTY)
  --strategy    conflict strategy (same as apply)

After reinstalling an agent, prefer this command.
`,

	"main.usage": `agent-skills-manager (asm) %s — agent skills home & governance hub

Usage:
  asm scan | plan | apply | sync | undo | doctor | outdated | remove | upgrade
  agent-skills-manager <command>

Commands:
  scan                  discover + classify
  plan [--out f.json]   preview migrate/link (no writes)
  apply                 migrate + symlink
  sync                  rescan → plan → apply (daily unify entry)
  undo [id|last]        rollback
  doctor                health check (incl. provenance hints)
  outdated [--update]   check skill updates; --update applies where possible
  remove <name...>      uninstall skill globally (alias: rm)
  upgrade [--check]     self-update from GitHub Releases

Flags:
  --json / --all / --update|-u / --yes / --strategy / --only / --exclude
  --extra-root / --project-root / --no-project / --no-probe
  --lang zh|en  (env ASM_LANG)  --zh / --en

Scope: unify under ~/.agents/skills; marketplace install stays with npx skills.
asm upgrade uses GitHub Releases (private repos need GITHUB_TOKEN or gh auth).
`,
}
