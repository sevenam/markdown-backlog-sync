// Package state implements the .sync/ sidecar store that records, for each
// item, the data needed for safe bidirectional sync (remote ids, etags,
// last-synced content hashes, mappings).
package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
)

// SchemaVersion is bumped when the on-disk JSON shape changes in a
// breaking way. Migration hooks live in this package.
const SchemaVersion = 1

// localIDPattern restricts on-disk identifiers to a safe character set.
// It deliberately rejects path separators, drive letters, and ".." to
// keep <Store>.itemPath / SnapshotPath inside .sync/items and
// .sync/snapshots respectively.
var localIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// ValidateLocalID returns an error if id is unsafe for use as a filename.
func ValidateLocalID(id string) error {
	if !localIDPattern.MatchString(id) {
		return fmt.Errorf("invalid local id %q: must match %s", id, localIDPattern)
	}
	if id == "." || id == ".." {
		return fmt.Errorf("invalid local id %q", id)
	}
	return nil
}

// ItemState captures the last-synced snapshot for one item.
type ItemState struct {
	Schema       int    `json:"schema"`
	LocalID      string `json:"local_id"`
	LocalPath    string `json:"local_path"`
	Provider     string `json:"provider"`
	RemoteID     string `json:"remote_id"`
	RemoteURL    string `json:"remote_url,omitempty"`
	RemoteRev    string `json:"remote_rev,omitempty"` // etag / Azure DevOps rev / GitHub updated_at
	ContentHash  string `json:"content_hash"`         // sha256 of last-synced canonical Markdown
	LastSyncedAt string `json:"last_synced_at"`       // RFC3339
}

// Index maps "<provider>\x00<remoteID>" → localID, kept in memory.
// Persisted as a sorted JSON object for determinism.
type Index struct {
	Schema  int               `json:"schema"`
	Mapping map[string]string `json:"mapping"` // key = provider + "\x00" + remoteID
}

// Store provides concurrency-safe access to the .sync/ directory.
type Store struct {
	root string // path to .sync/

	mu sync.Mutex
}

// Open returns a Store rooted at dir, creating directories as needed.
func Open(dir string) (*Store, error) {
	if dir == "" {
		return nil, errors.New("state dir is required")
	}
	for _, sub := range []string{"items", "snapshots", "providers"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o700); err != nil {
			return nil, err
		}
	}
	return &Store{root: dir}, nil
}

// Hash returns the canonical content hash used in ItemState.ContentHash.
func Hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// indexPath returns the path of index.json.
func (s *Store) indexPath() string { return filepath.Join(s.root, "index.json") }

// itemPath returns the path of an item state file. It validates id first.
func (s *Store) itemPath(localID string) (string, error) {
	if err := ValidateLocalID(localID); err != nil {
		return "", err
	}
	return filepath.Join(s.root, "items", localID+".json"), nil
}

// snapshotPath returns the path of an item's last-synced canonical
// Markdown snapshot. The 3-way merge in the sync engine reads this file
// as the "base" version.
func (s *Store) snapshotPath(localID string) (string, error) {
	if err := ValidateLocalID(localID); err != nil {
		return "", err
	}
	return filepath.Join(s.root, "snapshots", localID+".md"), nil
}

// SaveSnapshot writes the last-synced canonical Markdown for an item.
// Callers should pass the exact bytes that were last reconciled with the
// remote so the sync engine can perform a true 3-way merge.
func (s *Store) SaveSnapshot(localID string, canonicalMarkdown string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.snapshotPath(localID)
	if err != nil {
		return err
	}
	return atomicWrite(p, []byte(canonicalMarkdown), 0o600)
}

// LoadSnapshot returns the last-synced canonical Markdown for an item, or
// (empty, nil) if no snapshot exists yet (e.g. the item has never been
// synced).
func (s *Store) LoadSnapshot(localID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.snapshotPath(localID)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(b), nil
}

// LoadIndex reads index.json. A missing file returns an empty Index.
func (s *Store) LoadIndex() (*Index, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return loadIndexLocked(s.indexPath())
}

func loadIndexLocked(path string) (*Index, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Index{Schema: SchemaVersion, Mapping: map[string]string{}}, nil
		}
		return nil, err
	}
	var idx Index
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if idx.Schema > SchemaVersion {
		return nil, fmt.Errorf("%s: index schema v%d is newer than supported v%d (please upgrade mbs)", path, idx.Schema, SchemaVersion)
	}
	if idx.Mapping == nil {
		idx.Mapping = map[string]string{}
	}
	return &idx, nil
}

// SaveIndex writes index.json atomically.
func (s *Store) SaveIndex(idx *Index) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx.Schema = SchemaVersion
	if idx.Mapping == nil {
		idx.Mapping = map[string]string{}
	}
	// Serialize with sorted keys for determinism.
	keys := make([]string, 0, len(idx.Mapping))
	for k := range idx.Mapping {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := struct {
		Schema  int               `json:"schema"`
		Mapping map[string]string `json:"mapping"`
	}{Schema: SchemaVersion, Mapping: idx.Mapping}
	// json.Marshal already sorts map keys lexicographically, but we keep
	// the explicit sort above to make the contract obvious.
	_ = keys
	b, err := json.MarshalIndent(ordered, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(s.indexPath(), b, 0o600)
}

// LoadItem reads a single item state, returning (nil, nil) if missing.
func (s *Store) LoadItem(localID string) (*ItemState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.itemPath(localID)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var it ItemState
	if err := json.Unmarshal(b, &it); err != nil {
		return nil, fmt.Errorf("parse item state %s: %w", localID, err)
	}
	if it.Schema > SchemaVersion {
		return nil, fmt.Errorf("item %s: state schema v%d is newer than supported v%d (please upgrade mbs)", localID, it.Schema, SchemaVersion)
	}
	return &it, nil
}

// SaveItem writes an item state atomically.
func (s *Store) SaveItem(it *ItemState) error {
	if it == nil || it.LocalID == "" {
		return errors.New("item state requires local_id")
	}
	if err := ValidateLocalID(it.LocalID); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	it.Schema = SchemaVersion
	b, err := json.MarshalIndent(it, "", "  ")
	if err != nil {
		return err
	}
	p, err := s.itemPath(it.LocalID)
	if err != nil {
		return err
	}
	return atomicWrite(p, b, 0o600)
}

// DeleteItem removes an item state file and its snapshot. Missing files
// are not an error.
func (s *Store) DeleteItem(localID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, fn := range []func(string) (string, error){s.itemPath, s.snapshotPath} {
		p, err := fn(localID)
		if err != nil {
			return err
		}
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

// MapKey builds the index key for a (provider, remoteID) pair.
func MapKey(provider, remoteID string) string {
	return provider + "\x00" + remoteID
}

// atomicWrite writes data to path via temp file + rename to survive crashes
// mid-write.
func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Chmod(mode); err != nil && !isWindows() {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return err
	}
	return nil
}
