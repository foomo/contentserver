package repo

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestHistoryCurrent(t *testing.T) {
	var (
		h    = testHistory(t)
		test = []byte("test")
		b    bytes.Buffer
	)
	err := h.Add(test)
	require.NoError(t, err)
	err = h.GetCurrent(&b)
	require.NoError(t, err)
	if !bytes.Equal(b.Bytes(), test) {
		t.Fatalf("expected %q, got %q", string(test), b.String())
	}
}

func TestHistoryCleanup(t *testing.T) {
	h := testHistory(t)
	for i := 0; i < 50; i++ {
		err := h.Add([]byte(fmt.Sprint(i)))
		require.NoError(t, err)
		time.Sleep(time.Millisecond * 5)
	}
	err := h.cleanup()
	require.NoError(t, err)
	files, err := h.getHistory()
	require.NoError(t, err)

	// -1 for ignoring the current content backup file
	if len(files)-1 != 2 {
		t.Fatal("history too long", len(files), "instead of", 2)
	}
}

func TestHistoryOrder(t *testing.T) {
	h := testHistory(t)
	h.historyDir = "testdata/order"

	files, err := h.getHistory()
	require.NoError(t, err)
	assert.Equal(t, "testdata/order/contentserver-repo-current.json", files[0])
	assert.Equal(t, "testdata/order/contentserver-repo-2017-10-23.json", files[1])
	assert.Equal(t, "testdata/order/contentserver-repo-2017-10-22.json", files[2])
	assert.Equal(t, "testdata/order/contentserver-repo-2017-10-21.json", files[3])
}

func TestGetFilesForCleanup(t *testing.T) {
	h := testHistory(t)
	h.historyDir = "testdata/order"

	files, err := h.getFilesForCleanup(2)
	require.NoError(t, err)
	assert.Equal(t, "testdata/order/contentserver-repo-2017-10-21.json", files[0])
}

func testHistory(t *testing.T) *History {
	t.Helper()
	l := zaptest.NewLogger(t)
	return NewHistory(l, HistoryWithHistoryLimit(2), HistoryWithHistoryDir(t.TempDir()))
}
