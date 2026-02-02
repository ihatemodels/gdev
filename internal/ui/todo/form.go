package todo

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ihatemodels/gdev/internal/config"
	"github.com/ihatemodels/gdev/internal/todo"
	"github.com/ihatemodels/gdev/internal/ui/styles"
	"github.com/ihatemodels/gdev/internal/ui/terminal"
)

// UpdateFormView handles input for the create/edit form view.
func (m Model) UpdateFormView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	kb := m.Config.Keys()

	// If in edit mode, handle text input
	if m.FormEditing {
		return m.handleFormEditMode(msg)
	}

	// Navigation mode - handle shortcuts and navigation

	// Handle cancel (exit form)
	if config.Matches(key, kb.Form.Cancel) {
		m.CurrentView = ListView
		return m, nil
	}

	// Handle submit
	if config.Matches(key, kb.Form.Submit) {
		return m.saveForm()
	}

	// Handle vertical navigation
	if config.MatchesAny(key, kb.Global.MoveUp, kb.Global.MoveUpAlt) || msg.Type == tea.KeyUp {
		if m.FormField == FieldPrompts && m.FormPromptIdx > 0 {
			m.FormPromptIdx--
		} else if m.FormField > FieldBranch {
			m.FormField--
			if m.FormField == FieldPrompts {
				m.FormPromptIdx = len(m.FormPrompts) - 1
			}
		}
		return m, nil
	}

	if config.MatchesAny(key, kb.Global.MoveDown, kb.Global.MoveDownAlt) || msg.Type == tea.KeyDown {
		if m.FormField == FieldPrompts && m.FormPromptIdx < len(m.FormPrompts)-1 {
			m.FormPromptIdx++
		} else if m.FormField < FieldPrompts {
			m.FormField++
			if m.FormField == FieldPrompts {
				m.FormPromptIdx = 0
			}
		}
		return m, nil
	}

	// Handle field navigation with tab
	if config.Matches(key, kb.Form.NextField) {
		if m.FormField == FieldPrompts {
			m.FormField = FieldBranch
		} else {
			m.FormField++
		}
		if m.FormField == FieldPrompts {
			m.FormPromptIdx = 0
		}
		return m, nil
	}

	if config.Matches(key, kb.Form.PrevField) {
		if m.FormField == FieldBranch {
			m.FormField = FieldPrompts
			m.FormPromptIdx = len(m.FormPrompts) - 1
		} else {
			m.FormField--
		}
		return m, nil
	}

	// Handle edit key to start editing current field
	if config.MatchesAny(key, kb.Form.EditPrompt, kb.Editor.NewLine) {
		if m.FormField == FieldPrompts {
			// For prompts, open the full editor
			m.EditorContent = m.FormPrompts[m.FormPromptIdx]
			m.EditorCursorPos = len(m.EditorContent)
			m.PreviousView = m.CurrentView
			m.CurrentView = PromptEditorView
		} else {
			// For simple fields, enter inline edit mode
			m.FormEditing = true
		}
		return m, nil
	}

	// Handle prompt-specific shortcuts (only when on prompts field)
	if m.FormField == FieldPrompts {
		switch {
		case config.Matches(key, kb.Form.AddPrompt):
			m.FormPrompts = append(m.FormPrompts, "")
			m.FormPromptIdx = len(m.FormPrompts) - 1
			return m, nil

		case config.Matches(key, kb.Form.DeletePrompt):
			if len(m.FormPrompts) > 1 {
				m.FormPrompts = append(m.FormPrompts[:m.FormPromptIdx], m.FormPrompts[m.FormPromptIdx+1:]...)
				if m.FormPromptIdx >= len(m.FormPrompts) {
					m.FormPromptIdx = len(m.FormPrompts) - 1
				}
			}
			return m, nil

		case config.Matches(key, kb.Form.ImprovePrompt):
			if !m.Improving && strings.TrimSpace(m.FormPrompts[m.FormPromptIdx]) != "" {
				return m.openImprovePromptTerminal()
			}
			return m, nil
		}
	}

	return m, nil
}

// openImprovePromptTerminal opens the terminal modal to run the improve prompt command.
func (m Model) openImprovePromptTerminal() (tea.Model, tea.Cmd) {
	m.Improving = true
	prompt := m.FormPrompts[m.FormPromptIdx]
	idx := m.FormPromptIdx

	systemPrompt := `You are a prompt rewriter. Rewrite the user's prompt to be clearer and more effective for LLMs.

CRITICAL: Output ONLY the rewritten prompt. No introductions, no explanations, no "Here is...", no markdown formatting, no quotes around it. Just the raw improved prompt text and nothing else.

Guidelines for rewriting:
- Keep the original intent
- Be more specific and explicit
- Use clear structure if helpful
- Remove vague language`

	// Create terminal modal
	m.Terminal = terminal.New(m.Config, "Improve Prompt")
	m.Terminal.Dir = m.RepoPath
	m.Terminal.SetSize(m.Width, m.Height)

	// Set callback to handle the improved prompt when terminal closes
	m.TerminalCallback = func(model *Model, output string) {
		model.Improving = false
		improved := strings.TrimSpace(output)
		if improved != "" && idx >= 0 && idx < len(model.FormPrompts) {
			model.FormPrompts[idx] = improved
		}
	}

	// Store the current view to return to
	m.PreviousView = m.CurrentView
	m.CurrentView = TerminalView

	// Start the command
	cmd := m.Terminal.RunCommand("claude", "-p", prompt, "--system-prompt", systemPrompt)
	return m, cmd
}

// handleFormEditMode handles input when editing a simple field inline.
func (m Model) handleFormEditMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	kb := m.Config.Keys()

	// Cancel exits edit mode without saving (though changes are already in the field)
	if config.Matches(key, kb.Form.Cancel) {
		m.FormEditing = false
		return m, nil
	}

	// Enter/newline confirms and exits edit mode
	if config.Matches(key, kb.Editor.NewLine) {
		m.FormEditing = false
		return m, nil
	}

	// Handle text input for the current field
	switch m.FormField {
	case FieldBranch:
		m.FormBranch = handleTextInput(m.FormBranch, msg)
	case FieldName:
		m.FormName = handleTextInput(m.FormName, msg)
	case FieldDescription:
		m.FormDescription = handleTextInput(m.FormDescription, msg)
	}

	return m, nil
}

func handleTextInput(current string, msg tea.KeyMsg) string {
	key := msg.String()
	switch key {
	case "backspace":
		if len(current) > 0 {
			return current[:len(current)-1]
		}
	case "space":
		return current + " "
	default:
		if len(key) == 1 {
			return current + key
		}
	}
	return current
}

func (m Model) saveForm() (tea.Model, tea.Cmd) {
	if m.FormName == "" {
		m.ErrMsg = "Name is required"
		return m, nil
	}
	if m.FormBranch == "" {
		m.ErrMsg = "Branch is required"
		return m, nil
	}

	var prompts []string
	for _, p := range m.FormPrompts {
		if strings.TrimSpace(p) != "" {
			prompts = append(prompts, p)
		}
	}

	if m.CurrentView == EditView && m.FormEditingTodo != nil {
		m.FormEditingTodo.Branch = m.FormBranch
		m.FormEditingTodo.Name = m.FormName
		m.FormEditingTodo.Description = m.FormDescription
		m.FormEditingTodo.Prompts = prompts
		m.FormEditingTodo.Update()

		return m, func() tea.Msg {
			if err := m.Store.UpdateTodo(m.RepoPath, m.FormEditingTodo); err != nil {
				return TodoErrorMsg{Err: err}
			}
			return TodoSavedMsg{}
		}
	}

	t := todo.NewTodo(m.FormBranch, m.FormName, m.FormDescription, prompts)
	return m, func() tea.Msg {
		if err := m.Store.AddTodo(m.RepoPath, t); err != nil {
			return TodoErrorMsg{Err: err}
		}
		return TodoSavedMsg{}
	}
}

// AutoSavePrompt saves just the updated prompt without changing view.
func (m Model) AutoSavePrompt() tea.Cmd {
	return func() tea.Msg {
		if m.FormEditingTodo == nil {
			return nil
		}
		var prompts []string
		for _, p := range m.FormPrompts {
			if strings.TrimSpace(p) != "" {
				prompts = append(prompts, p)
			}
		}
		m.FormEditingTodo.Prompts = prompts
		m.FormEditingTodo.Update()

		if err := m.Store.UpdateTodo(m.RepoPath, m.FormEditingTodo); err != nil {
			return TodoErrorMsg{Err: err}
		}
		return nil
	}
}

// ViewForm renders the create/edit form view.
func (m Model) ViewForm(title string) string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("  " + title))
	if m.FormEditing {
		b.WriteString(styles.Confirm.Render("  [EDITING]"))
	}
	b.WriteString("\n\n")

	// Branch field
	b.WriteString(m.renderFormField("Branch", m.FormBranch, FieldBranch))

	// Name field
	b.WriteString(m.renderFormField("Name", m.FormName, FieldName))

	// Description field
	b.WriteString(m.renderFormField("Description", m.FormDescription, FieldDescription))
	b.WriteString("\n")

	// Prompts field
	promptsLabel := "Prompts:"
	if m.FormField == FieldPrompts {
		promptsLabel = styles.Selected.Render("▸ Prompts:")
	} else {
		promptsLabel = styles.Label.Render("  Prompts:")
	}
	b.WriteString(promptsLabel)
	b.WriteString("\n")

	for i, p := range m.FormPrompts {
		prefix := "    "
		if m.FormField == FieldPrompts && i == m.FormPromptIdx {
			prefix = styles.Cursor.Render("  ▸ ")
		}
		b.WriteString(prefix)
		b.WriteString(styles.Prompt.Render(fmt.Sprintf("%d. ", i+1)))

		// Show prompt preview (truncated if long)
		displayP := p
		if len(displayP) > 50 {
			displayP = displayP[:47] + "..."
		}
		// Replace newlines with spaces for display
		displayP = strings.ReplaceAll(displayP, "\n", " ")
		b.WriteString(styles.Input.Render(displayP))

		if m.FormField == FieldPrompts && i == m.FormPromptIdx {
			if m.Improving {
				b.WriteString(styles.Confirm.Render(" improving..."))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	kb := m.Config.Keys()

	var help string
	if m.FormEditing {
		// Edit mode help
		help = fmt.Sprintf("type to edit • %s confirm • %s cancel",
			kb.Editor.NewLine, kb.Form.Cancel)
	} else if m.FormField == FieldPrompts {
		// Prompts navigation help
		help = fmt.Sprintf("%s/%s nav • %s edit • %s improve • %s add • %s del • %s save",
			kb.Global.MoveUp, kb.Global.MoveDown, kb.Form.EditPrompt,
			kb.Form.ImprovePrompt, kb.Form.AddPrompt, kb.Form.DeletePrompt, kb.Form.Submit)
	} else {
		// Field navigation help
		help = fmt.Sprintf("%s/%s navigate • %s edit • %s save • %s cancel",
			kb.Global.MoveUp, kb.Global.MoveDown, kb.Form.EditPrompt, kb.Form.Submit, kb.Form.Cancel)
	}
	b.WriteString(styles.Help.Render(help))

	return b.String()
}

// renderFormField renders a single form field with appropriate styling.
func (m Model) renderFormField(label, value string, field FormField) string {
	var b strings.Builder

	isSelected := m.FormField == field
	isEditing := isSelected && m.FormEditing

	// Label
	if isSelected {
		b.WriteString(styles.Selected.Render(fmt.Sprintf("▸ %s: ", label)))
	} else {
		b.WriteString(styles.Label.Render(fmt.Sprintf("  %s: ", label)))
	}

	// Value
	b.WriteString(styles.Input.Render(value))

	// Cursor (only show when editing this field)
	if isEditing {
		b.WriteString(styles.Cursor.Render("█"))
	}

	b.WriteString("\n")
	return b.String()
}
