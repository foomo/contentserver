package repo

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	HistoryRepoJSONPrefix = "contentserver-repo-"
	HistoryRepoJSONSuffix = ".json"
)

type (
	History struct {
		l             *zap.Logger
		historyDir    string
		historyLimit  int
		currentMutext sync.RWMutex
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

// ------------------------------------------------------------------------------------------------
// ~ Constructor
// ------------------------------------------------------------------------------------------------

func NewHistory(l *zap.Logger, opts ...HistoryOption) *History {
	inst := &History{
		l:            l,
		historyDir:   "/var/lib/contentserver",
		historyLimit: 2,
	}

	for _, opt := range opts {
		opt(inst)
	}

	return inst
}

// ------------------------------------------------------------------------------------------------
// ~ Public methods
// ------------------------------------------------------------------------------------------------

func (h *History) Add(jsonBytes []byte) error {
	backupFilename := path.Join(h.historyDir, HistoryRepoJSONPrefix+time.Now().Format(time.RFC3339Nano)+HistoryRepoJSONSuffix)
	currentFilename := h.GetCurrentFilename()

	if err := os.MkdirAll(path.Dir(backupFilename), 0700); err != nil {
		return errors.Wrap(err, "failed to create history dir")
	}

	if err := os.WriteFile(backupFilename, jsonBytes, 0600); err != nil {
		return errors.Wrap(err, "failed to write backup history file")
	}

	h.l.Debug("writing files",
		zap.String("backup", backupFilename),
		zap.String("current", currentFilename),
	)

	// current filename
	h.currentMutext.Lock()
	defer h.currentMutext.Unlock()
	if err := os.WriteFile(currentFilename, jsonBytes, 0600); err != nil {
		return errors.Wrap(err, "failed to write current history")
	}

	if err := h.cleanup(); err != nil {
		return errors.Wrap(err, "failed to clean up history")
	}

	return nil
}

func (h *History) GetCurrentFilename() string {
	return path.Join(h.historyDir, HistoryRepoJSONPrefix+"current"+HistoryRepoJSONSuffix)
}

func (h *History) GetCurrent(buf *bytes.Buffer) error {
	h.currentMutext.RLock()
	defer h.currentMutext.RUnlock()
	f, err := os.Open(h.GetCurrentFilename())
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(buf, f)
	return err
}

// ------------------------------------------------------------------------------------------------
// ~ Private methods
// ------------------------------------------------------------------------------------------------

func (h *History) getHistory() (files []string, err error) {
	fileInfos, err := os.ReadDir(h.historyDir)
	if err != nil {
		return
	}
	currentName := h.GetCurrentFilename()
	for _, f := range fileInfos {
		if !f.IsDir() {
			filename := f.Name()
			if filename != currentName && (strings.HasPrefix(filename, HistoryRepoJSONPrefix) && strings.HasSuffix(filename, HistoryRepoJSONSuffix)) {
				files = append(files, path.Join(h.historyDir, filename))
			}
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return
}

func (h *History) cleanup() error {
	files, err := h.getFilesForCleanup(h.historyLimit)
	if err != nil {
		return err
	}

	for _, f := range files {
		h.l.Debug("removing outdated backup", zap.String("file", f))
		err := os.Remove(f)
		if err != nil {
			return fmt.Errorf("could not remove file %s : %s", f, err.Error())
		}
	}

	return nil
}

func (h *History) getFilesForCleanup(historyVersions int) (files []string, err error) {
	contentFiles, err := h.getHistory()
	if err != nil {
		return nil, errors.New("could not generate file cleanup list: " + err.Error())
	}

	// fmt.Println("contentFiles:")
	// for _, f := range contentFiles {
	// 	fmt.Println(f)
	// }

	// -1 to remove the current backup file from the number of items
	// so that only files with a timestamp are compared
	if len(contentFiles)-1 > historyVersions {
		for i := historyVersions + 1; i < len(contentFiles); i++ {
			// ignore current repository file to fall back on
			if contentFiles[i] == h.GetCurrentFilename() {
				continue
			}
			files = append(files, contentFiles[i])
		}
	}
	return files, nil
}
