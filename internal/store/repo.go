package store

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// RepoState holds the persisted state for a git repository.
type RepoState struct {
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	LastOpenedAt time.Time `json:"last_opened_at"`
}

// repoID generates a unique ID for a repo based on its path.
func repoID(path string) string {
	hash := sha256.Sum256([]byte(path))
	return hex.EncodeToString(hash[:8])
}

// GetRepoState loads the state for a repository by its path.
func (s *Store) GetRepoState(repoPath string) (*RepoState, error) {
	repos, err := s.SubDir("repos")
	if err != nil {
		return nil, err
	}

	id := repoID(repoPath)
	var state RepoState
	if err := repos.ReadJSON(id+".json", &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// SaveRepoState saves the state for a repository.
func (s *Store) SaveRepoState(state *RepoState) error {
	repos, err := s.SubDir("repos")
	if err != nil {
		return err
	}

	id := repoID(state.Path)
	return repos.WriteJSON(id+".json", state)
}

// TouchRepo updates the LastOpenedAt for a repository, creating state if needed.
func (s *Store) TouchRepo(repoPath, repoName string) (*RepoState, error) {
	state, err := s.GetRepoState(repoPath)
	if err == ErrNotFound {
		state = &RepoState{
			Path: repoPath,
			Name: repoName,
		}
	} else if err != nil {
		return nil, err
	}

	state.LastOpenedAt = time.Now()
	if err := s.SaveRepoState(state); err != nil {
		return nil, err
	}
	return state, nil
}
