// Package config loads and validates the workspace configuration file
// (backlog.config.toml). Configuration is layered: defaults are
// overridden by the file, then by env vars, then by CLI flags.
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config is the in-memory representation of backlog.config.toml.
type Config struct {
	Workspace WorkspaceSection `toml:"workspace"`
	Providers []ProviderConfig `toml:"provider"`
}

// WorkspaceSection holds workspace-level options.
type WorkspaceSection struct {
	ItemsDir string `toml:"items_dir"`
}

// ProviderConfig is a single named provider entry in the config file.
// Provider-specific options live in Options.
type ProviderConfig struct {
	Name    string         `toml:"name"`
	Type    string         `toml:"type"`
	Options map[string]any `toml:"-"`
	raw     map[string]any
}

// UnmarshalTOML lets us preserve provider-specific keys without enumerating
// every backend type here.
func (p *ProviderConfig) UnmarshalTOML(data any) error {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("provider entry must be a TOML table, got %T", data)
	}
	if v, ok := m["name"].(string); ok {
		p.Name = v
	}
	if v, ok := m["type"].(string); ok {
		p.Type = v
	}
	opts := make(map[string]any, len(m))
	for k, v := range m {
		if k == "name" || k == "type" {
			continue
		}
		opts[k] = v
	}
	p.Options = opts
	p.raw = m
	return nil
}

// Load parses the config file at path. It does not apply defaults beyond
// what TOML provides; callers should use ApplyDefaults.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if _, err := toml.Decode(string(b), &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}

// ApplyDefaults fills in default values on cfg in place.
func (c *Config) ApplyDefaults() {
	if c.Workspace.ItemsDir == "" {
		c.Workspace.ItemsDir = "backlog"
	}
}

// Validate returns an error if the config is internally inconsistent.
// Currently this checks for duplicate provider names and required fields.
func (c *Config) Validate() error {
	seen := make(map[string]struct{}, len(c.Providers))
	for i, p := range c.Providers {
		if p.Name == "" {
			return fmt.Errorf("provider[%d]: name is required", i)
		}
		if p.Type == "" {
			return fmt.Errorf("provider %q: type is required", p.Name)
		}
		if _, dup := seen[p.Name]; dup {
			return fmt.Errorf("duplicate provider name %q", p.Name)
		}
		seen[p.Name] = struct{}{}
	}
	return nil
}

// Provider returns the named provider config, or false if missing.
func (c *Config) Provider(name string) (ProviderConfig, bool) {
	for _, p := range c.Providers {
		if p.Name == name {
			return p, true
		}
	}
	return ProviderConfig{}, false
}

// ErrNoProviders signals an empty provider list (informational, not fatal).
var ErrNoProviders = errors.New("no providers configured")
