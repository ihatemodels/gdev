package store

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/ihatemodels/gdev/internal/todo"
)

// todoRepoID generates a unique ID for a repo's todos based on its path.
func todoRepoID(path string) string {
	hash := sha256.Sum256([]byte(path))
	return hex.EncodeToString(hash[:8])
}

// GetTodos loads the todo list for a repository by its path.
func (s *Store) GetTodos(repoPath string) (*todo.TodoList, error) {
	todos, err := s.SubDir("todos")
	if err != nil {
		return nil, err
	}

	id := todoRepoID(repoPath)
	var list todo.TodoList
	if err := todos.ReadJSON(id+".json", &list); err != nil {
		if err == ErrNotFound {
			// Return empty list if not found
			return &todo.TodoList{
				RepoPath: repoPath,
				Todos:    []todo.Todo{},
			}, nil
		}
		return nil, err
	}
	return &list, nil
}

// SaveTodos saves the todo list for a repository.
func (s *Store) SaveTodos(list *todo.TodoList) error {
	todos, err := s.SubDir("todos")
	if err != nil {
		return err
	}

	id := todoRepoID(list.RepoPath)
	return todos.WriteJSON(id+".json", list)
}

// AddTodo adds a new todo to a repository's list.
func (s *Store) AddTodo(repoPath string, t *todo.Todo) error {
	list, err := s.GetTodos(repoPath)
	if err != nil {
		return err
	}

	list.Todos = append(list.Todos, *t)
	return s.SaveTodos(list)
}

// UpdateTodo updates an existing todo in a repository's list.
func (s *Store) UpdateTodo(repoPath string, t *todo.Todo) error {
	list, err := s.GetTodos(repoPath)
	if err != nil {
		return err
	}

	for i, existing := range list.Todos {
		if existing.ID == t.ID {
			list.Todos[i] = *t
			return s.SaveTodos(list)
		}
	}

	return ErrNotFound
}

// DeleteTodo removes a todo from a repository's list by ID.
func (s *Store) DeleteTodo(repoPath string, todoID string) error {
	list, err := s.GetTodos(repoPath)
	if err != nil {
		return err
	}

	for i, existing := range list.Todos {
		if existing.ID == todoID {
			list.Todos = append(list.Todos[:i], list.Todos[i+1:]...)
			return s.SaveTodos(list)
		}
	}

	return ErrNotFound
}
