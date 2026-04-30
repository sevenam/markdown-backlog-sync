// Package keyring chooses the OS-native credential backend for the host.
// On systems where a keyring is available, it returns a wrapper around
// zalando/go-keyring; otherwise callers should fall back to the file-based
// backend.
package keyring

import (
	"errors"

	"github.com/sevenam/markdown-backlog-sync/internal/auth"
	gokeyring "github.com/zalando/go-keyring"
)

// Backend implements auth.Backend against the OS keyring.
type Backend struct {
	service string
}

// New returns a keyring-backed credential store under the given service
// name namespace.
func New(service string) *Backend { return &Backend{service: service} }

func (b *Backend) Name() string { return "keyring:" + b.service }

func (b *Backend) Get(account string) (string, error) {
	v, err := gokeyring.Get(b.service, account)
	if err != nil {
		if errors.Is(err, gokeyring.ErrNotFound) {
			return "", auth.ErrNotFound
		}
		return "", err
	}
	return v, nil
}

func (b *Backend) Set(account, secret string) error {
	return gokeyring.Set(b.service, account, secret)
}

func (b *Backend) Delete(account string) error {
	err := gokeyring.Delete(b.service, account)
	if err != nil {
		if errors.Is(err, gokeyring.ErrNotFound) {
			return auth.ErrNotFound
		}
		return err
	}
	return nil
}

// Available returns true if the host keyring appears to be usable. It does
// this by performing a no-op Get for a known sentinel account name and
// inspecting the error.
func Available(service string) bool {
	_, err := gokeyring.Get(service, "__mbs_probe__")
	if err == nil {
		return true
	}
	return errors.Is(err, gokeyring.ErrNotFound)
}
