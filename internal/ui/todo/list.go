package todo

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ihatemodels/gdev/internal/config"
	"github.com/ihatemodels/gdev/internal/ui/styles"
)

// UpdateListView handles input for the list view.
func (m Model) UpdateListView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visibleItems := (m.Height - 10) / 5
	if visibleItems < 1 {
		visibleItems = 1
	}

	key := msg.String()
	kb := m.Config.Keys()

	// Handle quit/back
	if config.MatchesAny(key, kb.Global.Quit, kb.Global.QuitAlt) {
		return m, func() tea.Msg { return BackToMenuMsg{} }
	}

	// Handle navigation
	if config.MatchesAny(key, kb.Global.MoveUp, kb.Global.MoveUpAlt) {
		if m.Cursor > 0 {
			m.Cursor--
			if m.Cursor < m.ListScroll {
				m.ListScroll = m.Cursor
			}
		}
		return m, nil
	}

	if config.MatchesAny(key, kb.Global.MoveDown, kb.Global.MoveDownAlt) {
		if m.Cursor < len(m.Todos)-1 {
			m.Cursor++
			if m.Cursor >= m.ListScroll+visibleItems {
				m.ListScroll = m.Cursor - visibleItems + 1
			}
		}
		return m, nil
	}

	// Handle list-specific keybindings
	switch {
	case config.Matches(key, kb.List.Top):
		m.Cursor = 0
		m.ListScroll = 0

	case config.Matches(key, kb.List.Bottom):
		if len(m.Todos) > 0 {
			m.Cursor = len(m.Todos) - 1
			if m.Cursor >= visibleItems {
				m.ListScroll = m.Cursor - visibleItems + 1
			}
		}

	case config.Matches(key, kb.List.PageUp):
		m.Cursor -= visibleItems
		if m.Cursor < 0 {
			m.Cursor = 0
		}
		m.ListScroll -= visibleItems
		if m.ListScroll < 0 {
			m.ListScroll = 0
		}

	case config.Matches(key, kb.List.PageDown):
		m.Cursor += visibleItems
		if m.Cursor >= len(m.Todos) {
			m.Cursor = len(m.Todos) - 1
		}
		if m.Cursor >= m.ListScroll+visibleItems {
			m.ListScroll = m.Cursor - visibleItems + 1
		}

	case config.Matches(key, kb.List.Select):
		if len(m.Todos) > 0 {
			t := m.Todos[m.Cursor]
			m.FormEditingTodo = &t
			m.FormBranch = t.Branch
			m.FormName = t.Name
			m.FormDescription = t.Description
			m.FormPrompts = make([]string, len(t.Prompts))
			copy(m.FormPrompts, t.Prompts)
			if len(m.FormPrompts) == 0 {
				m.FormPrompts = []string{""}
			}
			m.FormField = FieldBranch
			m.FormPromptIdx = 0
			m.CurrentView = EditView
		}

	case config.Matches(key, kb.List.New):
		m.CurrentView = CreateView
		m.FormBranch = m.Branch
		m.FormName = ""
		m.FormDescription = ""
		m.FormPrompts = []string{""}
		m.FormField = FieldBranch
		m.FormPromptIdx = 0
		m.FormEditingTodo = nil

	case config.Matches(key, kb.List.Delete):
		if len(m.Todos) > 0 {
			m.DeleteTarget = &m.Todos[m.Cursor]
			m.CurrentView = DeleteConfirmView
		}
	}

	return m, nil
}

// ViewList renders the list view.
func (m Model) ViewList() string {
	var b strings.Builder

	header := "  TODOs"
	if len(m.Todos) > 0 {
		header += styles.Help.Render(fmt.Sprintf(" (%d)", len(m.Todos)))
	}
	b.WriteString(styles.Title.Render(header))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("─────────────────────────────────────────"))
	b.WriteString("\n\n")

	if len(m.Todos) == 0 {
		b.WriteString(m.viewEmptyState())
	} else {
		b.WriteString(m.viewTodoCards())
	}

	b.WriteString("\n\n")
	kb := m.Config.Keys()
	b.WriteString(styles.Help.Render(fmt.Sprintf("↑/%s ↓/%s navigate • %s/%s top/bottom • %s/%s page",
		kb.Global.MoveUp, kb.Global.MoveDown, kb.List.Top, kb.List.Bottom, kb.List.PageUp, kb.List.PageDown)))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render(fmt.Sprintf("%s edit • %s new • %s delete • %s back",
		kb.List.Select, kb.List.New, kb.List.Delete, kb.Global.Quit)))

	return b.String()
}

func (m Model) viewEmptyState() string {
	var b strings.Builder

	b.WriteString(styles.Help.Render("  ┌─────────────────────────────────┐"))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("  │                                 │"))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("  │  "))
	b.WriteString(styles.Value.Render("No TODOs yet!"))
	b.WriteString(styles.Help.Render("              │"))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("  │                                 │"))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("  │  Press "))
	b.WriteString(styles.Selected.Render("n"))
	b.WriteString(styles.Help.Render(" to create your first  │"))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("  │                                 │"))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("  └─────────────────────────────────┘"))
	b.WriteString("\n")

	return b.String()
}

func (m Model) viewTodoCards() string {
	var b strings.Builder

	visibleItems := (m.Height - 10) / 5
	if visibleItems < 1 {
		visibleItems = 1
	}
	if visibleItems > len(m.Todos) {
		visibleItems = len(m.Todos)
	}

	if m.ListScroll > 0 {
		b.WriteString(styles.Help.Render("  ↑ more above"))
		b.WriteString("\n\n")
	}

	endIdx := m.ListScroll + visibleItems
	if endIdx > len(m.Todos) {
		endIdx = len(m.Todos)
	}

	for i := m.ListScroll; i < endIdx; i++ {
		t := m.Todos[i]
		isSelected := i == m.Cursor

		if isSelected {
			b.WriteString(styles.Cursor.Render("▸ "))
			b.WriteString(styles.Selected.Render("┌─ "))
			b.WriteString(styles.Selected.Render(t.Name))
		} else {
			b.WriteString("  ")
			b.WriteString(styles.Help.Render("┌─ "))
			b.WriteString(styles.Item.Render(t.Name))
		}
		b.WriteString("\n")

		prefix := "  "
		b.WriteString(prefix)
		b.WriteString(styles.Help.Render("│  "))
		b.WriteString(styles.Branch.Render(" " + t.Branch))
		b.WriteString(styles.Help.Render(fmt.Sprintf("  •  %d prompt", len(t.Prompts))))
		if len(t.Prompts) != 1 {
			b.WriteString(styles.Help.Render("s"))
		}
		b.WriteString("\n")

		if t.Description != "" {
			desc := strings.Split(t.Description, "\n")[0]
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}
			b.WriteString(prefix)
			b.WriteString(styles.Help.Render("│  "))
			b.WriteString(styles.Help.Render(desc))
			b.WriteString("\n")
		}

		b.WriteString(prefix)
		b.WriteString(styles.Help.Render("└───"))
		b.WriteString("\n")

		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}

	if endIdx < len(m.Todos) {
		b.WriteString("\n")
		b.WriteString(styles.Help.Render("  ↓ more below"))
	}

	return b.String()
}
