// Package todo provides the TODO management TUI component.
package todo

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ihatemodels/gdev/internal/config"
	"github.com/ihatemodels/gdev/internal/store"
	"github.com/ihatemodels/gdev/internal/todo"
	"github.com/ihatemodels/gdev/internal/ui/styles"
	"github.com/ihatemodels/gdev/internal/ui/terminal"
)

// View represents the current view within the TODO component.
type View int

const (
	ListView View = iota
	DetailView
	CreateView
	EditView
	DeleteConfirmView
	PromptEditorView
	TerminalView
)

// FormField represents which field is being edited in a form.
type FormField int

const (
	FieldBranch FormField = iota
	FieldName
	FieldDescription
	FieldPrompts
)

// Model is the Bubble Tea model for TODO management.
type Model struct {
	Store    *store.Store
	Config   *config.Config
	RepoPath string
	Branch   string // current branch for defaults

	CurrentView View
	Todos       []todo.Todo
	Cursor      int

	// For detail view
	SelectedTodo *todo.Todo

	// Form fields
	FormBranch      string
	FormName        string
	FormDescription string
	FormPrompts     []string
	FormField       FormField
	FormPromptIdx   int  // which prompt is selected when editing prompts
	FormEditing     bool // true when actively editing a field (insert mode)
	FormEditingTodo *todo.Todo

	// Delete confirmation
	DeleteTarget *todo.Todo

	// Prompt editor state
	EditorContent   string
	EditorCursorPos int
	PreviousView    View

	// Scrolling state
	ListScroll   int // scroll offset for list view
	DetailScroll int // scroll offset for detail view

	// UI state
	Width     int
	Height    int
	ErrMsg    string
	Loading   bool
	Improving bool // true when LLM is improving a prompt

	// Terminal modal for running commands
	Terminal         terminal.Model
	TerminalCallback func(m *Model, output string) // callback when terminal closes
}

// Message types
type (
	TodosLoadedMsg struct {
		Todos []todo.Todo
	}

	TodoErrorMsg struct {
		Err error
	}

	TodoSavedMsg struct{}

	TodoDeletedMsg struct{}

	BackToMenuMsg struct{}
)

// New creates a new Model.
func New(s *store.Store, cfg *config.Config, repoPath, branch string) Model {
	return Model{
		Store:       s,
		Config:      cfg,
		RepoPath:    repoPath,
		Branch:      branch,
		CurrentView: ListView,
		FormPrompts: []string{""},
	}
}

// SetSize sets the width and height of the model.
func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.LoadTodos
}

// LoadTodos loads the todos from the store.
func (m Model) LoadTodos() tea.Msg {
	list, err := m.Store.GetTodos(m.RepoPath)
	if err != nil {
		return TodoErrorMsg{Err: err}
	}
	return TodosLoadedMsg{Todos: list.Todos}
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Terminal.SetSize(msg.Width, msg.Height)
		return m, nil

	case TodosLoadedMsg:
		m.Todos = msg.Todos
		m.Loading = false
		return m, nil

	case TodoErrorMsg:
		m.ErrMsg = msg.Err.Error()
		m.Loading = false
		return m, nil

	case TodoSavedMsg:
		m.CurrentView = ListView
		m.ErrMsg = ""
		return m, m.LoadTodos

	case TodoDeletedMsg:
		m.CurrentView = ListView
		m.DeleteTarget = nil
		return m, m.LoadTodos

	case terminal.TickMsg:
		// Forward tick messages to terminal
		if m.CurrentView == TerminalView {
			var cmd tea.Cmd
			m.Terminal, cmd = m.Terminal.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		m.ErrMsg = ""
		return m.handleKeyMsg(msg)
	}
	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.CurrentView {
	case ListView:
		return m.UpdateListView(msg)
	case DetailView:
		return m.UpdateDetailView(msg)
	case CreateView, EditView:
		return m.UpdateFormView(msg)
	case DeleteConfirmView:
		return m.UpdateDeleteConfirmView(msg)
	case PromptEditorView:
		return m.UpdatePromptEditor(msg)
	case TerminalView:
		return m.UpdateTerminalView(msg)
	}
	return m, nil
}

// UpdateTerminalView handles input for the terminal modal.
func (m Model) UpdateTerminalView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check if user wants to close the terminal
	if m.Terminal.ShouldClose(msg) {
		// Get raw output before closing (without status messages)
		output := m.Terminal.GetRawOutput()
		// Execute callback if set
		if m.TerminalCallback != nil {
			m.TerminalCallback(&m, output)
		}
		m.CurrentView = m.PreviousView
		m.TerminalCallback = nil
		return m, nil
	}

	// Forward other keys to terminal for scrolling
	var cmd tea.Cmd
	m.Terminal, cmd = m.Terminal.Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	if m.Width == 0 {
		return "Loading..."
	}

	// Terminal view renders as a centered modal overlay
	if m.CurrentView == TerminalView {
		return m.Terminal.ViewCentered(m.Width, m.Height)
	}

	var content strings.Builder

	switch m.CurrentView {
	case ListView:
		content.WriteString(m.ViewList())
	case DetailView:
		content.WriteString(m.ViewDetail())
	case CreateView:
		content.WriteString(m.ViewForm("Create TODO"))
	case EditView:
		content.WriteString(m.ViewForm("Edit TODO"))
	case DeleteConfirmView:
		content.WriteString(m.ViewDeleteConfirm())
	case PromptEditorView:
		content.WriteString(m.ViewPromptEditor())
	}

	if m.ErrMsg != "" {
		content.WriteString("\n\n")
		content.WriteString(styles.Error.Render("Error: " + m.ErrMsg))
	}

	return lipgloss.NewStyle().
		Width(m.Width).
		Height(m.Height).
		Padding(1, 2).
		Render(content.String())
}
