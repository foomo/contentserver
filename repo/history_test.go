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
	var (
		h    = testHistory()
		test = []byte("test")
		b    bytes.Buffer
	)
	err := h.add(test)
	if err != nil {
		t.Fatal("failed to add: ", err)
	}
	err = h.getCurrent(&b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b.Bytes(), test) {
		t.Fatalf("expected %q, got %q", string(test), b.String())
	}
}

func TestHistoryCleanup(t *testing.T) {
	h := testHistory()
	for i := 0; i < 50; i++ {
		err := h.add([]byte(fmt.Sprint(i)))
		if err != nil {
			t.Fatal("failed to add: ", err)
		}
		time.Sleep(time.Millisecond * 5)
	}
	err := h.cleanup()
	if err != nil {
		t.Fatal("failed to run cleanup: ", err)
	}
	files, err := h.getHistory()
	if err != nil {
		t.Fatal(err)
	}

	// -1 for ignoring the current content backup file
	if len(files)-1 != *flagMaxHistoryVersions {
		t.Fatal("history too long", len(files), "instead of", *flagMaxHistoryVersions)
	}
}

func TestHistoryOrder(t *testing.T) {
	h := testHistory()
	h.varDir = "testdata/order"

	files, err := h.getHistory()
	if err != nil {
		t.Fatal("error not expected")
	}
	assertStringEqual(t, "testdata/order/contentserver-repo-current.json", files[0])
	assertStringEqual(t, "testdata/order/contentserver-repo-2017-10-23.json", files[1])
	assertStringEqual(t, "testdata/order/contentserver-repo-2017-10-22.json", files[2])
	assertStringEqual(t, "testdata/order/contentserver-repo-2017-10-21.json", files[3])
}

func TestGetFilesForCleanup(t *testing.T) {
	h := testHistory()
	h.varDir = "testdata/order"

	files, err := h.getFilesForCleanup(2)
	if err != nil {
		t.Fatal("error not expected")
	}
	assertStringEqual(t, "testdata/order/contentserver-repo-2017-10-21.json", files[0])
}

func assertStringEqual(t *testing.T, expected, actual string) {
	if expected != actual {
		t.Errorf("expected string %s differs from the actual %s", expected, actual)
	}
}
