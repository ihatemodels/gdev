package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const DirName = ".gdev"

var ErrNotFound = errors.New("not found")

type Store struct {
	path string
}

// New creates a new Store instance in ~/.gdev,
// ensuring the directory exists.
func New() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	s := &Store{
		path: filepath.Join(home, DirName),
	}

	if err := s.init(); err != nil {
		return nil, err
	}

	return s, nil
}

// init creates the ~/.gdev directory if it doesn't exist.
func (s *Store) init() error {
	return os.MkdirAll(s.path, 0755)
}

// Path returns the full path to the ~/.gdev directory.
func (s *Store) Path() string {
	return s.path
}

// Write writes raw bytes to a file in the ~/.gdev directory.
func (s *Store) Write(name string, data []byte) error {
	filePath := filepath.Join(s.path, name)
	return os.WriteFile(filePath, data, 0644)
}

// Read reads raw bytes from a file in the ~/.gdev directory.
func (s *Store) Read(name string) ([]byte, error) {
	filePath := filepath.Join(s.path, name)
	data, err := os.ReadFile(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return data, err
}

// WriteJSON marshals v to JSON and writes it to the ~/.gdev directory.
func (s *Store) WriteJSON(name string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return s.Write(name, data)
}

// ReadJSON reads a JSON file from the ~/.gdev directory and unmarshals it into v.
func (s *Store) ReadJSON(name string, v any) error {
	data, err := s.Read(name)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Delete removes a file from the ~/.gdev directory.
func (s *Store) Delete(name string) error {
	filePath := filepath.Join(s.path, name)
	err := os.Remove(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}
	return err
}

// Exists checks if a file exists in the ~/.gdev directory.
func (s *Store) Exists(name string) bool {
	filePath := filepath.Join(s.path, name)
	_, err := os.Stat(filePath)
	return err == nil
}

// List returns all files in the ~/.gdev directory.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.path)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

// SubDir returns a new Store scoped to a subdirectory within ~/.gdev.
func (s *Store) SubDir(name string) (*Store, error) {
	sub := &Store{
		path: filepath.Join(s.path, name),
	}
	if err := sub.init(); err != nil {
		return nil, err
	}
	return sub, nil
}
