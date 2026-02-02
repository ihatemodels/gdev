// Package app provides the main application TUI model.
package app

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ihatemodels/gdev/internal/config"
	"github.com/ihatemodels/gdev/internal/git"
	"github.com/ihatemodels/gdev/internal/store"
	"github.com/ihatemodels/gdev/internal/ui/commit"
	"github.com/ihatemodels/gdev/internal/ui/styles"
	"github.com/ihatemodels/gdev/internal/ui/terminal"
	"github.com/ihatemodels/gdev/internal/ui/todo"
)

const banner = `
  ██████╗ ██████╗ ███████╗██╗   ██╗
 ██╔════╝ ██╔══██╗██╔════╝██║   ██║
 ██║  ███╗██║  ██║█████╗  ██║   ██║
 ██║   ██║██║  ██║██╔══╝  ╚██╗ ██╔╝
 ╚██████╔╝██████╔╝███████╗ ╚████╔╝
  ╚═════╝ ╚═════╝ ╚══════╝  ╚═══╝`

// View represents which view is currently active.
type View int

const (
	MainMenuView View = iota
	TodosView
	TerminalTestView
	CommitView
)

// RepoInfo holds information about the current git repository.
type RepoInfo struct {
	Repo       *git.Repo
	State      *store.RepoState
	Ahead      int
	Behind     int
	HasChanges bool
}

// Model is the main application model.
type Model struct {
	store    *store.Store
	config   *config.Config
	repoInfo *RepoInfo
	version  string
	choices  []string
	cursor   int
	width    int
	height   int

	currentView View
	todoModel   *todo.Model
	commitModel *commit.Model
	terminal    terminal.Model
}

// New creates a new application model.
func New(s *store.Store, cfg *config.Config, ri *RepoInfo, version string, startView View) Model {
	m := Model{
		store:       s,
		config:      cfg,
		repoInfo:    ri,
		version:     version,
		currentView: startView,
		choices: []string{
			"󰘬  Branches",
			"  Pull Requests",
			"  Claude Sessions",
			"  TODOs",
			"  Smart Commit",
			"  Terminal Test",
			"  Settings",
			"  Quit",
		},
	}

	if ri != nil && ri.Repo != nil {
		tm := todo.New(s, cfg, ri.Repo.Root, ri.Repo.Branch)
		m.todoModel = &tm
	}

	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.currentView == TodosView && m.todoModel != nil {
		return m.todoModel.Init()
	}
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle terminal test view
	if m.currentView == TerminalTestView {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.terminal.SetSize(msg.Width, msg.Height)
			return m, nil
		case terminal.TickMsg:
			var cmd tea.Cmd
			m.terminal, cmd = m.terminal.Update(msg)
			return m, cmd
		case tea.KeyMsg:
			if m.terminal.ShouldClose(msg) {
				m.currentView = MainMenuView
				return m, nil
			}
			var cmd tea.Cmd
			m.terminal, cmd = m.terminal.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	if m.currentView == TodosView {
		if _, ok := msg.(todo.BackToMenuMsg); ok {
			m.currentView = MainMenuView
			return m, nil
		}

		if wsm, ok := msg.(tea.WindowSizeMsg); ok {
			m.width = wsm.Width
			m.height = wsm.Height
		}

		updatedModel, cmd := m.todoModel.Update(msg)
		if tm, ok := updatedModel.(todo.Model); ok {
			m.todoModel = &tm
		}
		return m, cmd
	}

	if m.currentView == CommitView && m.commitModel != nil {
		if _, ok := msg.(commit.BackToMenuMsg); ok {
			m.currentView = MainMenuView
			return m, nil
		}

		if wsm, ok := msg.(tea.WindowSizeMsg); ok {
			m.width = wsm.Width
			m.height = wsm.Height
		}

		updatedModel, cmd := m.commitModel.Update(msg)
		if cm, ok := updatedModel.(commit.Model); ok {
			m.commitModel = &cm
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		key := msg.String()
		kb := m.config.Keys()

		switch {
		case key == "ctrl+c" || config.MatchesAny(key, kb.Global.Quit, kb.Global.QuitAlt):
			return m, tea.Quit
		case config.MatchesAny(key, kb.Global.MoveUp, kb.Global.MoveUpAlt):
			if m.cursor > 0 {
				m.cursor--
			}
		case config.MatchesAny(key, kb.Global.MoveDown, kb.Global.MoveDownAlt):
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case config.MatchesAny(key, kb.List.Select, " "):
			return m.handleMenuSelection()
		}
	}
	return m, nil
}

func (m Model) handleMenuSelection() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case 3: // TODOs
		if m.repoInfo != nil && m.repoInfo.Repo != nil && m.todoModel != nil {
			m.currentView = TodosView
			m.todoModel.SetSize(m.width, m.height)
			return m, m.todoModel.Init()
		}
	case 4: // Smart Commit
		if m.repoInfo != nil && m.repoInfo.Repo != nil {
			cm := commit.New(m.config, m.repoInfo.Repo.Root)
			cm.SetSize(m.width, m.height)
			m.commitModel = &cm
			m.currentView = CommitView
			return m, m.commitModel.Init()
		}
	case 5: // Terminal Test
		if m.repoInfo != nil && m.repoInfo.Repo != nil {
			m.terminal = terminal.New(m.config, "Git Status Loop (0.5s)")
			m.terminal.Dir = m.repoInfo.Repo.Root
			m.terminal.SetSize(m.width, m.height)
			m.currentView = TerminalTestView
			// Run git status in a loop with 0.5s sleep
			cmd := m.terminal.RunCommand("bash", "-c",
				`for i in $(seq 1 20); do echo "=== Run $i at $(date +%H:%M:%S) ==="; git status --short; echo ""; sleep 0.5; done; echo "Done!"`)
			return m, cmd
		}
	case 7: // Quit
		return m, tea.Quit
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.currentView == TerminalTestView {
		return m.terminal.ViewCentered(m.width, m.height)
	}

	if m.currentView == TodosView {
		return m.todoModel.View()
	}

	if m.currentView == CommitView && m.commitModel != nil {
		return m.commitModel.View()
	}

	var content strings.Builder

	content.WriteString(styles.Banner.Render(banner))
	content.WriteString("\n")
	content.WriteString(styles.Version.Render(fmt.Sprintf("v%s", m.version)))
	content.WriteString("\n\n")

	if m.repoInfo != nil {
		content.WriteString(m.renderRepoInfo())
		content.WriteString("\n")
	} else {
		content.WriteString(styles.Dim.Render("  Not in a git repository"))
		content.WriteString("\n\n")
	}

	content.WriteString(styles.Title.Render("What would you like to do?"))
	content.WriteString("\n\n")

	for i, choice := range m.choices {
		if m.cursor == i {
			cursor := styles.Cursor.Render("▸ ")
			content.WriteString(styles.Selected.Render(cursor + choice))
		} else {
			content.WriteString(styles.Item.Render("  " + choice))
		}
		content.WriteString("\n")
	}

	content.WriteString("\n")
	kb := m.config.Keys()
	content.WriteString(styles.Help.Render(fmt.Sprintf("↑/%s up • ↓/%s down • %s select • %s quit",
		kb.Global.MoveUp, kb.Global.MoveDown, kb.List.Select, kb.Global.QuitAlt)))

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content.String())
}

func (m Model) renderRepoInfo() string {
	ri := m.repoInfo
	var parts []string

	repoName := styles.Repo.Render(ri.Repo.Name)
	branch := styles.Branch.Render(" " + ri.Repo.Branch)
	parts = append(parts, fmt.Sprintf("  %s %s", repoName, branch))

	var status []string
	if ri.Behind > 0 {
		status = append(status, styles.Status.Render(fmt.Sprintf("↓%d", ri.Behind)))
	}
	if ri.Ahead > 0 {
		status = append(status, styles.Status.Render(fmt.Sprintf("↑%d", ri.Ahead)))
	}
	if ri.HasChanges {
		status = append(status, styles.Status.Render("●"))
	}
	if len(status) > 0 {
		parts[0] += "  " + strings.Join(status, " ")
	}

	if ri.State != nil && !ri.State.LastOpenedAt.IsZero() {
		lastOpened := formatTimeAgo(ri.State.LastOpenedAt)
		parts = append(parts, styles.Dim.Render(fmt.Sprintf("  Last opened: %s", lastOpened)))
	}

	return strings.Join(parts, "\n") + "\n"
}

func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}
