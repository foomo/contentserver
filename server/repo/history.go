package repo

import (
	"io/ioutil"
	"path"
	"time"
)

const historyRepoJSONPrefix = "contentserver-repo-"
const historyRepoJSONSuffix = ".json"

type history struct {
	varDir string
}

func newHistory(varDir string) *history {
	return &history{
		varDir: varDir,
	}
}

func (h *history) add(jsonBytes []byte) error {
	// historic file name
	filename := path.Join(h.varDir, historyRepoJSONPrefix+time.Now().Format(time.RFC3339Nano)+historyRepoJSONSuffix)
	err := ioutil.WriteFile(filename, jsonBytes, 0644)
	if err != nil {
		return err
	}
	// current filename
	return ioutil.WriteFile(h.getCurrentFilename(), jsonBytes, 0644)
}

func (h *history) getCurrentFilename() string {
	return historyRepoJSONPrefix + "current" + historyRepoJSONSuffix
}

func (h *history) getCurrent() (jsonBytes []byte, err error) {
	return ioutil.ReadFile(h.getCurrentFilename())
}
