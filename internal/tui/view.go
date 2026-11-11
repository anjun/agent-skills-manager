package tui

import (
	"fmt"
	"strings"

	"github.com/anj/agent-skills-manager/internal/outdated"
)

func (m model) View() string {
	if m.screen == screenLoading {
		return m.viewLoading()
	}
	switch m.screen {
	case screenHome:
		return m.viewHome()
	case screenSyncPreview:
		return m.viewSyncPreview()
	case screenSyncMore:
		return m.viewMenuList(m.deps.Cfg.T().S("tui.sync.more_title"), m.syncMoreMenu(), m.cursor)
	case screenStrategy:
		return m.viewStrategy()
	case screenConfirm:
		return m.viewConfirm()
	case screenResult:
		return m.viewResult()
	case screenOutdated:
		return m.viewOutdated()
	case screenDoctor:
		return m.viewDoctor()
	case screenManage:
		return m.viewMenuList(m.deps.Cfg.T().S("tui.manage.title"), m.manageMenu(), m.cursor)
	case screenPickMulti, screenPickOne:
		return m.viewPick()
	case screenAbout:
		return m.viewAbout()
	default:
		return ""
	}
}

func (m model) viewLoading() string {
	tr := m.deps.Cfg.T()
	var b strings.Builder
	b.WriteString(headerStyle.Render(tr.S("tui.loading.title")))
	b.WriteString("\n\n")
	b.WriteString("  " + m.loadMsg + "\n")
	b.WriteString("\n")
	b.WriteString(footerStyle.Render(tr.S("tui.footer.loading")))
	return b.String()
}

func (m model) viewHome() string {
	tr := m.deps.Cfg.T()
	var b strings.Builder
	b.WriteString(titleStyle.Render("asm"))
	b.WriteString("  ")
	b.WriteString(subtitleStyle.Render(tr.S("tui.tagline")))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render(tr.S("tui.repo")))
	b.WriteString("\n")
	if m.banner != "" {
		b.WriteString(bannerStyle.Render(m.banner))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	items := m.mainMenu()
	for i, it := range items {
		name := tr.S(it.name)
		desc := tr.S(it.desc)
		prefix := fmt.Sprintf("%d. ", i+1)
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("> " + prefix + name))
			b.WriteString("  ")
			b.WriteString(descStyle.Render(desc))
		} else {
			b.WriteString(itemStyle.Render(prefix + name))
			b.WriteString("  ")
			b.WriteString(descStyle.Render(desc))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render(tr.S("tui.footer.home")))
	return b.String()
}

func (m model) viewMenuList(title string, items []menuItem, cursor int) string {
	tr := m.deps.Cfg.T()
	var b strings.Builder
	b.WriteString(headerStyle.Render(title))
	b.WriteString("\n")
	for i, it := range items {
		name := tr.S(it.name)
		desc := tr.S(it.desc)
		if i == cursor {
			b.WriteString(selectedStyle.Render("> " + name))
			b.WriteString("  ")
			b.WriteString(descStyle.Render(desc))
		} else {
			b.WriteString(itemStyle.Render(name))
			b.WriteString("  ")
			b.WriteString(descStyle.Render(desc))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render(tr.S("tui.footer.nav")))
	return b.String()
}

func (m model) viewSyncPreview() string {
	tr := m.deps.Cfg.T()
	var b strings.Builder
	b.WriteString(headerStyle.Render(tr.S("tui.sync.preview_title")))
	b.WriteString("\n")
	if m.plan != nil {
		for _, line := range formatPlanSummary(m.deps.Cfg, m.plan) {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	opts := []string{
		tr.S("tui.sync.action_apply"),
		tr.S("tui.sync.action_more"),
		tr.S("tui.sync.action_back"),
	}
	for i, o := range opts {
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("> " + o))
		} else {
			b.WriteString(itemStyle.Render(o))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render(tr.S("tui.footer.nav")))
	return b.String()
}

func (m model) viewStrategy() string {
	tr := m.deps.Cfg.T()
	var b strings.Builder
	b.WriteString(headerStyle.Render(tr.S("tui.sync.strategy_title")))
	b.WriteString("\n")
	b.WriteString(descStyle.Render(tr.S("tui.sync.strategy_hint")))
	b.WriteString("\n\n")
	for i, s := range m.strategies {
		label := tr.S("tui.sync.strategy." + s)
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("> " + label))
		} else {
			b.WriteString(itemStyle.Render(label))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render(tr.S("tui.footer.nav")))
	return b.String()
}

func (m model) viewConfirm() string {
	tr := m.deps.Cfg.T()
	var b strings.Builder
	b.WriteString(headerStyle.Render(tr.S("tui.confirm.title")))
	b.WriteString("\n\n")
	b.WriteString("  " + m.confirmMsg + "\n\n")
	yes := tr.S("tui.confirm.yes")
	no := tr.S("tui.confirm.no")
	if m.cursor == 0 {
		b.WriteString(selectedStyle.Render("> " + yes))
		b.WriteString("   ")
		b.WriteString(itemStyle.Render(no))
	} else {
		b.WriteString(itemStyle.Render(yes))
		b.WriteString("   ")
		b.WriteString(selectedStyle.Render("> " + no))
	}
	b.WriteString("\n\n")
	b.WriteString(footerStyle.Render(tr.S("tui.footer.confirm")))
	return b.String()
}

func (m model) viewResult() string {
	tr := m.deps.Cfg.T()
	var b strings.Builder
	b.WriteString(headerStyle.Render(m.resultTitle))
	b.WriteString("\n\n")
	if m.resultErr != "" {
		b.WriteString(errorStyle.Render(m.resultErr))
		b.WriteString("\n")
	}
	for _, line := range m.resultLines {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render(tr.S("tui.footer.result")))
	return b.String()
}

func (m model) viewOutdated() string {
	tr := m.deps.Cfg.T()
	var b strings.Builder
	b.WriteString(headerStyle.Render(tr.S("tui.outdated.title")))
	b.WriteString("\n")
	if m.outdatedReport != nil {
		b.WriteString(tr.S("outdated.summary",
			m.outdatedReport.Summary.Outdated,
			m.outdatedReport.Summary.Current,
			m.outdatedReport.Summary.Unknown,
			m.outdatedReport.Summary.Error,
			m.outdatedReport.Summary.Total,
		))
		b.WriteString("\n\n")
	}
	vis := m.visibleOutdated()
	if len(vis) == 0 {
		b.WriteString(okStyle.Render(tr.S("tui.outdated.empty")))
		b.WriteString("\n")
	}
	maxShow := m.listPageSize(20)
	start, end := listWindow(m.cursor, len(vis), maxShow)
	if start > 0 {
		b.WriteString(descStyle.Render(tr.S("tui.more_above", start)))
		b.WriteString("\n")
	}
	for i := start; i < end; i++ {
		e := vis[i]
		mark := " "
		if m.selected[i] {
			mark = "✓"
		}
		st := statusShort(tr, e.Status)
		line := fmt.Sprintf("[%s] %s  %s  %s", mark, st, e.Name, e.Message)
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("> " + line))
		} else {
			b.WriteString(itemStyle.Render(line))
		}
		b.WriteString("\n")
	}
	if end < len(vis) {
		b.WriteString(descStyle.Render(tr.S("tui.more_below", len(vis)-end)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	allHint := tr.S("tui.outdated.show_all_off")
	if m.showAll {
		allHint = tr.S("tui.outdated.show_all_on")
	}
	b.WriteString(descStyle.Render(allHint))
	b.WriteString("\n")
	b.WriteString(footerStyle.Render(tr.S("tui.footer.outdated")))
	return b.String()
}

func statusShort(tr interface{ S(string, ...any) string }, s outdated.Status) string {
	switch s {
	case outdated.StatusOutdated:
		return tr.S("outdated.status_outdated")
	case outdated.StatusCurrent:
		return tr.S("outdated.status_current")
	case outdated.StatusUnknown:
		return tr.S("outdated.status_unknown")
	case outdated.StatusError:
		return tr.S("outdated.status_error")
	default:
		return string(s)
	}
}

func (m model) viewDoctor() string {
	tr := m.deps.Cfg.T()
	var b strings.Builder
	b.WriteString(headerStyle.Render(tr.S("tui.menu.doctor")))
	b.WriteString("\n")
	issues := m.doctorIssues
	if len(issues) == 0 {
		b.WriteString("\n")
		b.WriteString(okStyle.Render(tr.S("doctor.ok")))
		b.WriteString("\n\n")
		b.WriteString(footerStyle.Render(tr.S("tui.footer.result")))
		return b.String()
	}

	cleanable := 0
	selected := 0
	for i, is := range issues {
		if is.Cleanable() {
			cleanable++
			if m.doctorSelected[i] {
				selected++
			}
		}
	}
	b.WriteString(descStyle.Render(tr.S("tui.doctor.summary", len(issues), cleanable, selected)))
	b.WriteString("\n\n")

	maxShow := m.listPageSize(14)
	start, end := listWindow(m.cursor, len(issues), maxShow)
	if start > 0 {
		b.WriteString(descStyle.Render(tr.S("tui.more_above", start)))
		b.WriteString("\n")
	}
	for i := start; i < end; i++ {
		is := issues[i]
		mark := " "
		if is.Cleanable() {
			if m.doctorSelected[i] {
				mark = "✓"
			} else {
				mark = " "
			}
		} else {
			mark = "-"
		}
		path := is.DisplayPath()
		line := fmt.Sprintf("[%s] %s  %s", mark, tr.SeverityLabel(is.Severity), path)
		// second line truncated message
		msg := is.Message
		if len(msg) > 90 {
			msg = msg[:87] + "..."
		}
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("> " + line))
			b.WriteString("\n")
			b.WriteString(selectedStyle.Render("        " + msg))
		} else {
			b.WriteString(itemStyle.Render(line))
			b.WriteString("\n")
			b.WriteString(descStyle.Render("        " + msg))
		}
		b.WriteString("\n")
	}
	if end < len(issues) {
		b.WriteString(descStyle.Render(tr.S("tui.more_below", len(issues)-end)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if cleanable > 0 {
		b.WriteString(descStyle.Render(tr.S("tui.doctor.hint_cleanup")))
		b.WriteString("\n")
	} else {
		b.WriteString(descStyle.Render(tr.S("tui.doctor.hint_sync")))
		b.WriteString("\n")
	}
	b.WriteString(footerStyle.Render(tr.S("tui.footer.doctor")))
	return b.String()
}

func (m model) viewPick() string {
	tr := m.deps.Cfg.T()
	var b strings.Builder
	b.WriteString(headerStyle.Render(m.pickTitle))
	b.WriteString("\n")
	if m.pickFilter != "" {
		b.WriteString(descStyle.Render(tr.S("tui.pick.filter", m.pickFilter)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	items := m.filteredPick()
	if len(items) == 0 {
		b.WriteString(tr.S("tui.pick.empty"))
		b.WriteString("\n")
	}
	maxShow := m.listPageSize(18)
	start, end := listWindow(m.cursor, len(items), maxShow)
	if start > 0 {
		b.WriteString(descStyle.Render(tr.S("tui.more_above", start)))
		b.WriteString("\n")
	}
	for i := start; i < end; i++ {
		it := items[i]
		line := it.label
		if m.pickMulti {
			// find selected by id
			sel := false
			for j, orig := range m.pickItems {
				if orig.id == it.id && m.pickSelected[j] {
					sel = true
					break
				}
			}
			mark := " "
			if sel {
				mark = "✓"
			}
			line = fmt.Sprintf("[%s] %s", mark, it.label)
		}
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("> " + line))
		} else {
			b.WriteString(itemStyle.Render(line))
		}
		b.WriteString("\n")
	}
	if end < len(items) {
		b.WriteString(descStyle.Render(tr.S("tui.more_below", len(items)-end)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if m.pickMulti {
		b.WriteString(footerStyle.Render(tr.S("tui.footer.pick_multi")))
	} else {
		b.WriteString(footerStyle.Render(tr.S("tui.footer.pick_one")))
	}
	return b.String()
}

// listWindow returns [start, end) so cursor stays visible in a page of pageSize rows.
func listWindow(cursor, total, pageSize int) (start, end int) {
	if total <= 0 {
		return 0, 0
	}
	if pageSize < 1 {
		pageSize = 1
	}
	if total <= pageSize {
		return 0, total
	}
	// Keep cursor on the last row of the window when scrolling down (minimal jump).
	start = cursor - pageSize + 1
	if start < 0 {
		start = 0
	}
	if start > total-pageSize {
		start = total - pageSize
	}
	return start, start + pageSize
}

// listPageSize picks how many rows to show; shrinks on short terminals.
func (m model) listPageSize(defaultN int) int {
	n := defaultN
	if m.height > 0 {
		// header + footer + padding ≈ 8 lines
		avail := m.height - 8
		if avail < 5 {
			avail = 5
		}
		if avail < n {
			n = avail
		}
	}
	return n
}

func (m model) viewAbout() string {
	tr := m.deps.Cfg.T()
	var b strings.Builder
	b.WriteString(headerStyle.Render(tr.S("tui.menu.about")))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  %s  %s\n", tr.S("tui.about.version"), m.deps.Version))
	b.WriteString("  " + tr.S("tui.tagline") + "\n")
	b.WriteString("  " + tr.S("tui.repo") + "\n")
	if m.banner != "" {
		b.WriteString("\n")
		b.WriteString(bannerStyle.Render(m.banner))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	items := m.aboutMenu()
	for i, it := range items {
		name := tr.S(it.name)
		desc := tr.S(it.desc)
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("> " + name))
			b.WriteString("  ")
			b.WriteString(descStyle.Render(desc))
		} else {
			b.WriteString(itemStyle.Render(name))
			b.WriteString("  ")
			b.WriteString(descStyle.Render(desc))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render(tr.S("tui.footer.nav")))
	return b.String()
}
