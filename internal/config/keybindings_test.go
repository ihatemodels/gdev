package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ihatemodels/gdev/internal/store"
)

func TestLoadKeybindings_CreatesDefaults(t *testing.T) {
	// Create a temporary directory for the test store
	tmpDir, err := os.MkdirTemp("", "gdev-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Override home directory for the test
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	s, err := store.New()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Verify keybindings file doesn't exist
	kbPath := filepath.Join(tmpDir, ".gdev", "keybindings.json")
	if _, err := os.Stat(kbPath); !os.IsNotExist(err) {
		t.Fatal("keybindings.json should not exist yet")
	}

	// Load keybindings - should create defaults
	kb, err := LoadKeybindings(s)
	if err != nil {
		t.Fatalf("Failed to load keybindings: %v", err)
	}

	// Verify keybindings file was created
	if _, err := os.Stat(kbPath); err != nil {
		t.Fatalf("keybindings.json should have been created: %v", err)
	}

	// Verify default values
	if kb.Global.Quit != "esc" {
		t.Errorf("Expected Global.Quit to be 'esc', got '%s'", kb.Global.Quit)
	}
	if kb.List.Select != "enter" {
		t.Errorf("Expected List.Select to be 'enter', got '%s'", kb.List.Select)
	}
	if kb.Form.Submit != "ctrl+s" {
		t.Errorf("Expected Form.Submit to be 'ctrl+s', got '%s'", kb.Form.Submit)
	}
}

func TestLoadKeybindings_LoadsExisting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gdev-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	s, err := store.New()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create custom keybindings
	customKb := DefaultKeybindings()
	customKb.Global.Quit = "ctrl+q"
	customKb.List.New = "a"

	if err := SaveKeybindings(s, customKb); err != nil {
		t.Fatalf("Failed to save keybindings: %v", err)
	}

	// Load and verify
	kb, err := LoadKeybindings(s)
	if err != nil {
		t.Fatalf("Failed to load keybindings: %v", err)
	}

	if kb.Global.Quit != "ctrl+q" {
		t.Errorf("Expected Global.Quit to be 'ctrl+q', got '%s'", kb.Global.Quit)
	}
	if kb.List.New != "a" {
		t.Errorf("Expected List.New to be 'a', got '%s'", kb.List.New)
	}
}

func TestMatchesAny(t *testing.T) {
	tests := []struct {
		key      string
		bindings []string
		expected bool
	}{
		{"esc", []string{"esc", "q"}, true},
		{"q", []string{"esc", "q"}, true},
		{"x", []string{"esc", "q"}, false},
		{"enter", []string{"enter"}, true},
	}

	for _, tt := range tests {
		result := MatchesAny(tt.key, tt.bindings...)
		if result != tt.expected {
			t.Errorf("MatchesAny(%q, %v) = %v, want %v", tt.key, tt.bindings, result, tt.expected)
		}
	}
}

func TestMatches(t *testing.T) {
	if !Matches("ctrl+s", "ctrl+s") {
		t.Error("Matches should return true for identical strings")
	}
	if Matches("ctrl+s", "ctrl+x") {
		t.Error("Matches should return false for different strings")
	}
}

func TestMatches_ShiftLetters(t *testing.T) {
	// In Bubble Tea, Shift+A produces "A", not "shift+a"
	// Our config allows "shift+a" which should match "A"
	tests := []struct {
		key      string
		binding  string
		expected bool
	}{
		{"A", "shift+a", true},
		{"D", "shift+d", true},
		{"E", "shift+e", true},
		{"I", "shift+i", true},
		{"G", "G", true},           // Direct uppercase also works
		{"G", "shift+g", true},     // shift+g should match G
		{"a", "shift+a", false},    // lowercase doesn't match shift+a
		{"shift+tab", "shift+tab", true}, // special keys still work
	}

	for _, tt := range tests {
		result := Matches(tt.key, tt.binding)
		if result != tt.expected {
			t.Errorf("Matches(%q, %q) = %v, want %v", tt.key, tt.binding, result, tt.expected)
		}
	}
}
