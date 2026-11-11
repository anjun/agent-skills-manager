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
	"scan.sum_unwired":       "  待接线(unwired)   %d  — 真源已有，但部分工具目录缺/错符号链接",
	"scan.sum_project":       "  项目内(project)    %d  — 仓库本地 skill，只展示、不迁移",
	"scan.sum_vendor":        "  内置/市场(vendor)  %d  — 工具自带或缓存，永不改动",
	"scan.sum_total":         "  合计               %d",
	"scan.sec_canonical":     "已归位",
	"scan.sec_canonical_h":   "已在 ~/.agents/skills",
	"scan.sec_user":          "待迁移",
	"scan.sec_user_h":        "将迁入 ~/.agents/skills，并在各工具目录建符号链接",
	"scan.sec_unwired":       "待接线",
	"scan.sec_unwired_h":     "真源已有；在工具目录补符号链接（asm plan / sync）",
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
	"scan.col_missing":       "缺链",
	"scan.col_wrong":         "错误链接",
	"scan.next_plan":         "下一步：asm plan   # 预览将如何改动（不写盘）",
	"scan.next_wire":         "下一步：asm sync   # 为真源 skill 补齐各工具符号链接（或 asm plan）",
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
	"note.fix_wire_link":     "修复 %s 上错误的符号链接/实目录",

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
	"doctor.gate_restore":   "技能 %s 的原目录缺失，不能直接启用；请先恢复技能文件或运行 asm sync 归口现有副本",
	"doctor.gate_orphan":    "技能 %s 已不存在，但关闭记录仍残留 — 清理: asm remove %s -y",
	"doctor.gate_global":    "skill %s 已全局关闭（工具均不可见）— 开启: asm enable %s",
	"doctor.gate_tools":     "skill %s 已对工具关闭: %s — 开启: asm enable %s --tool <name>",
	"doctor.bak_leftover":   "apply 残留备份目录（不会被迁移）— 确认无用后可手动删除: rm -rf %s",

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
	"main.need_tty":           "交互菜单需要终端（TTY）。请直接运行子命令，例如：asm sync、asm doctor。\n",

	// interactive TUI (mo-style menu)
	"tui.tagline":                    "Agent Skills 归口与治理中枢",
	"tui.repo":                       "https://github.com/anjun/agent-skills-manager",
	"tui.banner_update":              "发现新版本 %s，可在「关于与升级」中更新",
	"tui.menu.sync":                  "归口同步",
	"tui.menu.sync_desc":             "重扫、预览并归口接线",
	"tui.menu.outdated":              "检查更新",
	"tui.menu.outdated_desc":         "检查 skill 上游更新并可更新",
	"tui.menu.doctor":                "健康检查",
	"tui.menu.doctor_desc":           "断链、重复副本与 provenance",
	"tui.menu.manage":                "管理 Skill",
	"tui.menu.manage_desc":           "卸载 / 启用 / 关闭 / 查看状态",
	"tui.menu.about":                 "关于与升级",
	"tui.menu.about_desc":            "版本信息与 asm 自更新",
	"tui.footer.home":                "↑↓ 移动  Enter 进入  1-5 快捷  v 版本  q 退出",
	"tui.footer.nav":                 "↑↓ 移动  Enter 确认  b/Esc 返回  q 退出",
	"tui.footer.confirm":             "←→ 或 y/n  选择  Enter 确认  Esc 取消",
	"tui.footer.result":              "Enter / b 返回主菜单  q 退出",
	"tui.footer.loading":             "请稍候…  Esc 可尝试返回（进行中任务仍会跑完）",
	"tui.footer.outdated":            "↑↓ 移动  空格 勾选  a 全选可更新  u 更新所选  t 显示全部  b 返回  q 退出",
	"tui.footer.pick_multi":          "↑↓ 移动  空格 勾选  Enter 确认  输入过滤  Backspace 删字  b 返回",
	"tui.footer.pick_one":            "↑↓ 移动  Enter 选择  输入过滤  b 返回  q 退出",
	"tui.loading.title":              "处理中",
	"tui.loading.plan":               "正在扫描并生成归口计划…",
	"tui.loading.apply":              "正在执行归口同步…",
	"tui.loading.undo":               "正在回滚最近一次 apply…",
	"tui.loading.doctor":             "正在健康检查…",
	"tui.loading.outdated":           "正在检查 skill 更新…",
	"tui.loading.update":             "正在更新所选 skill（可能下载较大文件，请耐心等待）…",
	"tui.loading.skills":             "正在列出 skill…",
	"tui.loading.gates":              "正在读取关闭状态…",
	"tui.loading.remove":             "正在卸载…",
	"tui.loading.disable":            "正在关闭 skill…",
	"tui.loading.enable":             "正在启用 skill…",
	"tui.loading.upgrade_check":      "正在查询 GitHub Releases…",
	"tui.loading.upgrade":            "正在下载并安装 asm…",
	"tui.result.error":               "出错了",
	"tui.more_items":                 "  … 另有 %d 项未显示",
	"tui.more_above":                 "  … 上方还有 %d 项",
	"tui.more_below":                 "  … 下方还有 %d 项",
	"tui.sync.preview_title":         "归口预览",
	"tui.sync.action_apply":          "执行同步",
	"tui.sync.action_more":           "更多…",
	"tui.sync.action_back":           "返回主菜单",
	"tui.sync.more_title":            "归口 · 更多",
	"tui.sync.more_preview":          "仅预览（不写盘）",
	"tui.sync.more_preview_desc":     "重新扫描并展示计划",
	"tui.sync.more_undo":             "回滚最近一次",
	"tui.sync.more_undo_desc":        "undo last",
	"tui.sync.confirm_apply":         "确认执行同步？迁移 %d，链接 %d",
	"tui.sync.confirm_apply_strategy": "确认执行同步？迁移 %d，链接 %d，冲突策略=%s",
	"tui.sync.confirm_undo":          "确认回滚最近一次 apply？",
	"tui.sync.done_title":            "同步完成",
	"tui.sync.done_summary":          "迁移=%d  链接=%d  跳过=%d",
	"tui.sync.tx":                    "事务 ID: %s（可用菜单回滚或 asm undo last）",
	"tui.sync.undo_done":             "回滚完成",
	"tui.sync.strategy_title":        "选择冲突策略",
	"tui.sync.strategy_hint":         "存在同名不同内容的 skill，请选择如何处理：",
	"tui.sync.strategy.keep-agents":  "保留真源（keep-agents）",
	"tui.sync.strategy.keep-newest":  "保留较新（keep-newest）",
	"tui.sync.strategy.skip":         "跳过冲突（skip）",
	"tui.outdated.title":             "Skill 更新",
	"tui.outdated.empty":             "没有可显示的条目",
	"tui.outdated.confirm_update":    "确认更新 %d 个 skill？",
	"tui.outdated.update_done":       "更新完成",
	"tui.outdated.show_all_off":      "当前隐藏 unknown（按 t 显示全部）",
	"tui.outdated.show_all_on":       "当前显示全部含 unknown（按 t 切换）",
	"tui.doctor.hint_sync":           "提示：缺链接/重复副本可回主菜单选「归口同步」",
	"tui.doctor.hint_cleanup":        "提示：[✓] 为可清理项（apply 残留备份 / 断链）；确认后安全删除，无需手动 rm",
	"tui.doctor.summary":             "共 %d 个问题，其中可清理 %d，已勾选 %d",
	"tui.doctor.confirm_cleanup":     "确认删除 %d 个可清理项？（仅 bak 残留目录与断链）",
	"tui.doctor.cleanup_done":        "清理完成",
	"tui.doctor.cleanup_summary":     "已删除 %d，失败 %d",
	"tui.footer.doctor":              "↑↓ 移动  空格 勾选  a 全选可清理  d/Enter 删除  r 重检  b 返回  q 退出",
	"tui.loading.doctor_cleanup":     "正在清理…",
	"tui.manage.title":               "管理 Skill",
	"tui.manage.remove":              "卸载",
	"tui.manage.remove_desc":         "从真源与各工具移除",
	"tui.manage.disable":             "关闭",
	"tui.manage.disable_desc":        "全局或按工具隐藏",
	"tui.manage.enable":              "启用",
	"tui.manage.enable_desc":         "恢复已关闭的 skill",
	"tui.manage.gates":               "查看关闭状态",
	"tui.manage.gates_desc":          "列出 gates",
	"tui.manage.confirm_remove":      "确认卸载：%s ？",
	"tui.manage.confirm_disable":     "确认关闭 %s（范围：%s）？",
	"tui.manage.confirm_enable":      "确认启用 %s ？",
	"tui.manage.remove_done":         "卸载完成",
	"tui.manage.disable_done":        "关闭完成",
	"tui.manage.enable_done":         "启用完成",
	"tui.gate.global":                "全局关",
	"tui.gate.scope_global":          "全局",
	"tui.gate.pick_scope":            "关闭范围 — %s",
	"tui.pick.empty":                 "（列表为空）",
	"tui.pick.filter":                "过滤: %s",
	"tui.about.version":              "版本",
	"tui.about.check":                "检查更新",
	"tui.about.check_desc":           "查询 GitHub Releases",
	"tui.about.upgrade":              "立即升级",
	"tui.about.upgrade_desc":         "下载并安装最新 asm",
	"tui.about.confirm_upgrade":      "确认升级 asm 自身？",
	"tui.confirm.title":              "请确认",
	"tui.confirm.yes":                "是",
	"tui.confirm.no":                 "否",

	// progress (ephemeral status on stderr; TTY only)
	"progress.scanning":        "正在扫描 skill…",
	"progress.probe_grok":      "正在探测 Grok bundled skills…",
	"progress.doctor":          "正在体检…",
	"progress.outdated_begin":  "准备检查 %d 个 skill 更新（可能请求网络，请稍候）…",
	"progress.outdated_check":  "检查更新 %d/%d  %s…",
	"progress.upgrade_check":   "正在查询 GitHub Releases…",
	"progress.upgrade_download": "正在下载并安装更新…",

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

	"gate.need_name":  "请指定 skill 名称，例如: asm disable grill-me --tool codex",
	"gate.need_tool":  "--tool 需要参数，例如: --tool codex",
	"gate.bad_flag":   "未知参数 %s",
	"gate.list_empty": "当前没有关闭的 skill（全部启用）",
	"gate.list_title": "Skill 开关状态",
	"gate.col_name":   "名称",
	"gate.col_global": "全局关",
	"gate.col_tools":  "按工具关",
	"gate.yes":        "是",
	"gate.no":         "否",
	"gate.list_hint":  "\n开启: asm enable <name> [--tool codex]\n关闭: asm disable <name> [--tool codex]",
	"gate.disable_help": `asm disable — 关闭 skill（工具将扫描不到）

用法:
  asm disable <name> [name2...]                 # 全局关闭（从 ~/.agents/skills 挪走并拆掉所有工具链接）
  asm disable <name> --tool codex               # 仅对 codex 关闭
  asm disable <name> -t claude,cursor           # 多个工具

别名: off
说明: sync/apply 不会把已关闭的 skill 重新接线。
`,
	"gate.enable_help": `asm enable — 启用 skill

用法:
  asm enable <name>                             # 全局开启并接到全部工具
  asm enable <name> --tool codex                # 仅对 codex 开启
  asm enable <name> -t claude,grok

别名: on
`,
	"gate.list_help": `asm gates — 列出已关闭的 skill

用法:
  asm gates
  asm gates --json
`,
	"gate.msg.already_global":   "已全局关闭",
	"gate.msg.global_off":       "已全局关闭（所有工具不可见；内容已暂存）",
	"gate.msg.global_redundant": "已全局关闭，无需再按工具关闭；请先 asm enable %s",
	"gate.msg.already_tools":    "指定工具已是关闭状态",
	"gate.msg.tools_off":        "已对工具关闭: %s",
	"gate.msg.global_on":        "已全局开启（已接到全部工具）",
	"gate.msg.ensure_links":     "确保指定工具已接线",
	"gate.msg.tools_on":         "已对工具开启: %s",
	"gate.err.global_need_full": "skill %q 已全局关闭；请先: asm enable %s（不要加 --tool）",
	"plan.note.global_off":      "已全局关闭（asm enable %s）",
	"plan.note.tool_off":        "已对 %s 关闭（asm enable %s --tool %s）",
	"doctor.prov_source":        "SKILL.md 声明了 source（%s）但不在 skill-lock 中，无法对比上游。建议：npx skills add 写入 lock，或保持为独立 git 仓。不需要则: asm remove %s",
	"doctor.prov_version":       "仅有 frontmatter version（%s），无上游 URL，无法检查更新。建议补充 git remote / skill-lock，或不需要则: asm remove %s",
	"doctor.prov_none":          "无 provenance（手工安装）。无法用 asm outdated 检查更新。可选：npx skills add … -g（skill-lock）、git clone 到技能目录、或 knownsource.Catalog。清理: asm remove %s",
	"outdated.msg.git_ok":       "HEAD 与 origin/%s 一致",
	"outdated.msg.git_behind":   "HEAD 落后或分叉于 origin/%s；更新: asm outdated --update %s",
	"outdated.msg.lock_ok":      "skill-lock hash 匹配 %s@%s",
	"outdated.msg.lock_stale":   "上游路径在 %s 上已不存在（删除或改名）；lock 陈旧 — 清理: asm remove %s -y",
	"outdated.msg.lock_changed": "上游目录在 %s@%s 已变更；更新: asm outdated --update %s",
	"outdated.msg.fm_source":    "SKILL.md source=%s 但不在 skill-lock（无法对比）。提示: npx skills add <repo> -g、保持 git clone，或 asm remove %s",
	"outdated.msg.fm_version":   "仅有 frontmatter version（无上游 URL）。提示: 加 skill-lock / git remote，或 asm remove %s",
	"outdated.msg.no_prov":      "无 provenance（手工安装）。提示: npx skills add … -g、git clone 技能目录、knownsource Catalog，或 asm remove %s",
	"outdated.msg.project_lock": "项目 lock: %s；内容 hash 对比尚未实现（仅支持全局 skill-lock / git / known）",
	"outdated.msg.known_ok":     "known 源 %s@%s（%d 个文件一致）",
	"outdated.msg.known_diff":   "known 源 %s@%s: %d/%d 文件不同（如 %s）；%s",

	"main.usage": `agent-skills-manager (asm) %s — Agent Skills 归口与治理中枢

用法：
  asm                         交互菜单（需 TTY；类似 mo）
  asm scan | plan | apply | sync | undo | doctor | outdated | remove | disable | enable | gates | upgrade
  agent-skills-manager <command>

命令：
  (无参数)              打开交互菜单（推荐日常使用）
  scan                  发现并分类 skill
  plan [--out f.json]   预览归口/链接（不写盘）
  apply                 执行迁移与符号链接
  sync                  重扫 → plan → apply（日常归口入口）
  undo [id|last]        回滚
  doctor                健康检查（含 provenance 提示）
  outdated [--update]   检查 skill 更新；--update 尝试自动更新
  remove <name...>      全局卸载指定 skill（别名 rm）
  disable <name> [-t]   关闭 skill（全局或指定工具）
  enable <name> [-t]    启用 skill
  gates                 列出关闭状态
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
	"scan.sum_unwired":      "  unwired    %d  — in agents, but some tool dirs lack/wrong symlinks",
	"scan.sum_project":      "  project    %d  — repo-local skills; inventory only",
	"scan.sum_vendor":       "  vendor     %d  — bundled/curated; never touch",
	"scan.sum_total":        "  total      %d",
	"scan.sec_canonical":    "canonical",
	"scan.sec_canonical_h":  "already under ~/.agents/skills",
	"scan.sec_user":         "user",
	"scan.sec_user_h":       "migrate into ~/.agents/skills + symlinks in tool dirs",
	"scan.sec_unwired":      "unwired",
	"scan.sec_unwired_h":    "in agents; add tool symlinks (asm plan / sync)",
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
	"scan.col_missing":      "MISSING",
	"scan.col_wrong":        "WRONG",
	"scan.next_plan":        "Next: asm plan   # preview moves (no writes)",
	"scan.next_wire":        "Next: asm sync   # wire agents skills into tool dirs (or asm plan)",
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
	"note.fix_wire_link":        "fix wrong link/real dir in %s",

	"doctor.ok":               "OK — no issues found",
	"doctor.count":            "\n%d issue(s)",
	"doctor.broken_no_target": "broken symlink (no target)",
	"doctor.broken_target":    "broken symlink → %s",
	"doctor.dup_reals":        "%d real (non-symlink) user copies — run asm sync or plan/apply",
	"doctor.wire_missing":     "wire root does not exist yet (created on sync/apply if needed)",
	"doctor.agents_missing":   "canonical agents skills dir missing",
	"doctor.lock_orphan":    "skill-lock lists %s but skill not on disk — clean: asm remove %s -y",
	"doctor.lock_orphan_at": "skill-lock lists %s (file %s) but skill not on disk — clean: asm remove %s -y",
	"doctor.gate_restore":   "Skill %s has no canonical content to enable; restore its files or run asm sync to migrate an existing copy first",
	"doctor.gate_orphan":    "Skill %s is missing but its disable record remains — clean: asm remove %s -y",
	"doctor.gate_global":    "skill %s globally disabled — enable: asm enable %s",
	"doctor.gate_tools":     "skill %s disabled for tools: %s — enable: asm enable %s --tool <name>",
	"doctor.bak_leftover":   "leftover apply backup dir (never migrated) — remove if unused: rm -rf %s",

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
	"main.need_tty":     "Interactive menu needs a TTY. Run a subcommand, e.g. asm sync or asm doctor.\n",

	// interactive TUI (mo-style menu)
	"tui.tagline":                     "Agent skills home & governance hub",
	"tui.repo":                        "https://github.com/anjun/agent-skills-manager",
	"tui.banner_update":               "Update %s available — open About & upgrade",
	"tui.menu.sync":                   "Unify & sync",
	"tui.menu.sync_desc":              "Rescan, preview, migrate & wire",
	"tui.menu.outdated":               "Check updates",
	"tui.menu.outdated_desc":          "Check skill upstreams; optional update",
	"tui.menu.doctor":                 "Health check",
	"tui.menu.doctor_desc":            "Broken links, dupes, provenance",
	"tui.menu.manage":                 "Manage skills",
	"tui.menu.manage_desc":            "Remove / enable / disable / gates",
	"tui.menu.about":                  "About & upgrade",
	"tui.menu.about_desc":             "Version and asm self-update",
	"tui.footer.home":                 "↑↓ move  Enter select  1-5 jump  v version  q quit",
	"tui.footer.nav":                  "↑↓ move  Enter confirm  b/Esc back  q quit",
	"tui.footer.confirm":              "←→ or y/n  Enter confirm  Esc cancel",
	"tui.footer.result":               "Enter / b back to menu  q quit",
	"tui.footer.loading":              "Please wait… Esc may return (work may still finish)",
	"tui.footer.outdated":             "↑↓ move  space toggle  a select outdated  u update  t show all  b back  q quit",
	"tui.footer.pick_multi":           "↑↓ move  space toggle  Enter confirm  type to filter  Backspace  b back",
	"tui.footer.pick_one":             "↑↓ move  Enter select  type to filter  b back  q quit",
	"tui.loading.title":               "Working",
	"tui.loading.plan":                "Scanning and building plan…",
	"tui.loading.apply":               "Running unify & sync…",
	"tui.loading.undo":                "Undoing last apply…",
	"tui.loading.doctor":              "Running doctor…",
	"tui.loading.outdated":            "Checking skill updates…",
	"tui.loading.update":              "Updating selected skills (large downloads may take a while)…",
	"tui.loading.skills":              "Listing skills…",
	"tui.loading.gates":               "Loading gate state…",
	"tui.loading.remove":              "Removing…",
	"tui.loading.disable":             "Disabling skill…",
	"tui.loading.enable":              "Enabling skill…",
	"tui.loading.upgrade_check":       "Querying GitHub Releases…",
	"tui.loading.upgrade":             "Downloading and installing asm…",
	"tui.result.error":                "Something went wrong",
	"tui.more_items":                  "  … and %d more",
	"tui.more_above":                  "  … %d more above",
	"tui.more_below":                  "  … %d more below",
	"tui.sync.preview_title":          "Unify preview",
	"tui.sync.action_apply":           "Run sync",
	"tui.sync.action_more":            "More…",
	"tui.sync.action_back":            "Back to menu",
	"tui.sync.more_title":             "Unify · more",
	"tui.sync.more_preview":           "Preview only (no writes)",
	"tui.sync.more_preview_desc":      "Rescan and show plan",
	"tui.sync.more_undo":              "Undo last apply",
	"tui.sync.more_undo_desc":         "asm undo last",
	"tui.sync.confirm_apply":          "Run sync? migrate=%d link=%d",
	"tui.sync.confirm_apply_strategy": "Run sync? migrate=%d link=%d strategy=%s",
	"tui.sync.confirm_undo":           "Undo the last apply?",
	"tui.sync.done_title":             "Sync complete",
	"tui.sync.done_summary":           "migrated=%d linked=%d skipped=%d",
	"tui.sync.tx":                     "tx: %s (menu undo or asm undo last)",
	"tui.sync.undo_done":              "Undo finished",
	"tui.sync.strategy_title":         "Conflict strategy",
	"tui.sync.strategy_hint":          "Same name, different content — choose how to resolve:",
	"tui.sync.strategy.keep-agents":   "Keep agents truth (keep-agents)",
	"tui.sync.strategy.keep-newest":   "Keep newest (keep-newest)",
	"tui.sync.strategy.skip":          "Skip conflicts (skip)",
	"tui.outdated.title":              "Skill updates",
	"tui.outdated.empty":              "Nothing to show",
	"tui.outdated.confirm_update":     "Update %d skill(s)?",
	"tui.outdated.update_done":        "Update finished",
	"tui.outdated.show_all_off":       "Unknown hidden (press t to show all)",
	"tui.outdated.show_all_on":        "Showing all incl. unknown (press t to toggle)",
	"tui.doctor.hint_sync":            "Tip: missing links / dupes → main menu → Unify & sync",
	"tui.doctor.hint_cleanup":         "Tip: [✓] = cleanable (apply bak leftovers / broken links); no manual rm needed",
	"tui.doctor.summary":              "%d issue(s), %d cleanable, %d selected",
	"tui.doctor.confirm_cleanup":      "Delete %d cleanable item(s)? (bak leftovers & broken links only)",
	"tui.doctor.cleanup_done":         "Cleanup finished",
	"tui.doctor.cleanup_summary":      "removed=%d failed=%d",
	"tui.footer.doctor":               "↑↓ move  space toggle  a select cleanable  d/Enter delete  r recheck  b back  q quit",
	"tui.loading.doctor_cleanup":      "Cleaning up…",
	"tui.manage.title":                "Manage skills",
	"tui.manage.remove":               "Remove",
	"tui.manage.remove_desc":          "Uninstall from agents + tools",
	"tui.manage.disable":              "Disable",
	"tui.manage.disable_desc":         "Global or per-tool hide",
	"tui.manage.enable":               "Enable",
	"tui.manage.enable_desc":          "Restore disabled skills",
	"tui.manage.gates":                "List gates",
	"tui.manage.gates_desc":           "Show disable state",
	"tui.manage.confirm_remove":       "Remove: %s ?",
	"tui.manage.confirm_disable":      "Disable %s (scope: %s)?",
	"tui.manage.confirm_enable":       "Enable %s?",
	"tui.manage.remove_done":          "Remove finished",
	"tui.manage.disable_done":         "Disable finished",
	"tui.manage.enable_done":          "Enable finished",
	"tui.gate.global":                 "global off",
	"tui.gate.scope_global":           "global",
	"tui.gate.pick_scope":             "Disable scope — %s",
	"tui.pick.empty":                  "(empty list)",
	"tui.pick.filter":                 "filter: %s",
	"tui.about.version":               "Version",
	"tui.about.check":                 "Check for updates",
	"tui.about.check_desc":            "Query GitHub Releases",
	"tui.about.upgrade":               "Upgrade now",
	"tui.about.upgrade_desc":          "Download and install latest asm",
	"tui.about.confirm_upgrade":       "Upgrade asm itself?",
	"tui.confirm.title":               "Confirm",
	"tui.confirm.yes":                 "Yes",
	"tui.confirm.no":                  "No",

	// progress (ephemeral status on stderr; TTY only)
	"progress.scanning":         "Scanning skills…",
	"progress.probe_grok":       "Probing Grok bundled skills…",
	"progress.doctor":           "Running doctor…",
	"progress.outdated_begin":   "Checking updates for %d skills (may hit the network)…",
	"progress.outdated_check":   "Checking updates %d/%d  %s…",
	"progress.upgrade_check":    "Querying GitHub Releases…",
	"progress.upgrade_download": "Downloading and installing update…",

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

	"gate.need_name":  "specify skill name, e.g. asm disable grill-me --tool codex",
	"gate.need_tool":  "--tool needs a value, e.g. --tool codex",
	"gate.bad_flag":   "unknown flag %s",
	"gate.list_empty": "no disabled skills (all enabled)",
	"gate.list_title": "Skill gates",
	"gate.col_name":   "name",
	"gate.col_global": "global_off",
	"gate.col_tools":  "tools_off",
	"gate.yes":        "yes",
	"gate.no":         "no",
	"gate.list_hint":  "\nenable: asm enable <name> [--tool codex]\ndisable: asm disable <name> [--tool codex]",
	"gate.disable_help": `asm disable — hide a skill from agent tools

Usage:
  asm disable <name> [name2...]           # global off (park agents + remove all tool links)
  asm disable <name> --tool codex         # codex only
  asm disable <name> -t claude,cursor

Alias: off
sync/apply will not re-wire disabled skills.
`,
	"gate.enable_help": `asm enable — show a skill to agent tools again

Usage:
  asm enable <name>                       # global on + wire all tools
  asm enable <name> --tool codex
  asm enable <name> -t claude,grok

Alias: on
`,
	"gate.list_help": `asm gates — list disabled skills

Usage:
  asm gates
  asm gates --json
`,
	"gate.msg.already_global":   "already globally disabled",
	"gate.msg.global_off":       "globally disabled (hidden from all tools; content parked)",
	"gate.msg.global_redundant": "already globally disabled; per-tool disable is redundant until: asm enable %s",
	"gate.msg.already_tools":    "already disabled for requested tools",
	"gate.msg.tools_off":        "disabled for tools: %s",
	"gate.msg.global_on":        "globally enabled (wired to all tools)",
	"gate.msg.ensure_links":     "ensuring links for requested tools",
	"gate.msg.tools_on":         "enabled for tools: %s",
	"gate.err.global_need_full": "skill %q is globally disabled; run: asm enable %s (without --tool)",
	"plan.note.global_off":      "globally disabled (asm enable %s)",
	"plan.note.tool_off":        "disabled for %s (asm enable %s --tool %s)",
	"doctor.prov_source":        "SKILL.md declares source (%s) but not in skill-lock; cannot compare. Tip: npx skills add for lock, or keep as git clone. Or: asm remove %s",
	"doctor.prov_version":       "frontmatter version only (%s); no upstream URL. Tip: skill-lock / git remote, or: asm remove %s",
	"doctor.prov_none":          "no provenance (manual install). Cannot check with asm outdated. Tip: npx skills add … -g, git clone skill dir, knownsource.Catalog, or: asm remove %s",
	"outdated.msg.git_ok":       "HEAD matches origin/%s",
	"outdated.msg.git_behind":   "HEAD behind or diverged from origin/%s; update: asm outdated --update %s",
	"outdated.msg.lock_ok":      "skill-lock hash matches %s@%s",
	"outdated.msg.lock_stale":   "upstream path missing on %s (deleted or renamed); lock entry stale — clean: asm remove %s -y",
	"outdated.msg.lock_changed": "upstream folder changed on %s@%s; update: asm outdated --update %s",
	"outdated.msg.fm_source":    "SKILL.md source=%s but not in skill-lock (cannot compare). Tip: npx skills add <repo> -g, keep as git clone, or asm remove %s",
	"outdated.msg.fm_version":   "frontmatter version only (no upstream URL). Tip: add skill-lock / git remote, or asm remove %s",
	"outdated.msg.no_prov":      "no provenance (manual install). Tip: npx skills add … -g, git clone skill dir, knownsource Catalog, or asm remove %s",
	"outdated.msg.project_lock": "project lock at %s; content-hash compare not implemented yet (global skill-lock / git / known only)",
	"outdated.msg.known_ok":     "known source %s@%s (%d files match)",
	"outdated.msg.known_diff":   "known source %s@%s: %d/%d files differ (e.g. %s); %s",

	"main.usage": `agent-skills-manager (asm) %s — agent skills home & governance hub

Usage:
  asm                         interactive menu (TTY; mo-style)
  asm scan | plan | apply | sync | undo | doctor | outdated | remove | disable | enable | gates | upgrade
  agent-skills-manager <command>

Commands:
  (no args)             open interactive menu (recommended daily use)
  scan                  discover + classify
  plan [--out f.json]   preview migrate/link (no writes)
  apply                 migrate + symlink
  sync                  rescan → plan → apply (daily unify entry)
  undo [id|last]        rollback
  doctor                health check (incl. provenance hints)
  outdated [--update]   check skill updates; --update applies where possible
  remove <name...>      uninstall skill globally (alias: rm)
  disable <name> [-t]   disable skill (global or per-tool)
  enable <name> [-t]    enable skill
  gates                 list disable state
  upgrade [--check]     self-update from GitHub Releases

Flags:
  --json / --all / --update|-u / --yes / --strategy / --only / --exclude
  --extra-root / --project-root / --no-project / --no-probe
  --lang zh|en  (env ASM_LANG)  --zh / --en

Scope: unify under ~/.agents/skills; marketplace install stays with npx skills.
asm upgrade uses GitHub Releases (private repos need GITHUB_TOKEN or gh auth).
`,
}
