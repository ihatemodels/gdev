package todo

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ihatemodels/gdev/internal/config"
	"github.com/ihatemodels/gdev/internal/ui/styles"
)

// UpdateDetailView handles input for the detail view.
func (m Model) UpdateDetailView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	kb := m.Config.Keys()

	// Handle back/quit
	if config.MatchesAny(key, kb.Global.Quit, kb.Global.QuitAlt, kb.Detail.Back) {
		m.CurrentView = ListView
		m.SelectedTodo = nil
		return m, nil
	}

	// Handle scroll up
	if config.MatchesAny(key, kb.Detail.ScrollUp, kb.Global.MoveUp, kb.Global.MoveUpAlt) {
		if m.DetailScroll > 0 {
			m.DetailScroll--
		}
		return m, nil
	}

	// Handle scroll down
	if config.MatchesAny(key, kb.Detail.ScrollDown, kb.Global.MoveDown, kb.Global.MoveDownAlt) {
		m.DetailScroll++
		return m, nil
	}

	// Handle other keybindings
	switch {
	case config.Matches(key, kb.List.Top):
		m.DetailScroll = 0

	case config.Matches(key, kb.List.Bottom):
		m.DetailScroll = 9999

	case config.Matches(key, kb.List.PageUp):
		m.DetailScroll -= 10
		if m.DetailScroll < 0 {
			m.DetailScroll = 0
		}

	case config.Matches(key, kb.List.PageDown):
		m.DetailScroll += 10

	case config.Matches(key, kb.Detail.Edit):
		if m.SelectedTodo != nil {
			m.FormEditingTodo = m.SelectedTodo
			m.FormBranch = m.SelectedTodo.Branch
			m.FormName = m.SelectedTodo.Name
			m.FormDescription = m.SelectedTodo.Description
			m.FormPrompts = make([]string, len(m.SelectedTodo.Prompts))
			copy(m.FormPrompts, m.SelectedTodo.Prompts)
			if len(m.FormPrompts) == 0 {
				m.FormPrompts = []string{""}
			}
			m.FormField = FieldBranch
			m.FormPromptIdx = 0
			m.CurrentView = EditView
		}

	case config.Matches(key, kb.Detail.Delete):
		if m.SelectedTodo != nil {
			m.DeleteTarget = m.SelectedTodo
			m.CurrentView = DeleteConfirmView
		}
	}

	return m, nil
}

// ViewDetail renders the detail view.
func (m Model) ViewDetail() string {
	if m.SelectedTodo == nil {
		return ""
	}
	t := m.SelectedTodo

	var lines []string

	lines = append(lines, styles.Title.Render("  TODO Details"))
	lines = append(lines, styles.Help.Render("─────────────────────────────────────────────────────"))
	lines = append(lines, "")

	lines = append(lines, styles.Label.Render("Name: ")+styles.Value.Render(t.Name))
	lines = append(lines, styles.Label.Render("Branch: ")+styles.Branch.Render(" "+t.Branch))
	lines = append(lines, "")

	lines = append(lines, styles.Label.Render("Description:"))
	if t.Description != "" {
		descLines := strings.Split(t.Description, "\n")
		for _, dl := range descLines {
			lines = append(lines, "  "+styles.Value.Render(dl))
		}
	} else {
		lines = append(lines, "  "+styles.Help.Render("(no description)"))
	}
	lines = append(lines, "")

	lines = append(lines, styles.Label.Render("Prompts:"))
	if len(t.Prompts) == 0 {
		lines = append(lines, "  "+styles.Help.Render("(no prompts)"))
	} else {
		for i, p := range t.Prompts {
			lines = append(lines, "")
			lines = append(lines, styles.Prompt.Render(fmt.Sprintf("  ─── Prompt %d ───", i+1)))
			promptLines := strings.Split(p, "\n")
			for _, pl := range promptLines {
				lines = append(lines, "  "+styles.Value.Render(pl))
			}
		}
	}

	visibleLines := m.Height - 8
	if visibleLines < 5 {
		visibleLines = 5
	}

	maxScroll := len(lines) - visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	scroll := m.DetailScroll
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	var b strings.Builder

	if scroll > 0 {
		b.WriteString(styles.Help.Render("  ↑ scroll up for more"))
		b.WriteString("\n")
	}

	endIdx := scroll + visibleLines
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	for i := scroll; i < endIdx; i++ {
		b.WriteString(lines[i])
		b.WriteString("\n")
	}

	if endIdx < len(lines) {
		b.WriteString(styles.Help.Render("  ↓ scroll down for more"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	kb := m.Config.Keys()
	b.WriteString(styles.Help.Render(fmt.Sprintf("↑/%s ↓/%s scroll • %s/%s top/bottom • %s/%s page",
		kb.Detail.ScrollUp, kb.Detail.ScrollDown, kb.List.Top, kb.List.Bottom, kb.List.PageUp, kb.List.PageDown)))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render(fmt.Sprintf("%s edit • %s delete • %s back",
		kb.Detail.Edit, kb.Detail.Delete, kb.Detail.Back)))

	return b.String()
}

// UpdateDeleteConfirmView handles input for the delete confirmation view.
func (m Model) UpdateDeleteConfirmView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.DeleteTarget != nil {
			target := m.DeleteTarget
			return m, func() tea.Msg {
				if err := m.Store.DeleteTodo(m.RepoPath, target.ID); err != nil {
					return TodoErrorMsg{Err: err}
				}
				return TodoDeletedMsg{}
			}
		}
	case "n", "N", "esc":
		m.CurrentView = ListView
		m.DeleteTarget = nil
	}
	return m, nil
}

// ViewDeleteConfirm renders the delete confirmation view.
func (m Model) ViewDeleteConfirm() string {
	var b strings.Builder

	b.WriteString(styles.Confirm.Render("  Delete TODO?"))
	b.WriteString("\n\n")

	if m.DeleteTarget != nil {
		b.WriteString(styles.Value.Render(fmt.Sprintf("  \"%s\"", m.DeleteTarget.Name)))
		b.WriteString("\n")
		b.WriteString(styles.Branch.Render(fmt.Sprintf("   %s", m.DeleteTarget.Branch)))
	}

	b.WriteString("\n\n")
	b.WriteString(styles.Help.Render("y confirm • n cancel"))

	return b.String()
}
