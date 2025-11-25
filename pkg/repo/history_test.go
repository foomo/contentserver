package repo

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/memblob"
)

func TestHistoryCurrent(t *testing.T) {
	var (
		ctx  = context.Background()
		h    = testHistory(t)
		test = []byte("test")
		b    bytes.Buffer
	)
	err := h.Add(ctx, test)
	require.NoError(t, err)
	err = h.GetCurrent(ctx, &b)
	require.NoError(t, err)
	if !bytes.Equal(b.Bytes(), test) {
		t.Fatalf("expected %q, got %q", string(test), b.String())
	}
}

func TestHistoryCleanup(t *testing.T) {
	ctx := context.Background()
	h := testHistory(t)
	for i := 0; i < 50; i++ {
		err := h.Add(ctx, []byte(fmt.Sprint(i)))
		require.NoError(t, err)
		time.Sleep(time.Millisecond * 5)
	}
	err := h.cleanup(ctx)
	require.NoError(t, err)
	files, err := h.getHistory(ctx)
	require.NoError(t, err)

	// Should only keep historyLimit (2) backups
	if len(files) != 2 {
		t.Fatal("history too long", len(files), "instead of", 2)
	}
}

func TestHistoryOrder(t *testing.T) {
	ctx := context.Background()
	h := testHistoryWithTestdata(t)

	files, err := h.getHistory(ctx)
	require.NoError(t, err)
	// Files should be sorted descending (newest first)
	assert.Equal(t, "contentserver-repo-2017-10-23.json", files[0])
	assert.Equal(t, "contentserver-repo-2017-10-22.json", files[1])
	assert.Equal(t, "contentserver-repo-2017-10-21.json", files[2])
}

func TestGetFilesForCleanup(t *testing.T) {
	ctx := context.Background()
	h := testHistoryWithTestdata(t)

	files, err := h.getFilesForCleanup(ctx, 2)
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "contentserver-repo-2017-10-21.json", files[0])
}

func TestHistoryWithStorage(t *testing.T) {
	ctx := context.Background()
	storage, err := NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)

	l := zaptest.NewLogger(t)
	h, err := NewHistory(l, HistoryWithStorage(storage), HistoryWithHistoryLimit(2))
	require.NoError(t, err)

	err = h.Add(ctx, []byte("test-data"))
	require.NoError(t, err)

	var buf bytes.Buffer
	err = h.GetCurrent(ctx, &buf)
	require.NoError(t, err)
	assert.Equal(t, "test-data", buf.String())

	// Verify storage was used
	data, err := storage.Read(ctx, CurrentKey)
	require.NoError(t, err)
	assert.Equal(t, []byte("test-data"), data)
}

func TestHistoryWithBlobStorage(t *testing.T) {
	ctx := context.Background()
	bucket, err := blob.OpenBucket(ctx, "mem://")
	require.NoError(t, err)
	defer bucket.Close()

	storage := NewBlobStorageFromBucket(bucket, "test-prefix")
	l := zaptest.NewLogger(t)
	h, err := NewHistory(l, HistoryWithStorage(storage), HistoryWithHistoryLimit(2))
	require.NoError(t, err)

	// Test Add
	err = h.Add(ctx, []byte("test-data"))
	require.NoError(t, err)

	// Test GetCurrent
	var buf bytes.Buffer
	err = h.GetCurrent(ctx, &buf)
	require.NoError(t, err)
	assert.Equal(t, "test-data", buf.String())

	// Test cleanup - add more entries
	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond * 5) // Ensure unique timestamps
		err = h.Add(ctx, []byte(fmt.Sprintf("data-%d", i)))
		require.NoError(t, err)
	}

	// Verify only historyLimit backups remain
	files, err := h.getHistory(ctx)
	require.NoError(t, err)
	assert.Len(t, files, 2, "should only keep historyLimit backups")
}

func TestHistoryClose(t *testing.T) {
	h := testHistory(t)
	err := h.Close()
	require.NoError(t, err)
}

func testHistory(t *testing.T) *History {
	t.Helper()
	l := zaptest.NewLogger(t)
	h, err := NewHistory(l, HistoryWithHistoryLimit(2), HistoryWithHistoryDir(t.TempDir()))
	require.NoError(t, err)
	return h
}

func testHistoryWithTestdata(t *testing.T) *History {
	t.Helper()
	l := zaptest.NewLogger(t)
	// Use the testdata directory which has pre-existing files
	storage, err := NewFilesystemStorage("testdata/order")
	require.NoError(t, err)
	h, err := NewHistory(l, HistoryWithStorage(storage), HistoryWithHistoryLimit(2))
	require.NoError(t, err)
	return h
}
