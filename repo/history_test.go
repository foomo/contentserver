package repo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func testHistory() *history {
	tempDir, err := ioutil.TempDir(os.TempDir(), "contentserver-history-test")
	if err != nil {
		panic(err)
	}
	return newHistory(tempDir)
}

func TestHistoryCurrent(t *testing.T) {
	h := testHistory()
	test := []byte("test")
	h.add(test)
	current, err := h.getCurrent()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(current, test) != 0 {
		t.Fatal(fmt.Sprintf("expected %q, got %q", string(test), string(current)))
	}
}

func TestHistoryCleanup(t *testing.T) {
	h := testHistory()
	for i := 0; i < 50; i++ {
		h.add([]byte(fmt.Sprint(i)))
		time.Sleep(time.Millisecond * 5)
	}
	h.cleanup()
	files, err := h.getHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != maxHistoryVersions {
		t.Fatal("history too long", len(files), "instead of", maxHistoryVersions)
	}
}
