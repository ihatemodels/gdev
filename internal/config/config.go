package config

import (
	"github.com/ihatemodels/gdev/internal/store"
)

// Config holds all application configuration.
type Config struct {
	store       *store.Store
	Keybindings *Keybindings
}

// Load loads the application configuration from the store.
// Creates default configuration files if they don't exist.
func Load(s *store.Store) (*Config, error) {
	kb, err := LoadKeybindings(s)
	if err != nil {
		return nil, err
	}

	return &Config{
		store:       s,
		Keybindings: kb,
	}, nil
}

// Save persists the current configuration to the store.
func (c *Config) Save() error {
	return SaveKeybindings(c.store, c.Keybindings)
}

// ResetKeybindings resets keybindings to their defaults.
func (c *Config) ResetKeybindings() error {
	c.Keybindings = DefaultKeybindings()
	return c.Save()
}

// Keys returns the keybindings for convenient access.
func (c *Config) Keys() *Keybindings {
	return c.Keybindings
}
