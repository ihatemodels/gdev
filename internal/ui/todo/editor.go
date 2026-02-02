package todo

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ihatemodels/gdev/internal/config"
	"github.com/ihatemodels/gdev/internal/ui/styles"
)

// UpdatePromptEditor handles input for the prompt editor view.
func (m Model) UpdatePromptEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	kb := m.Config.Keys()

	// Handle arrow key navigation
	switch msg.Type {
	case tea.KeyUp:
		m.EditorCursorPos = m.moveCursorVertical(-1)
		return m, nil
	case tea.KeyDown:
		m.EditorCursorPos = m.moveCursorVertical(1)
		return m, nil
	case tea.KeyLeft:
		if m.EditorCursorPos > 0 {
			m.EditorCursorPos--
		}
		return m, nil
	case tea.KeyRight:
		if m.EditorCursorPos < len(m.EditorContent) {
			m.EditorCursorPos++
		}
		return m, nil
	case tea.KeyHome:
		for m.EditorCursorPos > 0 && m.EditorContent[m.EditorCursorPos-1] != '\n' {
			m.EditorCursorPos--
		}
		return m, nil
	case tea.KeyEnd:
		for m.EditorCursorPos < len(m.EditorContent) && m.EditorContent[m.EditorCursorPos] != '\n' {
			m.EditorCursorPos++
		}
		return m, nil
	}

	// Handle configurable keybindings
	switch {
	case config.Matches(key, kb.Editor.Cancel):
		m.CurrentView = m.PreviousView
		return m, nil

	case config.Matches(key, kb.Editor.Save):
		m.FormPrompts[m.FormPromptIdx] = m.EditorContent
		m.CurrentView = m.PreviousView
		return m, nil

	case config.Matches(key, kb.Editor.NewLine):
		m.EditorContent = m.EditorContent[:m.EditorCursorPos] + "\n" + m.EditorContent[m.EditorCursorPos:]
		m.EditorCursorPos++
		return m, nil

	case key == "backspace":
		if m.EditorCursorPos > 0 {
			m.EditorContent = m.EditorContent[:m.EditorCursorPos-1] + m.EditorContent[m.EditorCursorPos:]
			m.EditorCursorPos--
		}

	case key == "delete":
		if m.EditorCursorPos < len(m.EditorContent) {
			m.EditorContent = m.EditorContent[:m.EditorCursorPos] + m.EditorContent[m.EditorCursorPos+1:]
		}

	case config.Matches(key, kb.Editor.LineStart):
		for m.EditorCursorPos > 0 && m.EditorContent[m.EditorCursorPos-1] != '\n' {
			m.EditorCursorPos--
		}

	case config.Matches(key, kb.Editor.LineEnd):
		for m.EditorCursorPos < len(m.EditorContent) && m.EditorContent[m.EditorCursorPos] != '\n' {
			m.EditorCursorPos++
		}

	case key == "space":
		m.EditorContent = m.EditorContent[:m.EditorCursorPos] + " " + m.EditorContent[m.EditorCursorPos:]
		m.EditorCursorPos++

	case key == "tab":
		m.EditorContent = m.EditorContent[:m.EditorCursorPos] + "    " + m.EditorContent[m.EditorCursorPos:]
		m.EditorCursorPos += 4

	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			m.EditorContent = m.EditorContent[:m.EditorCursorPos] + key + m.EditorContent[m.EditorCursorPos:]
			m.EditorCursorPos++
		}
	}

	return m, nil
}

func (m Model) moveCursorVertical(direction int) int {
	lines := strings.Split(m.EditorContent, "\n")

	pos := 0
	currentLine := 0
	currentCol := 0
	for i, line := range lines {
		if pos+len(line) >= m.EditorCursorPos {
			currentLine = i
			currentCol = m.EditorCursorPos - pos
			break
		}
		pos += len(line) + 1
	}

	targetLine := currentLine + direction
	if targetLine < 0 {
		targetLine = 0
	}
	if targetLine >= len(lines) {
		targetLine = len(lines) - 1
	}

	newPos := 0
	for i := 0; i < targetLine; i++ {
		newPos += len(lines[i]) + 1
	}
	targetCol := currentCol
	if targetCol > len(lines[targetLine]) {
		targetCol = len(lines[targetLine])
	}
	newPos += targetCol

	return newPos
}

// ViewPromptEditor renders the prompt editor view.
func (m Model) ViewPromptEditor() string {
	var b strings.Builder

	editorWidth := m.Width - 8
	if editorWidth < 40 {
		editorWidth = 40
	}
	if editorWidth > 120 {
		editorWidth = 120
	}
	editorHeight := m.Height - 12
	if editorHeight < 10 {
		editorHeight = 10
	}
	contentWidth := editorWidth - 4

	// Header
	b.WriteString(styles.Title.Render("  Edit Prompt"))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render(strings.Repeat("─", editorWidth+4)))
	b.WriteString("\n\n")

	// Top border
	b.WriteString(styles.Help.Render("  ┌" + strings.Repeat("─", editorWidth) + "┐"))
	b.WriteString("\n")

	// Create display lines with wrapping
	displayLines, cursorDisplayLine, cursorDisplayCol := m.wrapEditorContent(contentWidth)

	// Calculate viewport
	startLine := 0
	if cursorDisplayLine >= editorHeight {
		startLine = cursorDisplayLine - editorHeight + 1
	}

	// Render lines
	for i := 0; i < editorHeight; i++ {
		lineIdx := startLine + i
		b.WriteString(styles.Help.Render("  │ "))

		if lineIdx < len(displayLines) {
			line := displayLines[lineIdx]

			if lineIdx == cursorDisplayLine {
				b.WriteString(m.renderLineWithCursor(line, cursorDisplayCol, contentWidth))
			} else {
				b.WriteString(styles.Input.Render(line))
				padding := contentWidth - len(line)
				if padding > 0 {
					b.WriteString(strings.Repeat(" ", padding))
				}
			}
		} else {
			b.WriteString(strings.Repeat(" ", contentWidth))
		}

		b.WriteString(styles.Help.Render(" │"))
		b.WriteString("\n")
	}

	// Bottom border
	b.WriteString(styles.Help.Render("  └" + strings.Repeat("─", editorWidth) + "┘"))
	b.WriteString("\n")

	// Character count
	b.WriteString(styles.Help.Render(fmt.Sprintf("  %d characters", len(m.EditorContent))))
	b.WriteString("\n\n")

	// Help text
	kb := m.Config.Keys()
	b.WriteString(styles.Help.Render(fmt.Sprintf("%s new line • %s save • %s cancel",
		kb.Editor.NewLine, kb.Editor.Save, kb.Editor.Cancel)))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render(fmt.Sprintf("←/→ move • ↑/↓ line • %s/%s line start/end",
		kb.Editor.LineStart, kb.Editor.LineEnd)))

	return b.String()
}

func (m Model) wrapEditorContent(contentWidth int) ([]string, int, int) {
	var displayLines []string
	var cursorDisplayLine, cursorDisplayCol int

	currentLine := ""
	charIdx := 0
	cursorFound := false

	for i, ch := range m.EditorContent {
		if ch == '\n' {
			for len(currentLine) > contentWidth {
				displayLines = append(displayLines, currentLine[:contentWidth])
				if !cursorFound && charIdx <= m.EditorCursorPos && m.EditorCursorPos <= charIdx+contentWidth {
					cursorDisplayLine = len(displayLines) - 1
					cursorDisplayCol = m.EditorCursorPos - charIdx
					cursorFound = true
				}
				charIdx += contentWidth
				currentLine = currentLine[contentWidth:]
			}
			if !cursorFound && i >= m.EditorCursorPos {
				cursorDisplayLine = len(displayLines)
				cursorDisplayCol = m.EditorCursorPos - charIdx
				cursorFound = true
			}
			displayLines = append(displayLines, currentLine)
			charIdx = i + 1
			currentLine = ""
		} else {
			currentLine += string(ch)
		}
	}

	for len(currentLine) > contentWidth {
		displayLines = append(displayLines, currentLine[:contentWidth])
		if !cursorFound && charIdx <= m.EditorCursorPos && m.EditorCursorPos < charIdx+contentWidth {
			cursorDisplayLine = len(displayLines) - 1
			cursorDisplayCol = m.EditorCursorPos - charIdx
			cursorFound = true
		}
		charIdx += contentWidth
		currentLine = currentLine[contentWidth:]
	}
	displayLines = append(displayLines, currentLine)
	if !cursorFound {
		cursorDisplayLine = len(displayLines) - 1
		cursorDisplayCol = m.EditorCursorPos - charIdx
		if cursorDisplayCol < 0 {
			cursorDisplayCol = 0
		}
		if cursorDisplayCol > len(currentLine) {
			cursorDisplayCol = len(currentLine)
		}
	}

	return displayLines, cursorDisplayLine, cursorDisplayCol
}

func (m Model) renderLineWithCursor(line string, cursorCol, contentWidth int) string {
	var b strings.Builder

	if cursorCol <= len(line) {
		if cursorCol > 0 {
			b.WriteString(styles.Input.Render(line[:cursorCol]))
		}
		b.WriteString(styles.Cursor.Render("█"))
		if cursorCol < len(line) {
			b.WriteString(styles.Input.Render(line[cursorCol:]))
		}
		padding := contentWidth - len(line) - 1
		if padding > 0 {
			b.WriteString(strings.Repeat(" ", padding))
		}
	} else {
		b.WriteString(styles.Input.Render(line))
		padding := contentWidth - len(line) - 1
		if padding > 0 {
			b.WriteString(strings.Repeat(" ", padding))
		}
		b.WriteString(styles.Cursor.Render("█"))
	}

	return b.String()
}
