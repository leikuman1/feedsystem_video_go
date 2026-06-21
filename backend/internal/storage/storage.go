package storage

import (
	"context"
	"io"
	"time"
)

type ObjectInfo struct {
	Key  string
	ETag string
	Size int64
}

type CompletedPart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

type ObjectStorage interface {
	EnsureReady(ctx context.Context) error
	PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (ObjectInfo, error)
	NewMultipartUpload(ctx context.Context, key, contentType string) (string, error)
	PutObjectPart(ctx context.Context, key, uploadID string, partNumber int, reader io.Reader, size int64) (CompletedPart, error)
	CompleteMultipartUpload(ctx context.Context, key, uploadID string, parts []CompletedPart, contentType string) (ObjectInfo, error)
	AbortMultipartUpload(ctx context.Context, key, uploadID string) error
	RemoveObject(ctx context.Context, key string) error
	PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}
