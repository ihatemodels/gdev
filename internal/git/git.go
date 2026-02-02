package git

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrNotRepo = errors.New("not a git repository")

type Repo struct {
	Root   string
	Name   string
	Branch string
}

// GetRepo returns info about the git repository at the current directory.
// Returns ErrNotRepo if not in a git repository.
func GetRepo() (*Repo, error) {
	root, err := findRepoRoot()
	if err != nil {
		return nil, err
	}

	branch, err := getCurrentBranch(root)
	if err != nil {
		branch = "unknown"
	}

	return &Repo{
		Root:   root,
		Name:   filepath.Base(root),
		Branch: branch,
	}, nil
}

// findRepoRoot walks up the directory tree to find the git repository root.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil && (info.IsDir() || info.Mode().IsRegular()) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotRepo
		}
		dir = parent
	}
}

// getCurrentBranch returns the current branch name.
func getCurrentBranch(repoRoot string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// HasRemoteChanges checks if there are unpulled changes from the remote.
func (r *Repo) HasRemoteChanges() (bool, error) {
	// Fetch latest from remote (silently)
	fetch := exec.Command("git", "fetch", "--dry-run")
	fetch.Dir = r.Root
	fetchOut, _ := fetch.CombinedOutput()

	// If fetch --dry-run has output, there are changes
	if len(bytes.TrimSpace(fetchOut)) > 0 {
		return true, nil
	}

	// Check if we're behind the remote
	cmd := exec.Command("git", "rev-list", "--count", "HEAD..@{upstream}")
	cmd.Dir = r.Root
	out, err := cmd.Output()
	if err != nil {
		// No upstream configured
		return false, nil
	}

	count := strings.TrimSpace(string(out))
	return count != "0", nil
}

// HasLocalChanges checks if there are uncommitted local changes.
func (r *Repo) HasLocalChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = r.Root
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(bytes.TrimSpace(out)) > 0, nil
}

// GetAheadBehind returns how many commits ahead/behind we are from upstream.
func (r *Repo) GetAheadBehind() (ahead int, behind int, err error) {
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	cmd.Dir = r.Root
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	parts := strings.Fields(string(out))
	if len(parts) != 2 {
		return 0, 0, nil
	}

	fmt := "%d"
	var a, b int
	if _, err := parseInts(parts[0], parts[1], &a, &b); err != nil {
		return 0, 0, err
	}
	_ = fmt

	return a, b, nil
}

func parseInts(s1, s2 string, i1, i2 *int) (bool, error) {
	var err error
	*i1, err = parseInt(s1)
	if err != nil {
		return false, err
	}
	*i2, err = parseInt(s2)
	return err == nil, err
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("invalid number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
