package tui

import (
	"fmt"
	"strings"

	"github.com/anj/agent-skills-manager/internal/doctor"
	"github.com/anj/agent-skills-manager/internal/fsutil"
	"github.com/anj/agent-skills-manager/internal/outdated"
	"github.com/anj/agent-skills-manager/internal/plan"
	"github.com/anj/agent-skills-manager/internal/selfupdate"
	"github.com/anj/agent-skills-manager/internal/skillgate"
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenHome screen = iota
	screenLoading
	screenSyncPreview
	screenSyncMore
	screenStrategy
	screenConfirm
	screenResult
	screenOutdated
	screenDoctor
	screenManage
	screenPickMulti
	screenPickOne
	screenAbout
)

type afterAction int

const (
	afterNone afterAction = iota
	afterSyncApply
	afterUndo
	afterOutdatedUpdate
	afterRemove
	afterDisable
	afterEnable
	afterUpgrade
	afterDoctorCleanup
)

type menuItem struct {
	id   string
	name string // i18n key
	desc string // i18n key
}

type pickItem struct {
	id    string
	label string
}

type model struct {
	deps *Deps

	screen screen
	cursor int
	width  int
	height int

	banner     string // self-update available line
	upgradeRes *selfupdate.Result

	// loading
	loadMsg string

	// shared result / confirm
	resultTitle string
	resultLines []string
	resultErr   string
	confirmMsg  string
	confirmYes  afterAction
	confirmData any

	// sync
	plan *plan.Plan

	// strategy pick
	strategies []string

	// outdated
	outdatedReport *outdated.Result
	showAll        bool
	selected       map[int]bool

	// doctor
	doctorIssues   []doctor.Issue
	doctorSelected map[int]bool // index into doctorIssues

	// manage / pick
	manageCursor int
	pickTitle    string
	pickItems    []pickItem
	pickMulti    bool
	pickSelected map[int]bool
	pickAfter    afterAction
	pickFilter   string
	// after pick multi for disable: need tool scope
	pendingSkill string
	disableScope string // "" | "global" | tool id

	// about cursor
	aboutCursor int
}

type (
	bannerMsg struct {
		text string
		res  *selfupdate.Result
	}
	loadedMsg struct {
		kind    string
		err     error
		payload any
	}
	// kind: plan, doctor, outdated, skills, gates, upgrade_check, upgrade_apply, apply, undo, update, remove, disable, enable
)

func newModel(deps *Deps) model {
	return model{
		deps:         deps,
		screen:       screenHome,
		selected:     map[int]bool{},
		pickSelected: map[int]bool{},
		strategies:   []string{"keep-agents", "keep-newest", "skip"},
	}
}

func (m model) Init() tea.Cmd {
	return m.checkBannerCmd()
}

func (m model) checkBannerCmd() tea.Cmd {
	deps := m.deps
	return func() tea.Msg {
		res, err := deps.upgradeCheck()
		if err != nil || res == nil || res.UpToDate {
			return bannerMsg{}
		}
		tr := deps.Cfg.T()
		return bannerMsg{
			text: tr.S("tui.banner_update", res.Latest),
			res:  res,
		}
	}
}

func (m model) mainMenu() []menuItem {
	return []menuItem{
		{id: "sync", name: "tui.menu.sync", desc: "tui.menu.sync_desc"},
		{id: "outdated", name: "tui.menu.outdated", desc: "tui.menu.outdated_desc"},
		{id: "doctor", name: "tui.menu.doctor", desc: "tui.menu.doctor_desc"},
		{id: "manage", name: "tui.menu.manage", desc: "tui.menu.manage_desc"},
		{id: "about", name: "tui.menu.about", desc: "tui.menu.about_desc"},
	}
}

func (m model) manageMenu() []menuItem {
	return []menuItem{
		{id: "remove", name: "tui.manage.remove", desc: "tui.manage.remove_desc"},
		{id: "disable", name: "tui.manage.disable", desc: "tui.manage.disable_desc"},
		{id: "enable", name: "tui.manage.enable", desc: "tui.manage.enable_desc"},
		{id: "gates", name: "tui.manage.gates", desc: "tui.manage.gates_desc"},
	}
}

func (m model) syncMoreMenu() []menuItem {
	return []menuItem{
		{id: "preview", name: "tui.sync.more_preview", desc: "tui.sync.more_preview_desc"},
		{id: "undo", name: "tui.sync.more_undo", desc: "tui.sync.more_undo_desc"},
	}
}

func (m model) aboutMenu() []menuItem {
	return []menuItem{
		{id: "check", name: "tui.about.check", desc: "tui.about.check_desc"},
		{id: "upgrade", name: "tui.about.upgrade", desc: "tui.about.upgrade_desc"},
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case bannerMsg:
		m.banner = msg.text
		m.upgradeRes = msg.res
		return m, nil

	case loadedMsg:
		return m.handleLoaded(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleLoaded(msg loadedMsg) (tea.Model, tea.Cmd) {
	tr := m.deps.Cfg.T()
	if msg.err != nil {
		m.screen = screenResult
		m.resultTitle = tr.S("tui.result.error")
		m.resultLines = nil
		m.resultErr = msg.err.Error()
		return m, nil
	}

	switch msg.kind {
	case "plan":
		m.plan = msg.payload.(*plan.Plan)
		m.screen = screenSyncPreview
		m.cursor = 0
	case "plan_preview_only":
		m.plan = msg.payload.(*plan.Plan)
		m.screen = screenResult
		m.resultTitle = tr.S("tui.sync.preview_title")
		m.resultLines = formatPlanSummary(m.deps.Cfg, m.plan)
		m.resultErr = ""
	case "apply":
		res := msg.payload.(*applyResultPayload)
		m.screen = screenResult
		m.resultTitle = tr.S("tui.sync.done_title")
		m.resultLines = []string{
			tr.S("tui.sync.done_summary", res.Migrated, res.Linked, res.Skipped),
		}
		if res.TxID != "" {
			m.resultLines = append(m.resultLines, tr.S("tui.sync.tx", res.TxID))
		}
		m.resultErr = ""
	case "undo":
		m.screen = screenResult
		m.resultTitle = tr.S("tui.sync.undo_done")
		m.resultLines = nil
		m.resultErr = ""
	case "doctor":
		issues := msg.payload.([]doctor.Issue)
		m.doctorIssues = issues
		m.doctorSelected = map[int]bool{}
		// Pre-select all cleanable (bak leftovers / broken links)
		for i, is := range issues {
			if is.Cleanable() {
				m.doctorSelected[i] = true
			}
		}
		m.screen = screenDoctor
		m.cursor = 0
		m.resultLines = nil
		m.resultErr = ""
	case "doctor_cleanup":
		res := msg.payload.(*doctor.CleanupResult)
		m.screen = screenResult
		m.resultTitle = tr.S("tui.doctor.cleanup_done")
		m.resultLines = []string{
			tr.S("tui.doctor.cleanup_summary", len(res.Removed), len(res.Failed)),
		}
		for _, p := range res.Removed {
			m.resultLines = append(m.resultLines, "  − "+fsutil.RelHome(p))
		}
		for _, f := range res.Failed {
			m.resultLines = append(m.resultLines, "  ! "+f)
		}
		m.resultErr = ""
	case "outdated":
		m.outdatedReport = msg.payload.(*outdated.Result)
		m.screen = screenOutdated
		m.cursor = 0
		m.selected = map[int]bool{}
		// preselect all outdated
		for i, e := range m.visibleOutdated() {
			if e.Status == outdated.StatusOutdated {
				m.selected[i] = true
			}
		}
	case "update":
		ur := msg.payload.(*outdated.UpdateResult)
		m.screen = screenResult
		m.resultTitle = tr.S("tui.outdated.update_done")
		m.resultLines = []string{
			tr.S("outdated.update_summary", ur.Updated, ur.Skipped, ur.Failed),
		}
		for _, it := range ur.Items {
			mark := "ok"
			if it.Skipped {
				mark = "skip"
			} else if !it.OK {
				mark = "fail"
			}
			m.resultLines = append(m.resultLines, fmt.Sprintf("  [%s] %s: %s", mark, it.Name, it.Message))
		}
		m.resultErr = ""
	case "skills_remove":
		names := msg.payload.([]string)
		m.openPick(tr.S("tui.manage.remove"), names, true, afterRemove)
	case "skills_disable":
		names := msg.payload.([]string)
		m.openPick(tr.S("tui.manage.disable"), names, false, afterDisable)
	case "skills_enable":
		rows := msg.payload.([]gateRow)
		var names []string
		for _, r := range rows {
			label := r.Name
			if r.GlobalOff {
				label += " [" + tr.S("tui.gate.global") + "]"
			} else if len(r.ToolsOff) > 0 {
				label += " [" + strings.Join(r.ToolsOff, ",") + "]"
			}
			names = append(names, r.Name)
			_ = label
		}
		// use labels in pick items
		items := make([]pickItem, 0, len(rows))
		for _, r := range rows {
			label := r.Name
			if r.GlobalOff {
				label += "  (" + tr.S("tui.gate.global") + ")"
			} else if len(r.ToolsOff) > 0 {
				label += "  (" + strings.Join(r.ToolsOff, ",") + ")"
			}
			items = append(items, pickItem{id: r.Name, label: label})
		}
		if len(items) == 0 {
			m.screen = screenResult
			m.resultTitle = tr.S("tui.manage.enable")
			m.resultLines = []string{tr.S("gate.list_empty")}
			m.resultErr = ""
			return m, nil
		}
		m.pickTitle = tr.S("tui.manage.enable")
		m.pickItems = items
		m.pickMulti = false
		m.pickSelected = map[int]bool{}
		m.pickAfter = afterEnable
		m.cursor = 0
		m.pickFilter = ""
		m.screen = screenPickOne
	case "gates":
		rows := msg.payload.([]gateRow)
		m.screen = screenResult
		m.resultTitle = tr.S("gate.list_title")
		if len(rows) == 0 {
			m.resultLines = []string{tr.S("gate.list_empty")}
		} else {
			m.resultLines = nil
			for _, r := range rows {
				g := tr.S("gate.no")
				if r.GlobalOff {
					g = tr.S("gate.yes")
				}
				tools := "-"
				if len(r.ToolsOff) > 0 {
					tools = strings.Join(r.ToolsOff, ",")
				}
				m.resultLines = append(m.resultLines, fmt.Sprintf("  %s  global=%s  tools=%s", r.Name, g, tools))
			}
		}
		m.resultErr = ""
	case "remove":
		res := msg.payload.(*removePayload)
		m.screen = screenResult
		m.resultTitle = tr.S("tui.manage.remove_done")
		m.resultLines = []string{tr.S("remove.done", res.Removed, res.Failed, res.Locks)}
		m.resultErr = ""
	case "disable", "enable":
		res := msg.payload.(*skillgate.Result)
		m.screen = screenResult
		if msg.kind == "disable" {
			m.resultTitle = tr.S("tui.manage.disable_done")
		} else {
			m.resultTitle = tr.S("tui.manage.enable_done")
		}
		m.resultLines = []string{fmt.Sprintf("%s — %s", res.Name, res.Message)}
		m.resultErr = ""
	case "upgrade_check":
		res := msg.payload.(*selfupdate.Result)
		m.upgradeRes = res
		m.screen = screenResult
		m.resultTitle = tr.S("tui.about.check")
		if res.UpToDate {
			m.resultLines = []string{tr.S("upgrade.current", res.Latest)}
		} else {
			m.resultLines = []string{tr.S("upgrade.available", res.Current, res.Latest, res.Asset, res.Repo)}
			m.banner = tr.S("tui.banner_update", res.Latest)
		}
		m.resultErr = ""
	case "upgrade_apply":
		res := msg.payload.(*selfupdate.Result)
		m.screen = screenResult
		m.resultTitle = tr.S("tui.about.upgrade")
		if res.Updated {
			m.resultLines = []string{tr.S("upgrade.done", res.Current, res.Latest)}
		} else if res.UpToDate {
			m.resultLines = []string{tr.S("upgrade.current", res.Latest)}
		} else {
			m.resultLines = []string{tr.S("upgrade.available", res.Current, res.Latest, res.Asset, res.Repo)}
		}
		m.resultErr = ""
	default:
		m.screen = screenHome
	}
	return m, nil
}

type applyResultPayload struct {
	Migrated, Linked, Skipped int
	TxID                      string
}

type removePayload struct {
	Removed, Failed, Locks int
}

func (m *model) openPick(title string, names []string, multi bool, after afterAction) {
	items := make([]pickItem, 0, len(names))
	for _, n := range names {
		items = append(items, pickItem{id: n, label: n})
	}
	m.pickTitle = title
	m.pickItems = items
	m.pickMulti = multi
	m.pickSelected = map[int]bool{}
	m.pickAfter = after
	m.cursor = 0
	m.pickFilter = ""
	if multi {
		m.screen = screenPickMulti
	} else {
		m.screen = screenPickOne
	}
	if len(items) == 0 {
		tr := m.deps.Cfg.T()
		m.screen = screenResult
		m.resultTitle = title
		m.resultLines = []string{tr.S("tui.pick.empty")}
		m.resultErr = ""
	}
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	tr := m.deps.Cfg.T()
	key := msg.String()

	// global quit on most screens
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.screen {
	case screenHome:
		return m.keyHome(key)
	case screenLoading:
		if key == "q" || key == "esc" {
			m.screen = screenHome
			return m, nil
		}
		return m, nil
	case screenSyncPreview:
		return m.keySyncPreview(key)
	case screenSyncMore:
		return m.keySyncMore(key)
	case screenStrategy:
		return m.keyStrategy(key)
	case screenConfirm:
		return m.keyConfirm(key)
	case screenResult:
		if key == "enter" || key == "esc" || key == "b" || key == "q" {
			if key == "q" {
				return m, tea.Quit
			}
			m.screen = screenHome
			m.cursor = 0
			return m, nil
		}
		return m, nil
	case screenOutdated:
		return m.keyOutdated(key)
	case screenDoctor:
		return m.keyDoctor(key)
	case screenManage:
		return m.keyManage(key)
	case screenPickMulti:
		return m.keyPickMulti(key)
	case screenPickOne:
		return m.keyPickOne(key)
	case screenAbout:
		return m.keyAbout(key)
	default:
		_ = tr
		return m, nil
	}
}

func (m model) keyHome(key string) (tea.Model, tea.Cmd) {
	items := m.mainMenu()
	switch key {
	case "q", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(items)-1 {
			m.cursor++
		}
	case "1", "2", "3", "4", "5":
		idx := int(key[0] - '1')
		if idx >= 0 && idx < len(items) {
			m.cursor = idx
			return m.activateHome()
		}
	case "enter":
		return m.activateHome()
	case "v":
		tr := m.deps.Cfg.T()
		m.screen = screenResult
		m.resultTitle = tr.S("tui.about.version")
		m.resultLines = []string{m.deps.Version}
		m.resultErr = ""
	}
	return m, nil
}

func (m model) activateHome() (tea.Model, tea.Cmd) {
	items := m.mainMenu()
	if m.cursor < 0 || m.cursor >= len(items) {
		return m, nil
	}
	tr := m.deps.Cfg.T()
	switch items[m.cursor].id {
	case "sync":
		m.screen = screenLoading
		m.loadMsg = tr.S("tui.loading.plan")
		return m, m.cmdPlan(false)
	case "outdated":
		m.screen = screenLoading
		m.loadMsg = tr.S("tui.loading.outdated")
		return m, m.cmdOutdated()
	case "doctor":
		m.screen = screenLoading
		m.loadMsg = tr.S("tui.loading.doctor")
		return m, m.cmdDoctor()
	case "manage":
		m.screen = screenManage
		m.cursor = 0
	case "about":
		m.screen = screenAbout
		m.aboutCursor = 0
		m.cursor = 0
	}
	return m, nil
}

func (m model) keySyncPreview(key string) (tea.Model, tea.Cmd) {
	// 0=apply 1=more 2=back
	opts := 3
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < opts-1 {
			m.cursor++
		}
	case "esc", "b":
		m.screen = screenHome
		m.cursor = 0
	case "q":
		return m, tea.Quit
	case "enter":
		switch m.cursor {
		case 0: // apply
			if m.plan == nil {
				return m, nil
			}
			if m.plan.Summary.Migrate == 0 && m.plan.Summary.Link == 0 && m.plan.Summary.Conflict == 0 {
				tr := m.deps.Cfg.T()
				m.screen = screenResult
				m.resultTitle = tr.S("tui.sync.preview_title")
				m.resultLines = []string{tr.S("sync.nothing")}
				m.resultErr = ""
				return m, nil
			}
			if m.plan.Summary.Conflict > 0 {
				m.screen = screenStrategy
				m.cursor = 0
				return m, nil
			}
			tr := m.deps.Cfg.T()
			m.confirmMsg = tr.S("tui.sync.confirm_apply", m.plan.Summary.Migrate, m.plan.Summary.Link)
			m.confirmYes = afterSyncApply
			m.confirmData = ""
			m.screen = screenConfirm
			m.cursor = 0
		case 1:
			m.screen = screenSyncMore
			m.cursor = 0
		case 2:
			m.screen = screenHome
			m.cursor = 0
		}
	}
	return m, nil
}

func (m model) keySyncMore(key string) (tea.Model, tea.Cmd) {
	items := m.syncMoreMenu()
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(items)-1 {
			m.cursor++
		}
	case "esc", "b":
		m.screen = screenSyncPreview
		m.cursor = 1
	case "q":
		return m, tea.Quit
	case "enter":
		tr := m.deps.Cfg.T()
		switch items[m.cursor].id {
		case "preview":
			m.screen = screenLoading
			m.loadMsg = tr.S("tui.loading.plan")
			return m, m.cmdPlan(true)
		case "undo":
			m.confirmMsg = tr.S("tui.sync.confirm_undo")
			m.confirmYes = afterUndo
			m.screen = screenConfirm
			m.cursor = 0
		}
	}
	return m, nil
}

func (m model) keyStrategy(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.strategies)-1 {
			m.cursor++
		}
	case "esc", "b":
		m.screen = screenSyncPreview
		m.cursor = 0
	case "q":
		return m, tea.Quit
	case "enter":
		strategy := m.strategies[m.cursor]
		tr := m.deps.Cfg.T()
		m.confirmMsg = tr.S("tui.sync.confirm_apply_strategy",
			m.plan.Summary.Migrate, m.plan.Summary.Link, strategy)
		m.confirmYes = afterSyncApply
		m.confirmData = strategy
		m.screen = screenConfirm
		m.cursor = 0
	}
	return m, nil
}

func (m model) keyConfirm(key string) (tea.Model, tea.Cmd) {
	// 0=yes 1=no
	switch key {
	case "up", "k", "left", "h":
		m.cursor = 0
	case "down", "j", "right", "l":
		m.cursor = 1
	case "esc", "n":
		m.cursor = 1
		return m.confirmChoose()
	case "y":
		m.cursor = 0
		return m.confirmChoose()
	case "enter":
		return m.confirmChoose()
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m model) confirmChoose() (tea.Model, tea.Cmd) {
	tr := m.deps.Cfg.T()
	if m.cursor != 0 {
		// no
		switch m.confirmYes {
		case afterSyncApply:
			m.screen = screenSyncPreview
			m.cursor = 0
		case afterUndo:
			m.screen = screenSyncMore
			m.cursor = 1
		case afterOutdatedUpdate:
			m.screen = screenOutdated
		case afterRemove, afterDisable, afterEnable:
			m.screen = screenManage
			m.cursor = 0
		case afterUpgrade:
			m.screen = screenAbout
			m.cursor = 0
		case afterDoctorCleanup:
			m.screen = screenDoctor
			m.cursor = 0
		default:
			m.screen = screenHome
			m.cursor = 0
		}
		return m, nil
	}
	// yes
	switch m.confirmYes {
	case afterSyncApply:
		strategy, _ := m.confirmData.(string)
		m.screen = screenLoading
		m.loadMsg = tr.S("tui.loading.apply")
		return m, m.cmdApply(strategy)
	case afterUndo:
		m.screen = screenLoading
		m.loadMsg = tr.S("tui.loading.undo")
		return m, m.cmdUndo()
	case afterOutdatedUpdate:
		names, _ := m.confirmData.([]string)
		m.screen = screenLoading
		m.loadMsg = tr.S("tui.loading.update")
		return m, m.cmdUpdate(names)
	case afterRemove:
		names, _ := m.confirmData.([]string)
		m.screen = screenLoading
		m.loadMsg = tr.S("tui.loading.remove")
		return m, m.cmdRemove(names)
	case afterDisable:
		data, _ := m.confirmData.(disableData)
		m.screen = screenLoading
		m.loadMsg = tr.S("tui.loading.disable")
		return m, m.cmdDisable(data)
	case afterEnable:
		data, _ := m.confirmData.(disableData)
		m.screen = screenLoading
		m.loadMsg = tr.S("tui.loading.enable")
		return m, m.cmdEnable(data)
	case afterUpgrade:
		m.screen = screenLoading
		m.loadMsg = tr.S("tui.loading.upgrade")
		return m, m.cmdUpgradeApply()
	case afterDoctorCleanup:
		items, _ := m.confirmData.([]doctor.Issue)
		m.screen = screenLoading
		m.loadMsg = tr.S("tui.loading.doctor_cleanup")
		return m, m.cmdDoctorCleanup(items)
	}
	m.screen = screenHome
	return m, nil
}

func (m model) keyDoctor(key string) (tea.Model, tea.Cmd) {
	n := len(m.doctorIssues)
	// when empty: only back
	if n == 0 {
		switch key {
		case "enter", "esc", "b":
			m.screen = screenHome
			m.cursor = 0
		case "q":
			return m, tea.Quit
		}
		return m, nil
	}
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < n-1 {
			m.cursor++
		}
	case " ":
		if m.doctorIssues[m.cursor].Cleanable() {
			m.doctorSelected[m.cursor] = !m.doctorSelected[m.cursor]
		}
	case "a":
		// toggle all cleanable
		allOn := true
		for i, is := range m.doctorIssues {
			if is.Cleanable() && !m.doctorSelected[i] {
				allOn = false
				break
			}
		}
		for i, is := range m.doctorIssues {
			if is.Cleanable() {
				m.doctorSelected[i] = !allOn
			}
		}
	case "d", "enter":
		var items []doctor.Issue
		for i, is := range m.doctorIssues {
			if m.doctorSelected[i] && is.Cleanable() {
				items = append(items, is)
			}
		}
		if len(items) == 0 {
			// enter with nothing selected → back home (old behavior convenience)
			if key == "enter" {
				m.screen = screenHome
				m.cursor = 2
			}
			return m, nil
		}
		tr := m.deps.Cfg.T()
		m.confirmMsg = tr.S("tui.doctor.confirm_cleanup", len(items))
		m.confirmYes = afterDoctorCleanup
		m.confirmData = items
		m.screen = screenConfirm
		m.cursor = 0
	case "r":
		// re-run doctor
		tr := m.deps.Cfg.T()
		m.screen = screenLoading
		m.loadMsg = tr.S("tui.loading.doctor")
		return m, m.cmdDoctor()
	case "esc", "b":
		m.screen = screenHome
		m.cursor = 2
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

type disableData struct {
	Name  string
	Tools []string
}

func (m model) keyOutdated(key string) (tea.Model, tea.Cmd) {
	vis := m.visibleOutdated()
	// last two options: update selected, toggle all, back — actually items then actions
	// layout: list of skills + footer actions via keys
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(vis)-1 {
			m.cursor++
		}
	case " ":
		if len(vis) > 0 {
			m.selected[m.cursor] = !m.selected[m.cursor]
		}
	case "a":
		all := true
		for i, e := range vis {
			if e.Status == outdated.StatusOutdated && !m.selected[i] {
				all = false
				break
			}
		}
		for i, e := range vis {
			if e.Status == outdated.StatusOutdated {
				m.selected[i] = !all
			}
		}
	case "u":
		// update selected
		var names []string
		for i, e := range vis {
			if m.selected[i] && e.Status == outdated.StatusOutdated {
				names = append(names, e.Name)
			}
		}
		if len(names) == 0 {
			return m, nil
		}
		tr := m.deps.Cfg.T()
		m.confirmMsg = tr.S("tui.outdated.confirm_update", len(names))
		m.confirmYes = afterOutdatedUpdate
		m.confirmData = names
		m.screen = screenConfirm
		m.cursor = 0
	case "t":
		// toggle show all
		m.showAll = !m.showAll
		tr := m.deps.Cfg.T()
		m.screen = screenLoading
		m.loadMsg = tr.S("tui.loading.outdated")
		return m, m.cmdOutdated()
	case "esc", "b":
		m.screen = screenHome
		m.cursor = 0
	case "q":
		return m, tea.Quit
	case "enter":
		// same as space toggle or if on outdated, toggle
		if len(vis) > 0 {
			m.selected[m.cursor] = !m.selected[m.cursor]
		}
	}
	return m, nil
}

func (m model) visibleOutdated() []outdated.Entry {
	if m.outdatedReport == nil {
		return nil
	}
	// 可更新置顶，其余保持原相对顺序
	var updatable, rest []outdated.Entry
	for _, e := range m.outdatedReport.Entries {
		if e.Status == outdated.StatusUnknown && !m.showAll {
			continue
		}
		if e.Status == outdated.StatusOutdated {
			updatable = append(updatable, e)
		} else {
			rest = append(rest, e)
		}
	}
	return append(updatable, rest...)
}

func (m model) keyManage(key string) (tea.Model, tea.Cmd) {
	items := m.manageMenu()
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(items)-1 {
			m.cursor++
		}
	case "esc", "b":
		m.screen = screenHome
		m.cursor = 3
	case "q":
		return m, tea.Quit
	case "enter":
		tr := m.deps.Cfg.T()
		switch items[m.cursor].id {
		case "remove":
			m.screen = screenLoading
			m.loadMsg = tr.S("tui.loading.skills")
			return m, m.cmdListSkills("skills_remove")
		case "disable":
			m.screen = screenLoading
			m.loadMsg = tr.S("tui.loading.skills")
			return m, m.cmdListSkills("skills_disable")
		case "enable":
			m.screen = screenLoading
			m.loadMsg = tr.S("tui.loading.gates")
			return m, m.cmdListGates("skills_enable")
		case "gates":
			m.screen = screenLoading
			m.loadMsg = tr.S("tui.loading.gates")
			return m, m.cmdListGates("gates")
		}
	}
	return m, nil
}

func (m model) filteredPick() []pickItem {
	if m.pickFilter == "" {
		return m.pickItems
	}
	f := strings.ToLower(m.pickFilter)
	var out []pickItem
	for _, it := range m.pickItems {
		if strings.Contains(strings.ToLower(it.label), f) || strings.Contains(strings.ToLower(it.id), f) {
			out = append(out, it)
		}
	}
	return out
}

func (m model) keyPickMulti(key string) (tea.Model, tea.Cmd) {
	items := m.filteredPick()
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(items)-1 {
			m.cursor++
		}
	case " ":
		if len(items) > 0 {
			// map cursor to original index by id
			id := items[m.cursor].id
			for i, it := range m.pickItems {
				if it.id == id {
					m.pickSelected[i] = !m.pickSelected[i]
					break
				}
			}
		}
	case "enter":
		var names []string
		for i, it := range m.pickItems {
			if m.pickSelected[i] {
				names = append(names, it.id)
			}
		}
		if len(names) == 0 && len(items) > 0 {
			// enter on single = select current
			names = []string{items[m.cursor].id}
		}
		if len(names) == 0 {
			return m, nil
		}
		tr := m.deps.Cfg.T()
		if m.pickAfter == afterRemove {
			m.confirmMsg = tr.S("tui.manage.confirm_remove", strings.Join(names, ", "))
			m.confirmYes = afterRemove
			m.confirmData = names
			m.screen = screenConfirm
			m.cursor = 0
		}
	case "esc", "b":
		m.screen = screenManage
		m.cursor = 0
	case "q":
		return m, tea.Quit
	case "backspace":
		if len(m.pickFilter) > 0 {
			m.pickFilter = m.pickFilter[:len(m.pickFilter)-1]
			m.cursor = 0
		}
	default:
		if len(key) == 1 && key[0] >= 32 && key != " " {
			m.pickFilter += key
			m.cursor = 0
		}
	}
	return m, nil
}

func (m model) keyPickOne(key string) (tea.Model, tea.Cmd) {
	items := m.filteredPick()
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(items)-1 {
			m.cursor++
		}
	case "enter":
		if len(items) == 0 {
			return m, nil
		}
		tr := m.deps.Cfg.T()
		// Choosing disable scope (second step)
		if m.disableScope == "await" && m.pendingSkill != "" {
			scope := items[m.cursor].id
			var tools []string
			if scope != "global" {
				tools = []string{scope}
			}
			m.confirmMsg = tr.S("tui.manage.confirm_disable", m.pendingSkill, items[m.cursor].label)
			m.confirmYes = afterDisable
			m.confirmData = disableData{Name: m.pendingSkill, Tools: tools}
			m.pendingSkill = ""
			m.disableScope = ""
			m.screen = screenConfirm
			m.cursor = 0
			return m, nil
		}
		name := items[m.cursor].id
		switch m.pickAfter {
		case afterDisable:
			m.pendingSkill = name
			scopeItems := []pickItem{
				{id: "global", label: tr.S("tui.gate.scope_global")},
			}
			for _, t := range skillgate.KnownTools {
				scopeItems = append(scopeItems, pickItem{id: t, label: t})
			}
			m.pickTitle = tr.S("tui.gate.pick_scope", name)
			m.pickItems = scopeItems
			m.pickMulti = false
			m.cursor = 0
			m.pickFilter = ""
			m.disableScope = "await"
			return m, nil
		case afterEnable:
			m.confirmMsg = tr.S("tui.manage.confirm_enable", name)
			m.confirmYes = afterEnable
			m.confirmData = disableData{Name: name, Tools: nil}
			m.screen = screenConfirm
			m.cursor = 0
			return m, nil
		}
	case "esc", "b":
		if m.pendingSkill != "" && m.disableScope == "await" {
			m.pendingSkill = ""
			m.disableScope = ""
			tr := m.deps.Cfg.T()
			m.screen = screenLoading
			m.loadMsg = tr.S("tui.loading.skills")
			return m, m.cmdListSkills("skills_disable")
		}
		m.screen = screenManage
		m.cursor = 0
	case "q":
		return m, tea.Quit
	case "backspace":
		if len(m.pickFilter) > 0 {
			m.pickFilter = m.pickFilter[:len(m.pickFilter)-1]
			m.cursor = 0
		}
	default:
		if len(key) == 1 && key[0] >= 32 {
			m.pickFilter += key
			m.cursor = 0
		}
	}
	return m, nil
}

func (m model) keyAbout(key string) (tea.Model, tea.Cmd) {
	items := m.aboutMenu()
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(items)-1 {
			m.cursor++
		}
	case "esc", "b":
		m.screen = screenHome
		m.cursor = 4
	case "q":
		return m, tea.Quit
	case "enter":
		tr := m.deps.Cfg.T()
		switch items[m.cursor].id {
		case "check":
			m.screen = screenLoading
			m.loadMsg = tr.S("tui.loading.upgrade_check")
			return m, m.cmdUpgradeCheck()
		case "upgrade":
			m.confirmMsg = tr.S("tui.about.confirm_upgrade")
			m.confirmYes = afterUpgrade
			m.screen = screenConfirm
			m.cursor = 0
		}
	}
	return m, nil
}

// --- commands ---

func (m model) cmdPlan(previewOnly bool) tea.Cmd {
	deps := m.deps
	kind := "plan"
	if previewOnly {
		kind = "plan_preview_only"
	}
	return func() tea.Msg {
		p, err := deps.buildPlan()
		return loadedMsg{kind: kind, err: err, payload: p}
	}
}

func (m model) cmdApply(strategy string) tea.Cmd {
	deps := m.deps
	p := m.plan
	return func() tea.Msg {
		res, _, err := deps.applyPlan(p, strategy)
		if err != nil {
			return loadedMsg{kind: "apply", err: err}
		}
		payload := &applyResultPayload{Migrated: res.Migrated, Linked: res.Linked, Skipped: res.Skipped}
		if res.Tx != nil {
			payload.TxID = res.Tx.ID
		}
		return loadedMsg{kind: "apply", payload: payload}
	}
}

func (m model) cmdUndo() tea.Cmd {
	deps := m.deps
	return func() tea.Msg {
		_, err := deps.undoLast()
		return loadedMsg{kind: "undo", err: err}
	}
}

func (m model) cmdDoctor() tea.Cmd {
	deps := m.deps
	return func() tea.Msg {
		issues, err := deps.runDoctor()
		return loadedMsg{kind: "doctor", err: err, payload: issues}
	}
}

func (m model) cmdDoctorCleanup(items []doctor.Issue) tea.Cmd {
	return func() tea.Msg {
		res := doctor.Cleanup(items)
		return loadedMsg{kind: "doctor_cleanup", payload: res}
	}
}

func (m model) cmdOutdated() tea.Cmd {
	deps := m.deps
	showAll := m.showAll
	return func() tea.Msg {
		res, err := deps.runOutdated(showAll)
		return loadedMsg{kind: "outdated", err: err, payload: res}
	}
}

func (m model) cmdUpdate(names []string) tea.Cmd {
	deps := m.deps
	return func() tea.Msg {
		ur, _, err := deps.updateSkills(names)
		return loadedMsg{kind: "update", err: err, payload: ur}
	}
}

func (m model) cmdListSkills(kind string) tea.Cmd {
	deps := m.deps
	return func() tea.Msg {
		var names []string
		var err error
		if kind == "skills_disable" {
			names, err = deps.listCanonicalSkills()
		} else {
			names, err = deps.listRemovableSkills()
		}
		return loadedMsg{kind: kind, err: err, payload: names}
	}
}

func (m model) cmdListGates(kind string) tea.Cmd {
	deps := m.deps
	return func() tea.Msg {
		rows, err := deps.listGateEntries()
		return loadedMsg{kind: kind, err: err, payload: rows}
	}
}

func (m model) cmdRemove(names []string) tea.Cmd {
	deps := m.deps
	return func() tea.Msg {
		res, _, err := deps.removeSkills(names)
		if err != nil {
			return loadedMsg{kind: "remove", err: err}
		}
		return loadedMsg{kind: "remove", payload: &removePayload{
			Removed: len(res.Removed),
			Failed:  len(res.Failed),
			Locks:   len(res.LockModified),
		}}
	}
}

func (m model) cmdDisable(d disableData) tea.Cmd {
	deps := m.deps
	return func() tea.Msg {
		res, err := deps.disableSkill(d.Name, d.Tools)
		return loadedMsg{kind: "disable", err: err, payload: res}
	}
}

func (m model) cmdEnable(d disableData) tea.Cmd {
	deps := m.deps
	return func() tea.Msg {
		res, err := deps.enableSkill(d.Name, d.Tools)
		return loadedMsg{kind: "enable", err: err, payload: res}
	}
}

func (m model) cmdUpgradeCheck() tea.Cmd {
	deps := m.deps
	return func() tea.Msg {
		res, err := deps.upgradeCheck()
		return loadedMsg{kind: "upgrade_check", err: err, payload: res}
	}
}

func (m model) cmdUpgradeApply() tea.Cmd {
	deps := m.deps
	return func() tea.Msg {
		res, err := deps.upgradeApply()
		return loadedMsg{kind: "upgrade_apply", err: err, payload: res}
	}
}
