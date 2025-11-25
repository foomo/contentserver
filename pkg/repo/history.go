package repo

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	HistoryRepoJSONPrefix = "contentserver-repo-"
	HistoryRepoJSONSuffix = ".json"
	CurrentKey            = HistoryRepoJSONPrefix + "current" + HistoryRepoJSONSuffix
)

type (
	History struct {
		l            *zap.Logger
		storage      Storage
		historyDir   string // directory used for default filesystem storage
		historyLimit int
		mu           sync.RWMutex
	}
	HistoryOption func(*History)
)

// ------------------------------------------------------------------------------------------------
// ~ Options
// ------------------------------------------------------------------------------------------------

func HistoryWithHistoryLimit(v int) HistoryOption {
	return func(o *History) {
		o.historyLimit = v
	}
}

func HistoryWithHistoryDir(v string) HistoryOption {
	return func(o *History) {
		o.historyDir = v
	}
}

func HistoryWithStorage(s Storage) HistoryOption {
	return func(o *History) {
		o.storage = s
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Constructor
// ------------------------------------------------------------------------------------------------

func NewHistory(l *zap.Logger, opts ...HistoryOption) (*History, error) {
	inst := &History{
		l:            l,
		historyDir:   "/var/lib/contentserver",
		historyLimit: 2,
	}

	for _, opt := range opts {
		opt(inst)
	}

	// If no storage provided, create a default filesystem storage
	if inst.storage == nil {
		storage, err := NewFilesystemStorage(inst.historyDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create default filesystem storage: %w", err)
		}
		inst.storage = storage
	}

	return inst, nil
}

// ------------------------------------------------------------------------------------------------
// ~ Public methods
// ------------------------------------------------------------------------------------------------

// Add writes the JSON bytes to storage as both a backup and current file.
func (h *History) Add(ctx context.Context, jsonBytes []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	backupKey := HistoryRepoJSONPrefix + time.Now().Format(time.RFC3339Nano) + HistoryRepoJSONSuffix

	if err := h.storage.Write(ctx, backupKey, jsonBytes); err != nil {
		return errors.Wrap(err, "failed to write backup history file")
	}

	h.l.Debug("writing files",
		zap.String("backup", backupKey),
		zap.String("current", CurrentKey),
	)

	if err := h.storage.Write(ctx, CurrentKey, jsonBytes); err != nil {
		return errors.Wrap(err, "failed to write current history")
	}

	if err := h.cleanup(ctx); err != nil {
		return errors.Wrap(err, "failed to clean up history")
	}

	return nil
}

// GetCurrent reads the current snapshot into the provided buffer.
func (h *History) GetCurrent(ctx context.Context, buf *bytes.Buffer) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	data, err := h.storage.Read(ctx, CurrentKey)
	if err != nil {
		return err
	}
	_, err = buf.Write(data)
	return err
}

// Close releases resources held by the history storage.
func (h *History) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.storage != nil {
		return h.storage.Close()
	}
	return nil
}

// ------------------------------------------------------------------------------------------------
// ~ Private methods
// ------------------------------------------------------------------------------------------------

func (h *History) getHistory(ctx context.Context) (files []string, err error) {
	keys, err := h.storage.List(ctx, HistoryRepoJSONPrefix)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if key != CurrentKey &&
			strings.HasPrefix(key, HistoryRepoJSONPrefix) &&
			strings.HasSuffix(key, HistoryRepoJSONSuffix) {
			files = append(files, key)
		}
	}
	return files, nil
}

func (h *History) cleanup(ctx context.Context) error {
	files, err := h.getFilesForCleanup(ctx, h.historyLimit)
	if err != nil {
		return err
	}

	for _, f := range files {
		h.l.Debug("removing outdated backup", zap.String("file", f))
		if err := h.storage.Delete(ctx, f); err != nil {
			return fmt.Errorf("could not remove file %s: %w", f, err)
		}
	}

	return nil
}

func (h *History) getFilesForCleanup(ctx context.Context, historyVersions int) (files []string, err error) {
	contentFiles, err := h.getHistory(ctx)
	if err != nil {
		return nil, errors.New("could not generate file cleanup list: " + err.Error())
	}

	if len(contentFiles) > historyVersions {
		for i := historyVersions; i < len(contentFiles); i++ {
			files = append(files, contentFiles[i])
		}
	}
	return files, nil
}
