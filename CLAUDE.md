# CLAUDE.md

## Project Overview

gdev is a TUI application written in Go that manages development workflows. It wraps git, GitHub CLI (gh), and Claude Code into a single interface.

## Build & Run

```bash
make build    # Build static binary
make run      # Build and run
make clean    # Remove binary
```

## Architecture

- **TUI Framework**: Bubble Tea (github.com/charmbracelet/bubbletea)
- **Styling**: Lip Gloss (github.com/charmbracelet/lipgloss) - add as needed
- **Entry Point**: main.go

## Code Style

- Follow standard Go conventions (gofmt, go vet)
- Use Bubble Tea's Model-View-Update pattern for all UI components
- Keep each TUI component/view in its own file under `internal/ui/`
- Business logic goes in `internal/` packages, separate from UI

## Project Structure

```
.
├── main.go                 # Entry point
├── internal/
│   ├── config/             # Configuration & keybindings
│   │   ├── config.go       # Config manager
│   │   └── keybindings.go  # Keybinding definitions & loading
│   ├── ui/                 # TUI components
│   │   ├── app/
│   │   │   └── app.go      # Main application model
│   │   ├── styles/
│   │   │   └── styles.go   # Shared UI styles (Dracula theme)
│   │   └── todo/           # TODO management views
│   │       ├── model.go    # TODO model & state
│   │       ├── list.go     # List view
│   │       ├── form.go     # Create/edit form
│   │       ├── detail.go   # Detail view
│   │       └── editor.go   # Multi-line prompt editor
│   ├── git/                # Git operations
│   ├── store/              # File-based persistence (~/.gdev/)
│   └── todo/               # TODO domain model
└── Makefile
```

## Key Features to Implement

1. **Branch Management**
   - List branches
   - Create new branch
   - Switch branches
   - Delete branches

2. **PR Management**
   - List open PRs
   - Create PR from current branch
   - View PR status/checks

3. **Claude Code Sessions**
   - Start new session on a branch
   - List active sessions
   - Switch to/attach to session
   - View session output/status

## External Dependencies

Commands that gdev wraps:
- `git` - branch operations, status
- `gh` - GitHub CLI for PR operations
- `claude` - Claude Code CLI for AI sessions

## Configuration

All configuration is stored in `~/.gdev/`. The config is loaded on startup and created with defaults if missing.

### Config Package (`internal/config/`)

- **config.go**: Main config manager that loads/saves all settings
- **keybindings.go**: Keybinding definitions, defaults, and key matching logic

### Loading Config

```go
cfg, err := config.Load(store)  // Loads or creates defaults
kb := cfg.Keys()                 // Access keybindings
```

### Key Matching

Use `config.Matches()` and `config.MatchesAny()` to check keybindings:

```go
kb := m.Config.Keys()

// Single key check
if config.Matches(key, kb.Form.Submit) { ... }

// Multiple key check
if config.MatchesAny(key, kb.Global.Quit, kb.Global.QuitAlt) { ... }
```

## Keybindings

Keybindings are stored in `~/.gdev/keybindings.json`. Created with defaults on first run.

### Key Format

Keys use Bubble Tea's key string format:
- Letters: `a`, `b`, `A`, `B`
- Modifiers: `ctrl+s`, `ctrl+a`, `shift+tab`
- Special: `enter`, `esc`, `tab`, `up`, `down`, `backspace`
- Shift+letter: Use `shift+a` (converted to `A` internally) or just `A`

### Keybinding Groups

| Group | Purpose | Keys |
|-------|---------|------|
| `global` | Work across views | quit, quit_alt, help, move_up, move_down, move_up_alt, move_down_alt |
| `list` | List navigation | select, new, delete, edit, top, bottom, page_up, page_down |
| `form` | Form editing | submit, cancel, next_field, prev_field, add_prompt, delete_prompt, edit_prompt, improve_prompt |
| `editor` | Multi-line editor | save, cancel, line_start, line_end, delete_line, new_line |
| `detail` | Detail view | back, edit, delete, scroll_up, scroll_down |

### Default Keybindings

```json
{
  "global": {
    "quit": "esc",
    "quit_alt": "q",
    "help": "?",
    "move_up": "k",
    "move_down": "j",
    "move_up_alt": "up",
    "move_down_alt": "down"
  },
  "list": {
    "select": "enter",
    "new": "n",
    "delete": "d",
    "edit": "e",
    "top": "g",
    "bottom": "G",
    "page_up": "ctrl+u",
    "page_down": "ctrl+d"
  },
  "form": {
    "submit": "ctrl+s",
    "cancel": "esc",
    "next_field": "tab",
    "prev_field": "shift+tab",
    "add_prompt": "ctrl+a",
    "delete_prompt": "ctrl+d",
    "edit_prompt": "ctrl+e",
    "improve_prompt": "ctrl+i"
  },
  "editor": {
    "save": "ctrl+s",
    "cancel": "esc",
    "line_start": "ctrl+a",
    "line_end": "ctrl+e",
    "delete_line": "ctrl+k",
    "new_line": "enter"
  },
  "detail": {
    "back": "esc",
    "edit": "e",
    "delete": "d",
    "scroll_up": "k",
    "scroll_down": "j"
  }
}
```

### Adding New Keybindings

1. Add field to appropriate struct in `keybindings.go` (e.g., `FormKeys`)
2. Add default value in `DefaultKeybindings()`
3. Add merge logic in `mergeWithDefaults()` for upgrades
4. Use `config.Matches(key, kb.Group.Action)` in the view handler
5. Update help text to show the keybinding dynamically: `fmt.Sprintf("%s save", kb.Form.Submit)`

### Form Edit Mode

Forms use a two-mode system (vim-like):
- **Navigation mode**: Arrow keys/j/k navigate fields, shortcuts work
- **Edit mode**: Enter on a field to type, esc/enter to exit

This prevents shortcuts from being captured while typing.

## Testing

Run tests with:
```bash
go test ./...
```
