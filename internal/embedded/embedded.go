// Package embedded provides embedded claude commands and prompts.
package embedded

import (
	"embed"
	"io/fs"
	"path/filepath"
	"strings"
)

//go:embed claude/commands/*.md
var claudeFS embed.FS

// GetCommand returns the content of an embedded claude command by name.
// Name should be without extension, e.g., "generate-commit-msg".
func GetCommand(name string) (string, error) {
	path := filepath.Join("claude", "commands", name+".md")
	data, err := claudeFS.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetCommandPrompt returns just the prompt portion of a command (without frontmatter).
func GetCommandPrompt(name string) (string, error) {
	content, err := GetCommand(name)
	if err != nil {
		return "", err
	}
	return stripFrontmatter(content), nil
}

// ListCommands returns a list of all embedded command names.
func ListCommands() ([]string, error) {
	var commands []string
	err := fs.WalkDir(claudeFS, "claude/commands", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".md") {
			name := strings.TrimSuffix(filepath.Base(path), ".md")
			commands = append(commands, name)
		}
		return nil
	})
	return commands, err
}

// stripFrontmatter removes YAML frontmatter from markdown content.
func stripFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}

	// Find the closing ---
	rest := content[3:]
	idx := strings.Index(rest, "---")
	if idx == -1 {
		return content
	}

	return strings.TrimSpace(rest[idx+3:])
}
