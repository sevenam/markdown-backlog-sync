package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/argon2"
)

// MemoryBackend is an in-process backend used in tests and as a fallback
// when no persistent store is available.
type MemoryBackend struct {
	mu   sync.Mutex
	data map[string]string
}

// NewMemoryBackend returns an empty in-memory backend.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{data: map[string]string{}}
}

func (m *MemoryBackend) Name() string { return "memory" }

func (m *MemoryBackend) Get(account string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[account]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (m *MemoryBackend) Set(account, secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[account] = secret
	return nil
}

func (m *MemoryBackend) Delete(account string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[account]; !ok {
		return ErrNotFound
	}
	delete(m.data, account)
	return nil
}

// FileBackend persists credentials to a JSON file. Each entry is encrypted
// with AES-GCM; the encryption key is derived from a passphrase using
// Argon2id with a per-file random salt. The file is a small versioned
// envelope:
//
//	{
//	  "version": 1,
//	  "kdf": "argon2id",
//	  "salt": "<base64>",
//	  "nonce": "<base64>",
//	  "ciphertext": "<base64>"
//	}
//
// Threat model: protects against casual disk inspection and reasonable
// offline brute force given a strong passphrase. Does NOT protect against
// attackers who can read the passphrase or the running process memory.
type FileBackend struct {
	path       string
	passphrase string

	mu sync.Mutex
}

// NewFileBackend returns a FileBackend storing data at path.
func NewFileBackend(path, passphrase string) *FileBackend {
	return &FileBackend{path: path, passphrase: passphrase}
}

func (f *FileBackend) Name() string { return "file:" + f.path }

// envelope is the on-disk container.
type envelope struct {
	Version    int    `json:"version"`
	KDF        string `json:"kdf"`
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

const (
	envelopeVersion = 1
	saltLen         = 16
	keyLen          = 32
	argonTime       = 2
	argonMemoryKiB  = 64 * 1024 // 64 MiB
	argonThreads    = 1
)

func (f *FileBackend) load() (map[string]string, error) {
	b, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	if len(b) == 0 {
		return map[string]string{}, nil
	}
	var env envelope
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, fmt.Errorf("parse %s: %w", f.path, err)
	}
	if env.Version != envelopeVersion {
		return nil, fmt.Errorf("%s: unsupported credentials version %d", f.path, env.Version)
	}
	if env.KDF != "argon2id" {
		return nil, fmt.Errorf("%s: unsupported KDF %q", f.path, env.KDF)
	}
	key := deriveKey(f.passphrase, env.Salt)
	plain, err := decrypt(env.Ciphertext, env.Nonce, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt %s: %w", f.path, err)
	}
	var out map[string]string
	if err := json.Unmarshal(plain, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (f *FileBackend) save(data map[string]string) error {
	plain, err := json.Marshal(data)
	if err != nil {
		return err
	}
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}
	key := deriveKey(f.passphrase, salt)
	nonce, ct, err := encrypt(plain, key)
	if err != nil {
		return err
	}
	env := envelope{
		Version:    envelopeVersion,
		KDF:        "argon2id",
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: ct,
	}
	body, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(f.path, body, 0o600)
}

func (f *FileBackend) Get(account string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, err := f.load()
	if err != nil {
		return "", err
	}
	v, ok := d[account]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (f *FileBackend) Set(account, secret string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, err := f.load()
	if err != nil {
		return err
	}
	d[account] = secret
	return f.save(d)
}

func (f *FileBackend) Delete(account string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, err := f.load()
	if err != nil {
		return err
	}
	if _, ok := d[account]; !ok {
		return ErrNotFound
	}
	delete(d, account)
	return f.save(d)
}

func deriveKey(passphrase string, salt []byte) []byte {
	return argon2.IDKey([]byte(passphrase), salt, argonTime, argonMemoryKiB, argonThreads, keyLen)
}

func encrypt(plain, key []byte) (nonce, ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	return nonce, gcm.Seal(nil, nonce, plain, nil), nil
}

func decrypt(ciphertext, nonce, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, errors.New("invalid nonce length")
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}
