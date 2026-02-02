// Package commit provides a TUI component for smart commits using Claude.
package commit

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ihatemodels/gdev/internal/config"
	"github.com/ihatemodels/gdev/internal/ui/styles"
	"github.com/ihatemodels/gdev/internal/ui/terminal"
)

// State represents the current state of the commit flow.
type State int

const (
	StateChecking State = iota
	StateNoChanges
	StateGenerating
	StateEditing
	StateCommitting
	StateDone
	StateError
)

// BackToMenuMsg signals that we should return to the main menu.
type BackToMenuMsg struct{}

// CommitDoneMsg signals that the commit completed.
type CommitDoneMsg struct {
	Err error
}

// CheckDoneMsg signals that the check for changes completed.
type CheckDoneMsg struct {
	HasChanges bool
	Diff       string
	Err        error
}

// Model represents the commit view state.
type Model struct {
	Config   *config.Config
	RepoPath string

	State    State
	ErrMsg   string
	Diff     string // git diff output for context

	// Commit message editing
	Subject       string // first line
	Body          string // rest of the message
	EditingField  int    // 0 = subject, 1 = body
	CursorPos     int    // cursor position within current field
	BodyScrollPos int    // scroll position in body

	// Terminal for running commands
	Terminal terminal.Model

	Width  int
	Height int
}

// New creates a new commit model.
func New(cfg *config.Config, repoPath string) Model {
	return Model{
		Config:   cfg,
		RepoPath: repoPath,
		State:    StateChecking,
	}
}

// SetSize sets the dimensions for the view.
func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.checkForChanges()
}

func (m Model) checkForChanges() tea.Cmd {
	repoPath := m.RepoPath
	return func() tea.Msg {
		// Check if there are any changes
		cmd := exec.Command("git", "status", "--porcelain")
		cmd.Dir = repoPath
		out, err := cmd.Output()
		if err != nil {
			return CheckDoneMsg{Err: err}
		}

		hasChanges := len(strings.TrimSpace(string(out))) > 0
		if !hasChanges {
			return CheckDoneMsg{HasChanges: false}
		}

		// Get the diff for context
		diffCmd := exec.Command("git", "diff", "HEAD")
		diffCmd.Dir = repoPath
		diffOut, _ := diffCmd.Output()

		return CheckDoneMsg{HasChanges: true, Diff: string(diffOut)}
	}
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Terminal.SetSize(msg.Width, msg.Height)
		return m, nil

	case CheckDoneMsg:
		if msg.Err != nil {
			m.State = StateError
			m.ErrMsg = msg.Err.Error()
			return m, nil
		}
		if !msg.HasChanges {
			m.State = StateNoChanges
			return m, nil
		}
		m.Diff = msg.Diff
		return m.startGenerating()

	case terminal.TickMsg:
		if m.State == StateGenerating || m.State == StateCommitting {
			var cmd tea.Cmd
			m.Terminal, cmd = m.Terminal.Update(msg)

			// Check if done
			if !m.Terminal.Running {
				if m.State == StateGenerating {
					return m.handleGenerateDone()
				} else if m.State == StateCommitting {
					return m.handleCommitDone()
				}
			}
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) startGenerating() (Model, tea.Cmd) {
	m.State = StateGenerating
	m.Terminal = terminal.New(m.Config, "Generating commit message...")
	m.Terminal.Dir = m.RepoPath
	m.Terminal.SetSize(m.Width, m.Height)

	// Run claude with the generate-commit-msg skill
	cmd := m.Terminal.RunCommand("claude", "-p", "/generate-commit-msg")
	return m, cmd
}

func (m Model) handleGenerateDone() (Model, tea.Cmd) {
	if m.Terminal.Err != nil {
		m.State = StateError
		m.ErrMsg = "Failed to generate commit message: " + m.Terminal.Err.Error()
		return m, nil
	}

	// Parse the output into subject and body
	output := strings.TrimSpace(m.Terminal.GetRawOutput())

	// Extract the actual commit message from Claude's response
	subject, body := parseCommitMessage(output)

	m.Subject = subject
	m.Body = body

	m.State = StateEditing
	m.EditingField = 0
	m.CursorPos = len(m.Subject)

	return m, nil
}

// parseCommitMessage extracts a commit message from Claude's output.
// It handles markdown code blocks and preamble text.
func parseCommitMessage(output string) (subject, body string) {
	lines := strings.Split(output, "\n")

	// Commit type prefixes to look for
	prefixes := []string{"feat:", "fix:", "refactor:", "docs:", "style:", "test:", "chore:"}

	// Find the line that starts with a commit type
	startIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip code block delimiters
		if strings.HasPrefix(trimmed, "```") {
			continue
		}
		// Check if line starts with a commit type
		for _, prefix := range prefixes {
			if strings.HasPrefix(strings.ToLower(trimmed), prefix) {
				startIdx = i
				break
			}
		}
		if startIdx != -1 {
			break
		}
	}

	// If no commit type found, fall back to stripping code blocks and taking first line
	if startIdx == -1 {
		cleaned := stripCodeBlocks(output)
		parts := strings.SplitN(cleaned, "\n", 2)
		subject = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			body = strings.TrimSpace(parts[1])
		}
		return
	}

	// Extract from the commit type line onwards
	var resultLines []string
	for i := startIdx; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		// Stop at code block end or obvious non-commit content
		if strings.HasPrefix(trimmed, "```") {
			continue
		}
		resultLines = append(resultLines, lines[i])
	}

	result := strings.TrimSpace(strings.Join(resultLines, "\n"))
	parts := strings.SplitN(result, "\n", 2)

	subject = strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		body = strings.TrimSpace(parts[1])
	}

	return
}

// stripCodeBlocks removes markdown code block delimiters from the output.
func stripCodeBlocks(s string) string {
	lines := strings.Split(s, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip lines that are just code block delimiters
		if strings.HasPrefix(trimmed, "```") {
			continue
		}
		result = append(result, line)
	}

	return strings.TrimSpace(strings.Join(result, "\n"))
}

func (m Model) handleCommitDone() (Model, tea.Cmd) {
	if m.Terminal.Err != nil {
		m.State = StateError
		m.ErrMsg = "Commit failed: " + m.Terminal.Err.Error()
		return m, nil
	}

	m.State = StateDone
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	kb := m.Config.Keys()

	// Global: escape to go back
	if config.MatchesAny(key, kb.Global.Quit, kb.Global.QuitAlt) {
		if m.State == StateEditing {
			// Confirm cancel?
			return m, func() tea.Msg { return BackToMenuMsg{} }
		}
		return m, func() tea.Msg { return BackToMenuMsg{} }
	}

	switch m.State {
	case StateNoChanges, StateDone, StateError:
		// Any key returns to menu
		if key == "enter" || key == " " {
			return m, func() tea.Msg { return BackToMenuMsg{} }
		}

	case StateGenerating, StateCommitting:
		// Handle terminal scrolling
		var cmd tea.Cmd
		m.Terminal, cmd = m.Terminal.Update(msg)
		return m, cmd

	case StateEditing:
		return m.handleEditKey(msg)
	}

	return m, nil
}

func (m Model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	kb := m.Config.Keys()

	// Submit commit
	if config.Matches(key, kb.Form.Submit) {
		if m.Subject == "" {
			m.ErrMsg = "Subject is required"
			return m, nil
		}
		return m.doCommit()
	}

	// Navigate between fields
	if config.Matches(key, kb.Form.NextField) || key == "down" {
		if m.EditingField == 0 {
			m.EditingField = 1
			m.CursorPos = len(m.Body)
		}
		return m, nil
	}

	if config.Matches(key, kb.Form.PrevField) || key == "up" {
		if m.EditingField == 1 {
			m.EditingField = 0
			m.CursorPos = len(m.Subject)
		}
		return m, nil
	}

	// Handle text input
	if m.EditingField == 0 {
		m.Subject, m.CursorPos = handleTextEdit(m.Subject, m.CursorPos, msg)
		// Limit subject to 72 chars
		if len(m.Subject) > 72 {
			m.Subject = m.Subject[:72]
			if m.CursorPos > 72 {
				m.CursorPos = 72
			}
		}
	} else {
		m.Body, m.CursorPos = handleTextEdit(m.Body, m.CursorPos, msg)
	}

	return m, nil
}

func handleTextEdit(text string, cursor int, msg tea.KeyMsg) (string, int) {
	key := msg.String()

	switch key {
	case "backspace":
		if cursor > 0 {
			text = text[:cursor-1] + text[cursor:]
			cursor--
		}
	case "delete":
		if cursor < len(text) {
			text = text[:cursor] + text[cursor+1:]
		}
	case "left":
		if cursor > 0 {
			cursor--
		}
	case "right":
		if cursor < len(text) {
			cursor++
		}
	case "home", "ctrl+a":
		// Go to start of current line
		for cursor > 0 && text[cursor-1] != '\n' {
			cursor--
		}
	case "end", "ctrl+e":
		// Go to end of current line
		for cursor < len(text) && text[cursor] != '\n' {
			cursor++
		}
	case "enter":
		text = text[:cursor] + "\n" + text[cursor:]
		cursor++
	case "space":
		text = text[:cursor] + " " + text[cursor:]
		cursor++
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			text = text[:cursor] + key + text[cursor:]
			cursor++
		}
	}

	return text, cursor
}

func (m Model) doCommit() (Model, tea.Cmd) {
	m.State = StateCommitting
	m.Terminal = terminal.New(m.Config, "Committing changes...")
	m.Terminal.Dir = m.RepoPath
	m.Terminal.SetSize(m.Width, m.Height)

	// Build commit message
	commitMsg := m.Subject
	if m.Body != "" {
		commitMsg += "\n\n" + m.Body
	}

	// Build the git command using HEREDOC to preserve newlines
	gitCmd := fmt.Sprintf(`git add -A && git commit -m "$(cat <<'COMMITMSG'
%s
COMMITMSG
)"`, commitMsg)

	// On Linux, ensure ssh-agent is available for commit signing
	var cmd tea.Cmd
	if runtime.GOOS == "linux" && os.Getenv("SSH_AUTH_SOCK") == "" {
		// Try to find existing ssh-agent socket or start new one
		sshSetup := findOrStartSSHAgent()
		cmd = m.Terminal.RunCommand("bash", "-c", sshSetup+gitCmd)
	} else {
		cmd = m.Terminal.RunCommand("bash", "-c", gitCmd)
	}

	return m, cmd
}

// findOrStartSSHAgent returns a bash snippet that ensures ssh-agent is available.
// It tries common socket locations before starting a new agent.
func findOrStartSSHAgent() string {
	return `
# Try to find existing ssh-agent socket
if [ -z "$SSH_AUTH_SOCK" ]; then
    # Check common socket locations
    for sock in \
        "$XDG_RUNTIME_DIR/ssh-agent.socket" \
        "$XDG_RUNTIME_DIR/keyring/ssh" \
        "$XDG_RUNTIME_DIR/gcr/ssh" \
        /tmp/ssh-*/agent.*; do
        if [ -S "$sock" ]; then
            export SSH_AUTH_SOCK="$sock"
            break
        fi
    done
fi

# If still no agent, start one and add keys
if [ -z "$SSH_AUTH_SOCK" ]; then
    eval $(ssh-agent -s) > /dev/null
    ssh-add 2>/dev/null
fi

`
}

// View implements tea.Model.
func (m Model) View() string {
	if m.Width == 0 {
		return "Loading..."
	}

	switch m.State {
	case StateChecking:
		return m.viewCentered(m.viewChecking())
	case StateNoChanges:
		return m.viewCentered(m.viewNoChanges())
	case StateGenerating:
		return m.Terminal.ViewCentered(m.Width, m.Height)
	case StateEditing:
		return m.viewCentered(m.viewEditing())
	case StateCommitting:
		return m.Terminal.ViewCentered(m.Width, m.Height)
	case StateDone:
		return m.viewCentered(m.viewDone())
	case StateError:
		return m.viewCentered(m.viewError())
	}

	return ""
}

func (m Model) viewCentered(content string) string {
	return lipgloss.Place(
		m.Width,
		m.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (m Model) viewChecking() string {
	return styles.Title.Render("  Checking for changes...")
}

func (m Model) viewNoChanges() string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("  No Changes"))
	b.WriteString("\n\n")
	b.WriteString(styles.Help.Render("There are no uncommitted changes in this repository."))
	b.WriteString("\n\n")
	b.WriteString(styles.Help.Render("Press Enter to go back"))
	return b.String()
}

func (m Model) viewEditing() string {
	var b strings.Builder
	kb := m.Config.Keys()

	b.WriteString(styles.Title.Render("  Smart Commit"))
	b.WriteString("\n\n")

	// Subject field
	subjectLabel := "Subject:"
	if m.EditingField == 0 {
		subjectLabel = styles.Selected.Render("▸ Subject:")
	} else {
		subjectLabel = styles.Label.Render("  Subject:")
	}
	b.WriteString(subjectLabel)
	b.WriteString("\n")

	// Subject input box
	boxWidth := 72
	subjectDisplay := m.Subject
	if m.EditingField == 0 {
		// Show cursor
		if m.CursorPos <= len(subjectDisplay) {
			subjectDisplay = subjectDisplay[:m.CursorPos] + "█" + subjectDisplay[m.CursorPos:]
		}
	}
	b.WriteString(styles.Help.Render("  ┌" + strings.Repeat("─", boxWidth) + "┐"))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("  │ "))
	b.WriteString(styles.Input.Render(padRight(subjectDisplay, boxWidth-2)))
	b.WriteString(styles.Help.Render(" │"))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("  └" + strings.Repeat("─", boxWidth) + "┘"))
	b.WriteString("\n")

	// Character count for subject
	charCount := fmt.Sprintf("  %d/72 characters", len(m.Subject))
	if len(m.Subject) > 50 {
		charCount = styles.Confirm.Render(charCount)
	} else {
		charCount = styles.Help.Render(charCount)
	}
	b.WriteString(charCount)
	b.WriteString("\n\n")

	// Body field
	bodyLabel := "Body (optional):"
	if m.EditingField == 1 {
		bodyLabel = styles.Selected.Render("▸ Body (optional):")
	} else {
		bodyLabel = styles.Label.Render("  Body (optional):")
	}
	b.WriteString(bodyLabel)
	b.WriteString("\n")

	// Body input box (multi-line)
	bodyHeight := 8
	bodyDisplay := m.Body
	if m.EditingField == 1 {
		// Show cursor
		if m.CursorPos <= len(bodyDisplay) {
			bodyDisplay = bodyDisplay[:m.CursorPos] + "█" + bodyDisplay[m.CursorPos:]
		}
	}
	bodyLines := strings.Split(bodyDisplay, "\n")
	for len(bodyLines) < bodyHeight {
		bodyLines = append(bodyLines, "")
	}

	b.WriteString(styles.Help.Render("  ┌" + strings.Repeat("─", boxWidth) + "┐"))
	b.WriteString("\n")
	for i := 0; i < bodyHeight && i < len(bodyLines); i++ {
		line := bodyLines[i]
		if len(line) > boxWidth-2 {
			line = line[:boxWidth-2]
		}
		b.WriteString(styles.Help.Render("  │ "))
		b.WriteString(styles.Input.Render(padRight(line, boxWidth-2)))
		b.WriteString(styles.Help.Render(" │"))
		b.WriteString("\n")
	}
	b.WriteString(styles.Help.Render("  └" + strings.Repeat("─", boxWidth) + "┘"))
	b.WriteString("\n\n")

	// Error message
	if m.ErrMsg != "" {
		b.WriteString(styles.Error.Render("  " + m.ErrMsg))
		b.WriteString("\n\n")
	}

	// Help
	b.WriteString(styles.Help.Render(fmt.Sprintf("↑/↓ or %s/%s switch fields • %s commit • %s cancel",
		kb.Form.PrevField, kb.Form.NextField, kb.Form.Submit, kb.Global.Quit)))

	return b.String()
}

func (m Model) viewDone() string {
	var b strings.Builder
	b.WriteString(styles.Selected.Render("  ✓ Commit Created"))
	b.WriteString("\n\n")
	b.WriteString(styles.Label.Render("  " + m.Subject))
	b.WriteString("\n\n")
	b.WriteString(styles.Help.Render("Press Enter to go back"))
	return b.String()
}

func (m Model) viewError() string {
	var b strings.Builder
	b.WriteString(styles.Error.Render("  ✗ Error"))
	b.WriteString("\n\n")
	b.WriteString(styles.Help.Render("  " + m.ErrMsg))
	b.WriteString("\n\n")
	b.WriteString(styles.Help.Render("Press Enter to go back"))
	return b.String()
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s[:length]
	}
	return s + strings.Repeat(" ", length-len(s))
}
