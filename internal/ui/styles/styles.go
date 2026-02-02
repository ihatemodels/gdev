// Package styles provides shared colors and styling for the TUI.
package styles

import "github.com/charmbracelet/lipgloss"

// Dracula color palette
var (
	Purple = lipgloss.Color("#BD93F9")
	Cyan   = lipgloss.Color("#8BE9FD")
	Pink   = lipgloss.Color("#FF79C6")
	Green  = lipgloss.Color("#50FA7B")
	Yellow = lipgloss.Color("#F1FA8C")
	Red    = lipgloss.Color("#FF5555")
	Subtle = lipgloss.Color("#6272A4")
	White  = lipgloss.Color("#F8F8F2")
)

// Common styles
var (
	Title = lipgloss.NewStyle().
		Foreground(Cyan).
		Bold(true)

	Item = lipgloss.NewStyle().
		Foreground(White)

	Selected = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true)

	Cursor = lipgloss.NewStyle().
		Foreground(Pink).
		Bold(true)

	Help = lipgloss.NewStyle().
		Foreground(Subtle)

	Branch = lipgloss.NewStyle().
		Foreground(Pink)

	Label = lipgloss.NewStyle().
		Foreground(Cyan).
		Bold(true)

	Value = lipgloss.NewStyle().
		Foreground(White)

	Input = lipgloss.NewStyle().
		Foreground(Yellow).
		Bold(true)

	Error = lipgloss.NewStyle().
		Foreground(Red).
		Bold(true)

	Prompt = lipgloss.NewStyle().
		Foreground(Purple)

	Confirm = lipgloss.NewStyle().
		Foreground(Yellow).
		Bold(true)

	Banner = lipgloss.NewStyle().
		Foreground(Purple).
		Bold(true)

	Version = lipgloss.NewStyle().
		Foreground(Subtle).
		Italic(true)

	Status = lipgloss.NewStyle().
		Foreground(Yellow)

	Dim = lipgloss.NewStyle().
		Foreground(Subtle)

	Repo = lipgloss.NewStyle().
		Foreground(Cyan)
)
