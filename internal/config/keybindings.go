package config

import (
	"errors"
	"strings"

	"github.com/ihatemodels/gdev/internal/store"
)

const keybindingsFile = "keybindings.json"

// Keybindings holds all configurable keyboard shortcuts.
// Keys use Bubble Tea key string format (e.g., "ctrl+s", "enter", "esc").
type Keybindings struct {
	// Global keybindings (work in most views)
	Global GlobalKeys `json:"global"`

	// List view keybindings
	List ListKeys `json:"list"`

	// Form keybindings (create/edit views)
	Form FormKeys `json:"form"`

	// Multi-line editor keybindings
	Editor EditorKeys `json:"editor"`

	// Detail view keybindings
	Detail DetailKeys `json:"detail"`
}

// GlobalKeys are keybindings that work across multiple views.
type GlobalKeys struct {
	Quit      string `json:"quit"`       // Quit/back
	QuitAlt   string `json:"quit_alt"`   // Alternative quit key
	Help      string `json:"help"`       // Show help
	MoveUp    string `json:"move_up"`    // Move cursor up
	MoveDown  string `json:"move_down"`  // Move cursor down
	MoveUpAlt string `json:"move_up_alt"`   // Alternative move up (arrow key)
	MoveDownAlt string `json:"move_down_alt"` // Alternative move down (arrow key)
}

// ListKeys are keybindings for list views.
type ListKeys struct {
	Select     string `json:"select"`       // Select/enter item
	New        string `json:"new"`          // Create new item
	Delete     string `json:"delete"`       // Delete item
	Edit       string `json:"edit"`         // Edit item
	Top        string `json:"top"`          // Jump to top
	Bottom     string `json:"bottom"`       // Jump to bottom
	PageUp     string `json:"page_up"`      // Page up
	PageDown   string `json:"page_down"`    // Page down
}

// FormKeys are keybindings for form/input views.
type FormKeys struct {
	Submit       string `json:"submit"`         // Submit form
	Cancel       string `json:"cancel"`         // Cancel form
	NextField    string `json:"next_field"`     // Move to next field
	PrevField    string `json:"prev_field"`     // Move to previous field
	AddPrompt    string `json:"add_prompt"`     // Add new prompt
	DeletePrompt string `json:"delete_prompt"`  // Delete current prompt
	EditPrompt   string `json:"edit_prompt"`    // Open prompt editor
	ImprovePrompt string `json:"improve_prompt"` // Improve prompt with AI
}

// EditorKeys are keybindings for the multi-line text editor.
type EditorKeys struct {
	Save         string `json:"save"`           // Save and exit editor
	Cancel       string `json:"cancel"`         // Cancel editing
	LineStart    string `json:"line_start"`     // Move to line start
	LineEnd      string `json:"line_end"`       // Move to line end
	DeleteLine   string `json:"delete_line"`    // Delete current line
	NewLine      string `json:"new_line"`       // Insert new line
}

// DetailKeys are keybindings for detail/view screens.
type DetailKeys struct {
	Back         string `json:"back"`           // Go back
	Edit         string `json:"edit"`           // Edit item
	Delete       string `json:"delete"`         // Delete item
	ScrollUp     string `json:"scroll_up"`      // Scroll up
	ScrollDown   string `json:"scroll_down"`    // Scroll down
}

// DefaultKeybindings returns the default keybinding configuration.
func DefaultKeybindings() *Keybindings {
	return &Keybindings{
		Global: GlobalKeys{
			Quit:        "esc",
			QuitAlt:     "q",
			Help:        "?",
			MoveUp:      "k",
			MoveDown:    "j",
			MoveUpAlt:   "up",
			MoveDownAlt: "down",
		},
		List: ListKeys{
			Select:   "enter",
			New:      "n",
			Delete:   "d",
			Edit:     "e",
			Top:      "g",
			Bottom:   "G",
			PageUp:   "ctrl+u",
			PageDown: "ctrl+d",
		},
		Form: FormKeys{
			Submit:        "ctrl+s",
			Cancel:        "esc",
			NextField:     "tab",
			PrevField:     "shift+tab",
			AddPrompt:     "ctrl+a",
			DeletePrompt:  "ctrl+d",
			EditPrompt:    "ctrl+e",
			ImprovePrompt: "ctrl+i",
		},
		Editor: EditorKeys{
			Save:       "ctrl+s",
			Cancel:     "esc",
			LineStart:  "ctrl+a",
			LineEnd:    "ctrl+e",
			DeleteLine: "ctrl+k",
			NewLine:    "enter",
		},
		Detail: DetailKeys{
			Back:       "esc",
			Edit:       "e",
			Delete:     "d",
			ScrollUp:   "k",
			ScrollDown: "j",
		},
	}
}

// LoadKeybindings loads keybindings from the store.
// If the keybindings file doesn't exist, it creates one with defaults.
func LoadKeybindings(s *store.Store) (*Keybindings, error) {
	var kb Keybindings

	err := s.ReadJSON(keybindingsFile, &kb)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// File doesn't exist, create with defaults
			kb = *DefaultKeybindings()
			if err := SaveKeybindings(s, &kb); err != nil {
				return nil, err
			}
			return &kb, nil
		}
		return nil, err
	}

	// Merge with defaults to ensure new fields are populated
	kb = mergeWithDefaults(&kb)

	return &kb, nil
}

// SaveKeybindings saves keybindings to the store.
func SaveKeybindings(s *store.Store, kb *Keybindings) error {
	return s.WriteJSON(keybindingsFile, kb)
}

// mergeWithDefaults fills in any missing keybindings with defaults.
// This handles cases where new keybindings are added in updates.
func mergeWithDefaults(kb *Keybindings) Keybindings {
	defaults := DefaultKeybindings()
	result := *kb

	// Global
	if result.Global.Quit == "" {
		result.Global.Quit = defaults.Global.Quit
	}
	if result.Global.QuitAlt == "" {
		result.Global.QuitAlt = defaults.Global.QuitAlt
	}
	if result.Global.Help == "" {
		result.Global.Help = defaults.Global.Help
	}
	if result.Global.MoveUp == "" {
		result.Global.MoveUp = defaults.Global.MoveUp
	}
	if result.Global.MoveDown == "" {
		result.Global.MoveDown = defaults.Global.MoveDown
	}
	if result.Global.MoveUpAlt == "" {
		result.Global.MoveUpAlt = defaults.Global.MoveUpAlt
	}
	if result.Global.MoveDownAlt == "" {
		result.Global.MoveDownAlt = defaults.Global.MoveDownAlt
	}

	// List
	if result.List.Select == "" {
		result.List.Select = defaults.List.Select
	}
	if result.List.New == "" {
		result.List.New = defaults.List.New
	}
	if result.List.Delete == "" {
		result.List.Delete = defaults.List.Delete
	}
	if result.List.Edit == "" {
		result.List.Edit = defaults.List.Edit
	}
	if result.List.Top == "" {
		result.List.Top = defaults.List.Top
	}
	if result.List.Bottom == "" {
		result.List.Bottom = defaults.List.Bottom
	}
	if result.List.PageUp == "" {
		result.List.PageUp = defaults.List.PageUp
	}
	if result.List.PageDown == "" {
		result.List.PageDown = defaults.List.PageDown
	}

	// Form
	if result.Form.Submit == "" {
		result.Form.Submit = defaults.Form.Submit
	}
	if result.Form.Cancel == "" {
		result.Form.Cancel = defaults.Form.Cancel
	}
	if result.Form.NextField == "" {
		result.Form.NextField = defaults.Form.NextField
	}
	if result.Form.PrevField == "" {
		result.Form.PrevField = defaults.Form.PrevField
	}
	if result.Form.AddPrompt == "" {
		result.Form.AddPrompt = defaults.Form.AddPrompt
	}
	if result.Form.DeletePrompt == "" {
		result.Form.DeletePrompt = defaults.Form.DeletePrompt
	}
	if result.Form.EditPrompt == "" {
		result.Form.EditPrompt = defaults.Form.EditPrompt
	}
	if result.Form.ImprovePrompt == "" {
		result.Form.ImprovePrompt = defaults.Form.ImprovePrompt
	}

	// Editor
	if result.Editor.Save == "" {
		result.Editor.Save = defaults.Editor.Save
	}
	if result.Editor.Cancel == "" {
		result.Editor.Cancel = defaults.Editor.Cancel
	}
	if result.Editor.LineStart == "" {
		result.Editor.LineStart = defaults.Editor.LineStart
	}
	if result.Editor.LineEnd == "" {
		result.Editor.LineEnd = defaults.Editor.LineEnd
	}
	if result.Editor.DeleteLine == "" {
		result.Editor.DeleteLine = defaults.Editor.DeleteLine
	}
	if result.Editor.NewLine == "" {
		result.Editor.NewLine = defaults.Editor.NewLine
	}

	// Detail
	if result.Detail.Back == "" {
		result.Detail.Back = defaults.Detail.Back
	}
	if result.Detail.Edit == "" {
		result.Detail.Edit = defaults.Detail.Edit
	}
	if result.Detail.Delete == "" {
		result.Detail.Delete = defaults.Detail.Delete
	}
	if result.Detail.ScrollUp == "" {
		result.Detail.ScrollUp = defaults.Detail.ScrollUp
	}
	if result.Detail.ScrollDown == "" {
		result.Detail.ScrollDown = defaults.Detail.ScrollDown
	}

	return result
}

// Matches checks if a key string matches a keybinding.
// It handles shift+letter bindings by converting them to uppercase.
// For example, "shift+a" in config matches "A" from Bubble Tea.
func Matches(key string, binding string) bool {
	return key == normalizeBinding(binding)
}

// MatchesAny checks if a key matches any of the provided bindings.
func MatchesAny(key string, bindings ...string) bool {
	for _, b := range bindings {
		if key == normalizeBinding(b) {
			return true
		}
	}
	return false
}

// normalizeBinding converts a binding string to match Bubble Tea's key format.
// Specifically, "shift+x" becomes "X" for letter keys.
func normalizeBinding(binding string) string {
	// Handle shift+letter -> uppercase letter
	if strings.HasPrefix(binding, "shift+") {
		letter := strings.TrimPrefix(binding, "shift+")
		if len(letter) == 1 && letter[0] >= 'a' && letter[0] <= 'z' {
			return strings.ToUpper(letter)
		}
	}
	return binding
}
