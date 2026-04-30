// Package workspace defines the on-disk layout used by mbs and provides
// discovery helpers.
package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Names of well-known files and directories within a workspace.
const (
	ConfigFileName  = "backlog.config.toml"
	StateDirName    = ".sync"
	DefaultItemsDir = "backlog"
)

// Workspace describes a discovered workspace on disk.
type Workspace struct {
	Root       string // absolute path to the workspace root
	ConfigPath string // absolute path to backlog.config.toml
	ItemsDir   string // absolute path to the items directory
	StateDir   string // absolute path to .sync/
}

// ErrNotFound is returned when no workspace can be discovered.
var ErrNotFound = errors.New("no workspace found (no backlog.config.toml in current or parent directories)")

// Discover walks upward from start until ConfigFileName is found.
// If start is empty, the current working directory is used.
func Discover(start string) (string, error) {
	dir := start
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	cur := abs
	for {
		candidate := filepath.Join(cur, ConfigFileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", ErrNotFound
		}
		cur = parent
	}
}

// Open resolves a Workspace from an explicit root or by discovery.
// itemsDirRel is the relative items directory from config (defaults applied
// if empty).
func Open(root, itemsDirRel string) (*Workspace, error) {
	if root == "" {
		var err error
		root, err = Discover("")
		if err != nil {
			return nil, err
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if itemsDirRel == "" {
		itemsDirRel = DefaultItemsDir
	}
	return &Workspace{
		Root:       abs,
		ConfigPath: filepath.Join(abs, ConfigFileName),
		ItemsDir:   filepath.Join(abs, itemsDirRel),
		StateDir:   filepath.Join(abs, StateDirName),
	}, nil
}

// Create initializes a new workspace at root, creating the items dir, the
// state dir, and a default config file if they do not exist. It returns the
// resolved Workspace and a bool indicating whether a fresh config was
// written. It is an error to call Create when the config already exists
// unless overwrite is true.
func Create(root, itemsDirRel string, overwrite bool) (*Workspace, bool, error) {
	ws, err := Open(root, itemsDirRel)
	if err != nil {
		return nil, false, err
	}
	if err := os.MkdirAll(ws.Root, 0o755); err != nil {
		return nil, false, err
	}
	if err := os.MkdirAll(ws.ItemsDir, 0o755); err != nil {
		return nil, false, err
	}
	if err := os.MkdirAll(ws.StateDir, 0o700); err != nil {
		return nil, false, err
	}
	wroteConfig := false
	if _, err := os.Stat(ws.ConfigPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, false, err
		}
		if err := os.WriteFile(ws.ConfigPath, []byte(defaultConfig(itemsDirRel)), 0o644); err != nil {
			return nil, false, err
		}
		wroteConfig = true
	} else if overwrite {
		if err := os.WriteFile(ws.ConfigPath, []byte(defaultConfig(itemsDirRel)), 0o644); err != nil {
			return nil, false, err
		}
		wroteConfig = true
	}
	return ws, wroteConfig, nil
}

func defaultConfig(itemsDir string) string {
	if itemsDir == "" {
		itemsDir = DefaultItemsDir
	}
	return fmt.Sprintf(`# markdown-backlog-sync workspace configuration

[workspace]
items_dir = %q

# Add one [[provider]] block per remote backlog. Examples:
#
# [[provider]]
# name = "gh-main"
# type = "github-issues"
# repo = "owner/repo"
#
# [[provider]]
# name = "azdo-platform"
# type = "azure-devops-boards"
# organization = "contoso"
# project = "Platform"
# team = "Platform Team"
`, itemsDir)
}
