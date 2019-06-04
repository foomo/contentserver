package repo

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	. "github.com/foomo/contentserver/logger"
	"go.uber.org/zap"
)

const (
	historyRepoJSONPrefix = "contentserver-repo-"
	historyRepoJSONSuffix = ".json"
)

var flagMaxHistoryVersions = flag.Int("max-history", 2, "set the maximum number of content backup files")

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

	Log.Info("adding content backup", zap.String("file", filename))

	// current filename
	err = ioutil.WriteFile(h.getCurrentFilename(), jsonBytes, 0644)
	if err != nil {
		return err
	}

	err = h.cleanup()
	if err != nil {
		Log.Error("an error occured while cleaning up my history", zap.Error(err))
		return err
	}

	return nil
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
	files, err := h.getFilesForCleanup(*flagMaxHistoryVersions)
	if err != nil {
		return err
	}

	for _, f := range files {
		Log.Info("removing outdated backup", zap.String("file", f))
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

	// fmt.Println("contentFiles:")
	// for _, f := range contentFiles {
	// 	fmt.Println(f)
	// }

	// -1 to remove the current backup file from the number of items
	// so that only files with a timestamp are compared
	if len(contentFiles)-1 > historyVersions {
		for i := historyVersions + 1; i < len(contentFiles); i++ {
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
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(buf, f)
	return err
}
