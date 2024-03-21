package repo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	HistoryRepoJSONPrefix = "contentserver-repo-"
	HistoryRepoJSONSuffix = ".json"
)

type (
	History struct {
		l      *zap.Logger
		max    int
		varDir string
	}
	HistoryOption func(*History)
)

// ------------------------------------------------------------------------------------------------
// ~ Options
// ------------------------------------------------------------------------------------------------

func HistoryWithMax(v int) HistoryOption {
	return func(o *History) {
		o.max = v
	}
}

func HistoryWithVarDir(v string) HistoryOption {
	return func(o *History) {
		o.varDir = v
	}
}

// ------------------------------------------------------------------------------------------------
// ~ Constructor
// ------------------------------------------------------------------------------------------------

func NewHistory(l *zap.Logger, opts ...HistoryOption) *History {
	inst := &History{
		l:      l,
		max:    2,
		varDir: "/var/lib/contentserver",
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
	var (
		// historiy file name
		filename = path.Join(h.varDir, HistoryRepoJSONPrefix+time.Now().Format(time.RFC3339Nano)+HistoryRepoJSONSuffix)
		err      = os.WriteFile(filename, jsonBytes, 0600)
	)
	if err != nil {
		return err
	}

	h.l.Debug("adding content backup", zap.String("file", filename))

	// current filename
	err = os.WriteFile(h.GetCurrentFilename(), jsonBytes, 0600)
	if err != nil {
		return err
	}

	err = h.cleanup()
	if err != nil {
		h.l.Error("an error occurred while cleaning up my history", zap.Error(err))
		return err
	}

	return nil
}

func (h *History) GetCurrentFilename() string {
	return path.Join(h.varDir, HistoryRepoJSONPrefix+"current"+HistoryRepoJSONSuffix)
}

func (h *History) GetCurrent(buf *bytes.Buffer) (err error) {
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
	fileInfos, err := os.ReadDir(h.varDir)
	if err != nil {
		return
	}
	currentName := h.GetCurrentFilename()
	for _, f := range fileInfos {
		if !f.IsDir() {
			filename := f.Name()
			if filename != currentName && (strings.HasPrefix(filename, HistoryRepoJSONPrefix) && strings.HasSuffix(filename, HistoryRepoJSONSuffix)) {
				files = append(files, path.Join(h.varDir, filename))
			}
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return
}

func (h *History) cleanup() error {
	files, err := h.getFilesForCleanup(h.max)
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
