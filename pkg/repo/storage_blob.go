package repo

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"gocloud.dev/blob"
	"gocloud.dev/gcerrors"

	// Import GCS driver for production use
	_ "gocloud.dev/blob/gcsblob"
)

// BlobStorage implements Storage using gocloud.dev/blob.
// Currently supports GCS. Support for S3, Azure, and other cloud storage
// providers can be easily added by importing the corresponding driver.
type BlobStorage struct {
	bucket *blob.Bucket
	prefix string
}

// NewBlobStorage creates a new blob-backed storage.
// bucketURL should be in the format "gs://bucket-name" for GCS.
// prefix is an optional path prefix for all keys.
func NewBlobStorage(ctx context.Context, bucketURL, prefix string) (*BlobStorage, error) {
	bucket, err := blob.OpenBucket(ctx, bucketURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open bucket %q: %w", bucketURL, err)
	}
	// Normalize prefix: ensure trailing slash if non-empty
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return &BlobStorage{
		bucket: bucket,
		prefix: prefix,
	}, nil
}

// NewBlobStorageFromBucket creates a new blob-backed storage from an existing bucket.
// This is useful for testing with memblob.
func NewBlobStorageFromBucket(bucket *blob.Bucket, prefix string) *BlobStorage {
	// Normalize prefix: ensure trailing slash if non-empty
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return &BlobStorage{
		bucket: bucket,
		prefix: prefix,
	}
}

func (b *BlobStorage) fullKey(key string) string {
	if b.prefix == "" {
		return key
	}
	return b.prefix + key
}

func (b *BlobStorage) Write(ctx context.Context, key string, data []byte) error {
	if err := b.bucket.WriteAll(ctx, b.fullKey(key), data, nil); err != nil {
		return fmt.Errorf("failed to write blob %q: %w", key, err)
	}
	return nil
}

func (b *BlobStorage) Read(ctx context.Context, key string) ([]byte, error) {
	data, err := b.bucket.ReadAll(ctx, b.fullKey(key))
	if err != nil {
		if gcerrors.Code(err) == gcerrors.NotFound {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("failed to read blob %q: %w", key, err)
	}
	return data, nil
}

func (b *BlobStorage) List(ctx context.Context, prefix string) ([]string, error) {
	iter := b.bucket.List(&blob.ListOptions{
		Prefix: b.fullKey(prefix),
	})

	var keys []string
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list blobs with prefix %q: %w", prefix, err)
		}
		key := obj.Key
		if b.prefix != "" {
			// Skip keys that don't have our prefix (shouldn't happen, but be safe)
			if !strings.HasPrefix(key, b.prefix) {
				continue
			}
			key = strings.TrimPrefix(key, b.prefix)
		}
		keys = append(keys, key)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	return keys, nil
}

func (b *BlobStorage) Delete(ctx context.Context, key string) error {
	err := b.bucket.Delete(ctx, b.fullKey(key))
	if err != nil {
		if gcerrors.Code(err) == gcerrors.NotFound {
			return nil
		}
		return fmt.Errorf("failed to delete blob %q: %w", key, err)
	}
	return nil
}

func (b *BlobStorage) Close() error {
	if err := b.bucket.Close(); err != nil {
		return fmt.Errorf("failed to close bucket: %w", err)
	}
	return nil
}
