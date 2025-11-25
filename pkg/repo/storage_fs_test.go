package repo

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemStorage_Write(t *testing.T) {
	ctx := context.Background()
	storage, err := NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)

	err = storage.Write(ctx, "test-key", []byte("test-data"))
	require.NoError(t, err)

	data, err := storage.Read(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("test-data"), data)
}

func TestFilesystemStorage_Write_Overwrite(t *testing.T) {
	ctx := context.Background()
	storage, err := NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)

	err = storage.Write(ctx, "test-key", []byte("original"))
	require.NoError(t, err)

	err = storage.Write(ctx, "test-key", []byte("updated"))
	require.NoError(t, err)

	data, err := storage.Read(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("updated"), data)
}

func TestFilesystemStorage_Read(t *testing.T) {
	ctx := context.Background()
	storage, err := NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)

	err = storage.Write(ctx, "test-key", []byte("test-data"))
	require.NoError(t, err)

	data, err := storage.Read(ctx, "test-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("test-data"), data)
}

func TestFilesystemStorage_Read_NotFound(t *testing.T) {
	ctx := context.Background()
	storage, err := NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)

	_, err = storage.Read(ctx, "nonexistent-key")
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestFilesystemStorage_List(t *testing.T) {
	ctx := context.Background()
	storage, err := NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)

	err = storage.Write(ctx, "prefix-a", []byte("a"))
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

func TestFilesystemStorage_List_Empty(t *testing.T) {
	ctx := context.Background()
	storage, err := NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)

	keys, err := storage.List(ctx, "nonexistent-")
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestFilesystemStorage_Delete(t *testing.T) {
	ctx := context.Background()
	storage, err := NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)

	err = storage.Write(ctx, "test-key", []byte("test-data"))
	require.NoError(t, err)

	err = storage.Delete(ctx, "test-key")
	require.NoError(t, err)

	_, err = storage.Read(ctx, "test-key")
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestFilesystemStorage_Delete_NotFound(t *testing.T) {
	ctx := context.Background()
	storage, err := NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)

	// Delete should be idempotent - no error for non-existent key
	err = storage.Delete(ctx, "nonexistent-key")
	require.NoError(t, err)
}

func TestFilesystemStorage_ConcurrentOperations(t *testing.T) {
	ctx := context.Background()
	storage, err := NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)

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

func TestFilesystemStorage_Close(t *testing.T) {
	storage, err := NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)

	err = storage.Close()
	require.NoError(t, err)
}
