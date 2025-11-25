package repo

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/memblob"
)

func newTestBlobStorage(t *testing.T, prefix string) *BlobStorage {
	t.Helper()
	ctx := context.Background()
	bucket, err := blob.OpenBucket(ctx, "mem://")
	require.NoError(t, err)
	t.Cleanup(func() { bucket.Close() })
	return NewBlobStorageFromBucket(bucket, prefix)
}

func TestBlobStorage_Write(t *testing.T) {
	ctx := context.Background()
	storage := newTestBlobStorage(t, "")

	err := storage.Write(ctx, "test-key", []byte("test-data"))
	require.NoError(t, err)

	data, err := storage.Read(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("test-data"), data)
}

func TestBlobStorage_Write_Overwrite(t *testing.T) {
	ctx := context.Background()
	storage := newTestBlobStorage(t, "")

	err := storage.Write(ctx, "test-key", []byte("original"))
	require.NoError(t, err)

	err = storage.Write(ctx, "test-key", []byte("updated"))
	require.NoError(t, err)

	data, err := storage.Read(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("updated"), data)
}

func TestBlobStorage_Write_WithPrefix(t *testing.T) {
	ctx := context.Background()
	storage := newTestBlobStorage(t, "my-prefix/")

	err := storage.Write(ctx, "test-key", []byte("test-data"))
	require.NoError(t, err)

	data, err := storage.Read(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("test-data"), data)
}

func TestBlobStorage_Read(t *testing.T) {
	ctx := context.Background()
	storage := newTestBlobStorage(t, "")

	err := storage.Write(ctx, "test-key", []byte("test-data"))
	require.NoError(t, err)

	data, err := storage.Read(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("test-data"), data)
}

func TestBlobStorage_Read_NotFound(t *testing.T) {
	ctx := context.Background()
	storage := newTestBlobStorage(t, "")

	_, err := storage.Read(ctx, "nonexistent-key")
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestBlobStorage_List(t *testing.T) {
	ctx := context.Background()
	storage := newTestBlobStorage(t, "")

	err := storage.Write(ctx, "prefix-a", []byte("a"))
	require.NoError(t, err)
	err = storage.Write(ctx, "prefix-b", []byte("b"))
	require.NoError(t, err)
	err = storage.Write(ctx, "prefix-c", []byte("c"))
	require.NoError(t, err)
	err = storage.Write(ctx, "other-key", []byte("other"))
	require.NoError(t, err)

	keys, err := storage.List(ctx, "prefix-")
	require.NoError(t, err)
	assert.Len(t, keys, 3)
	// Should be sorted descending
	assert.Equal(t, "prefix-c", keys[0])
	assert.Equal(t, "prefix-b", keys[1])
	assert.Equal(t, "prefix-a", keys[2])
}

func TestBlobStorage_ListWithPrefix(t *testing.T) {
	ctx := context.Background()
	storage := newTestBlobStorage(t, "bucket-prefix/")

	err := storage.Write(ctx, "prefix-a", []byte("a"))
	require.NoError(t, err)
	err = storage.Write(ctx, "prefix-b", []byte("b"))
	require.NoError(t, err)
	err = storage.Write(ctx, "other-key", []byte("other"))
	require.NoError(t, err)

	keys, err := storage.List(ctx, "prefix-")
	require.NoError(t, err)
	assert.Len(t, keys, 2)
	// Keys should not include the bucket prefix
	assert.Equal(t, "prefix-b", keys[0])
	assert.Equal(t, "prefix-a", keys[1])
}

func TestBlobStorage_List_Empty(t *testing.T) {
	ctx := context.Background()
	storage := newTestBlobStorage(t, "")

	keys, err := storage.List(ctx, "nonexistent-")
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestBlobStorage_Delete(t *testing.T) {
	ctx := context.Background()
	storage := newTestBlobStorage(t, "")

	err := storage.Write(ctx, "test-key", []byte("test-data"))
	require.NoError(t, err)

	err = storage.Delete(ctx, "test-key")
	require.NoError(t, err)

	_, err = storage.Read(ctx, "test-key")
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestBlobStorage_Delete_NotFound(t *testing.T) {
	ctx := context.Background()
	storage := newTestBlobStorage(t, "")

	// Delete should be idempotent - no error for non-existent key
	err := storage.Delete(ctx, "nonexistent-key")
	require.NoError(t, err)
}

func TestBlobStorage_Delete_WithPrefix(t *testing.T) {
	ctx := context.Background()
	storage := newTestBlobStorage(t, "my-prefix/")

	err := storage.Write(ctx, "test-key", []byte("test-data"))
	require.NoError(t, err)

	err = storage.Delete(ctx, "test-key")
	require.NoError(t, err)

	_, err = storage.Read(ctx, "test-key")
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestBlobStorage_ConcurrentOperations(t *testing.T) {
	ctx := context.Background()
	storage := newTestBlobStorage(t, "")

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "concurrent-key"
			data := []byte("data")
			_ = storage.Write(ctx, key, data)
			_, _ = storage.Read(ctx, key)
			_, _ = storage.List(ctx, "concurrent-")
		}(i)
	}
	wg.Wait()
}

func TestBlobStorage_LargeBlob(t *testing.T) {
	ctx := context.Background()
	storage := newTestBlobStorage(t, "")

	// Create a 1MB blob
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	err := storage.Write(ctx, "large-key", largeData)
	require.NoError(t, err)

	data, err := storage.Read(ctx, "large-key")
	require.NoError(t, err)
	assert.Equal(t, largeData, data)
}
