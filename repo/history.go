package repo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

const (
	historyRepoJSONPrefix = "contentserver-repo-"
	historyRepoJSONSuffix = ".json"
	maxHistoryVersions    = 20
)

type history struct {
	varDir string
}

func newHistory(varDir string) *history {
	return &history{
		varDir: varDir,
	}
}

func (h *history) add(jsonBytes []byte) error {

	var (
		// historiy file name
		filename = path.Join(h.varDir, historyRepoJSONPrefix+time.Now().Format(time.RFC3339Nano)+historyRepoJSONSuffix)
		err      = ioutil.WriteFile(filename, jsonBytes, 0644)
	)
	if err != nil {
		return err
	}

	// current filename
	return ioutil.WriteFile(h.getCurrentFilename(), jsonBytes, 0644)
}

func (h *history) getHistory() (files []string, err error) {
	fileInfos, err := ioutil.ReadDir(h.varDir)
	if err != nil {
		return
	}
	currentName := h.getCurrentFilename()
	for _, f := range fileInfos {
		if !f.IsDir() {
			filename := f.Name()
			if filename != currentName && (strings.HasPrefix(filename, historyRepoJSONPrefix) && strings.HasSuffix(filename, historyRepoJSONSuffix)) {
				files = append(files, path.Join(h.varDir, filename))
			}
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return
}

func (h *history) cleanup() error {
	files, err := h.getFilesForCleanup(maxHistoryVersions)
	if err != nil {
		return err
	}
	for _, f := range files {
		err := os.Remove(f)
		if err != nil {
			return fmt.Errorf("could not remove file %s : %s", f, err.Error())
		}
	}

	return nil
}

func (h *history) getFilesForCleanup(historyVersions int) (files []string, err error) {
	contentFiles, err := h.getHistory()
	if err != nil {
		return nil, errors.New("could not generate file cleanup list: " + err.Error())
	}
	if len(contentFiles) > historyVersions {
		for i := historyVersions; i < len(contentFiles); i++ {
			// ignore current repository file to fall back on
			if contentFiles[i] == h.getCurrentFilename() {
				continue
			}
			files = append(files, contentFiles[i])
		}
	}
	return files, nil
}

func (h *history) getCurrentFilename() string {
	return path.Join(h.varDir, historyRepoJSONPrefix+"current"+historyRepoJSONSuffix)
}

func (h *history) getCurrent(buf *bytes.Buffer) (err error) {
	f, err := os.Open(h.getCurrentFilename())
	defer f.Close()
	_, err = io.Copy(buf, f)
	return err
}
