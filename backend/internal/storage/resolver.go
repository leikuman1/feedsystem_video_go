package storage

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"
)

type URLResolver struct {
	storage ObjectStorage
	expiry  time.Duration
}

func NewURLResolver(objectStorage ObjectStorage, expiry time.Duration) *URLResolver {
	return &URLResolver{storage: objectStorage, expiry: expiry}
}

func (r *URLResolver) Resolve(ctx context.Context, objectKey, legacyURL string) (string, error) {
	if objectKey == "" {
		return legacyURL, nil
	}
	if r == nil || r.storage == nil {
		return "", fmt.Errorf("object storage is unavailable")
	}
	return r.storage.PresignedGetURL(ctx, objectKey, r.expiry)
}

func IsOwnedMediaObjectKey(key string, kind MediaKind, accountID uint) bool {
	if key == "" || path.Clean(key) != key || strings.HasPrefix(key, "/") {
		return false
	}
	prefix := fmt.Sprintf("%s/%d/", kind, accountID)
	return strings.HasPrefix(key, prefix) && len(key) > len(prefix)
}
