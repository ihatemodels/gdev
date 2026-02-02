package todo

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Todo represents a single TODO item with associated Claude Code prompts.
type Todo struct {
	ID          string    `json:"id"`
	Branch      string    `json:"branch"`
	Name        string    `json:"name"`
	Description string    `json:"description"` // supports markdown
	Prompts     []string  `json:"prompts"`     // markdown prompts for Claude Code
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TodoList holds all TODOs for a repository.
type TodoList struct {
	RepoPath string `json:"repo_path"`
	Todos    []Todo `json:"todos"`
}

// NewTodo creates a new Todo with a generated ID and timestamps.
func NewTodo(branch, name, description string, prompts []string) *Todo {
	now := time.Now()
	return &Todo{
		ID:          generateID(),
		Branch:      branch,
		Name:        name,
		Description: description,
		Prompts:     prompts,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// generateID creates a random 8-byte hex ID.
func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Update sets the UpdatedAt timestamp to now.
func (t *Todo) Update() {
	t.UpdatedAt = time.Now()
}

// AddPrompt adds a new prompt to the Todo.
func (t *Todo) AddPrompt(prompt string) {
	t.Prompts = append(t.Prompts, prompt)
	t.Update()
}

// RemovePrompt removes a prompt at the given index.
func (t *Todo) RemovePrompt(index int) {
	if index < 0 || index >= len(t.Prompts) {
		return
	}
	t.Prompts = append(t.Prompts[:index], t.Prompts[index+1:]...)
	t.Update()
}

// Validate checks that required fields are set.
func (t *Todo) Validate() bool {
	return t.Name != "" && t.Branch != ""
}
