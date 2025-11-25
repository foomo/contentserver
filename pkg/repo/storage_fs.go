package repo

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// FilesystemStorage implements Storage using the local filesystem.
type FilesystemStorage struct {
	baseDir string
	mu      sync.RWMutex
}

// NewFilesystemStorage creates a new filesystem-backed storage.
func NewFilesystemStorage(baseDir string) (*FilesystemStorage, error) {
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, err
	}
	return &FilesystemStorage{baseDir: baseDir}, nil
}

func (f *FilesystemStorage) Write(_ context.Context, key string, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	path := filepath.Join(f.baseDir, key)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (f *FilesystemStorage) Read(_ context.Context, key string) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	path := filepath.Join(f.baseDir, key)
	return os.ReadFile(path)
}

// List returns keys matching the prefix.
// Note: Only lists files in the base directory (non-recursive).
// Keys must not contain path separators for correct behavior.
func (f *FilesystemStorage) List(_ context.Context, prefix string) ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	entries, err := os.ReadDir(f.baseDir)
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			keys = append(keys, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	return keys, nil
}

func (f *FilesystemStorage) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	path := filepath.Join(f.baseDir, key)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (f *FilesystemStorage) Close() error {
	return nil
}
