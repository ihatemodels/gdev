// Package terminal provides a reusable popup terminal modal for running commands.
package terminal

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ihatemodels/gdev/internal/config"
	"github.com/ihatemodels/gdev/internal/ui/styles"
)

// TickMsg triggers a UI refresh to show new output lines.
type TickMsg struct {
	ID int
}

// CommandDoneMsg is sent when the command finishes.
type CommandDoneMsg struct {
	Err    error
	ID     int
	Output []string // all output lines
}

// sharedOutput holds output lines that can be safely accessed from goroutines.
type sharedOutput struct {
	mu    sync.Mutex
	lines []string
	done  bool
	err   error
}

func (s *sharedOutput) addLine(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lines = append(s.lines, line)
}

func (s *sharedOutput) getLines() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, len(s.lines))
	copy(result, s.lines)
	return result
}

func (s *sharedOutput) setDone(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.done = true
	s.err = err
}

func (s *sharedOutput) isDone() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done, s.err
}

// Model represents a terminal popup modal.
type Model struct {
	ID       int    // unique identifier for this terminal instance
	Title    string // title shown in the modal header
	Command  string // the command being run (for display)
	Dir      string // working directory for the command

	Lines      []string // output lines
	ScrollPos  int      // current scroll position
	MaxLines   int      // max lines to keep in buffer
	Running    bool     // true if command is still running
	Err        error    // error from command execution
	AutoScroll bool     // auto-scroll to bottom on new lines

	Width  int // modal width
	Height int // modal height

	Config *config.Config

	// Internal state for streaming
	output *sharedOutput
}

var instanceCounter int

// New creates a new terminal model.
func New(cfg *config.Config, title string) Model {
	instanceCounter++
	return Model{
		ID:         instanceCounter,
		Title:      title,
		Config:     cfg,
		Lines:      []string{},
		MaxLines:   1000,
		Width:      80,
		Height:     20,
		AutoScroll: true,
	}
}

// SetSize sets the modal dimensions.
func (m *Model) SetSize(width, height int) {
	// Modal takes up 80% of screen, with min/max bounds
	m.Width = width * 80 / 100
	if m.Width < 60 {
		m.Width = 60
	}
	if m.Width > 120 {
		m.Width = 120
	}

	m.Height = height * 70 / 100
	if m.Height < 10 {
		m.Height = 10
	}
	if m.Height > 40 {
		m.Height = 40
	}
}

// RunCommand starts executing a command and streams output.
func (m *Model) RunCommand(name string, args ...string) tea.Cmd {
	return m.RunCommandWithEnv(nil, name, args...)
}

// RunCommandWithEnv starts executing a command with environment variables.
func (m *Model) RunCommandWithEnv(env []string, name string, args ...string) tea.Cmd {
	m.Command = name + " " + strings.Join(args, " ")
	m.Running = true
	m.Lines = []string{styles.Help.Render("$ " + m.Command), ""}
	m.ScrollPos = 0
	m.Err = nil
	m.output = &sharedOutput{lines: []string{}}

	dir := m.Dir
	output := m.output

	// Start the command in a goroutine
	go func() {
		err := executeCommandStreaming(dir, env, output, name, args...)
		output.setDone(err)
	}()

	// Return a tick command to start polling for output
	return m.tick()
}

func (m Model) tick() tea.Cmd {
	id := m.ID
	return tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg {
		return TickMsg{ID: id}
	})
}

func executeCommandStreaming(dir string, env []string, output *sharedOutput, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if env != nil {
		cmd.Env = env
	}

	// Get pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Read both stdout and stderr
	var wg sync.WaitGroup
	wg.Add(2)

	go readPipeToOutput(stdout, output, &wg)
	go readPipeToOutput(stderr, output, &wg)

	wg.Wait()

	return cmd.Wait()
}

func readPipeToOutput(pipe io.ReadCloser, output *sharedOutput, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		output.addLine(scanner.Text())
	}
}

// Update handles input for the terminal modal.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil

	case TickMsg:
		if msg.ID != m.ID {
			return m, nil
		}
		return m.handleTick()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleTick() (Model, tea.Cmd) {
	if m.output == nil {
		return m, nil
	}

	// Get latest lines from shared output
	newLines := m.output.getLines()

	// Update lines, keeping the command header
	if len(newLines) > 0 {
		m.Lines = append([]string{styles.Help.Render("$ " + m.Command), ""}, newLines...)
	}

	// Trim to max lines
	if len(m.Lines) > m.MaxLines {
		m.Lines = m.Lines[len(m.Lines)-m.MaxLines:]
	}

	// Auto-scroll to bottom if enabled
	if m.AutoScroll {
		m.ScrollPos = m.maxScroll()
	}

	// Check if command is done
	done, err := m.output.isDone()
	if done {
		m.Running = false
		m.Err = err
		if err != nil {
			m.Lines = append(m.Lines, "")
			m.Lines = append(m.Lines, styles.Error.Render("Error: "+err.Error()))
		} else {
			m.Lines = append(m.Lines, "")
			m.Lines = append(m.Lines, styles.Selected.Render("✓ Command completed"))
		}
		if m.AutoScroll {
			m.ScrollPos = m.maxScroll()
		}
		return m, nil
	}

	// Continue ticking while running
	return m, m.tick()
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()
	kb := m.Config.Keys()

	// Disable auto-scroll when user scrolls manually
	if config.MatchesAny(key, kb.Global.MoveUp, kb.Global.MoveUpAlt) || msg.Type == tea.KeyUp {
		m.AutoScroll = false
		if m.ScrollPos > 0 {
			m.ScrollPos--
		}
		return m, nil
	}

	if config.MatchesAny(key, kb.Global.MoveDown, kb.Global.MoveDownAlt) || msg.Type == tea.KeyDown {
		if m.ScrollPos < m.maxScroll() {
			m.ScrollPos++
		}
		// Re-enable auto-scroll if at bottom
		if m.ScrollPos >= m.maxScroll() {
			m.AutoScroll = true
		}
		return m, nil
	}

	// Page up/down
	if config.Matches(key, kb.List.PageUp) {
		m.AutoScroll = false
		m.ScrollPos -= m.visibleLines()
		if m.ScrollPos < 0 {
			m.ScrollPos = 0
		}
		return m, nil
	}

	if config.Matches(key, kb.List.PageDown) {
		m.ScrollPos += m.visibleLines()
		if m.ScrollPos > m.maxScroll() {
			m.ScrollPos = m.maxScroll()
		}
		if m.ScrollPos >= m.maxScroll() {
			m.AutoScroll = true
		}
		return m, nil
	}

	// Jump to top/bottom
	if config.Matches(key, kb.List.Top) {
		m.AutoScroll = false
		m.ScrollPos = 0
		return m, nil
	}

	if config.Matches(key, kb.List.Bottom) {
		m.ScrollPos = m.maxScroll()
		m.AutoScroll = true
		return m, nil
	}

	return m, nil
}

// ShouldClose returns true if the user pressed a quit key.
func (m Model) ShouldClose(msg tea.KeyMsg) bool {
	key := msg.String()
	kb := m.Config.Keys()
	return config.MatchesAny(key, kb.Global.Quit, kb.Global.QuitAlt)
}

// GetOutput returns all display lines (excluding the command header).
// Note: This includes status messages. Use GetRawOutput() for just command output.
func (m Model) GetOutput() string {
	if len(m.Lines) <= 2 {
		return ""
	}
	// Skip the command header lines
	return strings.Join(m.Lines[2:], "\n")
}

// GetRawOutput returns only the raw command output without any status messages.
// This is useful when you need to process the actual command output.
func (m Model) GetRawOutput() string {
	if m.output == nil {
		return ""
	}
	lines := m.output.getLines()
	return strings.Join(lines, "\n")
}

// GetRawOutputLines returns the raw command output lines as a slice.
func (m Model) GetRawOutputLines() []string {
	if m.output == nil {
		return nil
	}
	return m.output.getLines()
}

// GetOutputLines returns the display output lines as a slice.
// Note: This includes status messages. Use GetRawOutputLines() for just command output.
func (m Model) GetOutputLines() []string {
	if len(m.Lines) <= 2 {
		return nil
	}
	return m.Lines[2:]
}

func (m Model) visibleLines() int {
	// Account for borders and header/footer
	return m.Height - 6
}

func (m Model) maxScroll() int {
	max := len(m.Lines) - m.visibleLines()
	if max < 0 {
		return 0
	}
	return max
}

// View renders the terminal modal.
func (m Model) View() string {
	// Calculate content area
	contentWidth := m.Width - 4 // borders + padding
	visibleLines := m.visibleLines()

	// Build header
	status := styles.Selected.Render("✓ Done")
	if m.Running {
		status = styles.Confirm.Render("● Running...")
	} else if m.Err != nil {
		status = styles.Error.Render("✗ Failed")
	}

	titleText := m.Title
	if len(titleText) > contentWidth-15 {
		titleText = titleText[:contentWidth-18] + "..."
	}

	header := fmt.Sprintf(" %s  %s", styles.Title.Render(titleText), status)

	// Build content
	var content strings.Builder

	// Get visible lines
	start := m.ScrollPos
	end := start + visibleLines
	if end > len(m.Lines) {
		end = len(m.Lines)
	}

	for i := start; i < end; i++ {
		line := m.Lines[i]
		// Truncate long lines
		if len(line) > contentWidth {
			line = line[:contentWidth-3] + "..."
		}
		content.WriteString(line)
		if i < end-1 {
			content.WriteString("\n")
		}
	}

	// Pad with empty lines if needed
	for i := end - start; i < visibleLines; i++ {
		content.WriteString("\n")
	}

	// Build footer with help text
	kb := m.Config.Keys()
	scrollInfo := fmt.Sprintf(" %d/%d ", m.ScrollPos+1, max(len(m.Lines), 1))
	helpText := fmt.Sprintf("%s/%s scroll • %s/%s page • %s close",
		kb.Global.MoveUp, kb.Global.MoveDown,
		kb.List.PageUp, kb.List.PageDown,
		kb.Global.Quit)
	footer := styles.Help.Render(scrollInfo + " │ " + helpText)

	// Create the modal box
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Purple).
		Padding(0, 1).
		Width(m.Width)

	contentStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Height(visibleLines)

	box := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		styles.Help.Render(strings.Repeat("─", contentWidth)),
		contentStyle.Render(content.String()),
		styles.Help.Render(strings.Repeat("─", contentWidth)),
		footer,
	)

	return borderStyle.Render(box)
}

// ViewCentered renders the terminal modal centered on screen.
func (m Model) ViewCentered(screenWidth, screenHeight int) string {
	modal := m.View()

	// Center horizontally and vertically
	return lipgloss.Place(
		screenWidth,
		screenHeight,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")),
	)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
