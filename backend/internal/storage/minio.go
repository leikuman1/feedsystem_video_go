package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"feedsystem_video_go/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

const incompleteUploadExpiryDays = 1

type MinIOStorage struct {
	client       *minio.Client
	core         *minio.Core
	publicClient *minio.Client
	bucket       string
	region       string
}

func NewMinIOStorage(cfg config.MinIOConfig) (*MinIOStorage, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	internalOptions := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	}
	client, err := minio.New(cfg.Endpoint, internalOptions)
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}
	core, err := minio.NewCore(cfg.Endpoint, internalOptions)
	if err != nil {
		return nil, fmt.Errorf("create minio multipart client: %w", err)
	}
	publicClient, err := minio.New(cfg.PublicEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.PublicUseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio public client: %w", err)
	}

	return &MinIOStorage{
		client:       client,
		core:         core,
		publicClient: publicClient,
		bucket:       cfg.Bucket,
		region:       cfg.Region,
	}, nil
}

func (s *MinIOStorage) EnsureReady(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check minio bucket %q: %w", s.bucket, err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: s.region}); err != nil {
			return fmt.Errorf("create minio bucket %q: %w", s.bucket, err)
		}
	}

	rule := lifecycle.Rule{
		ID:     "abort-incomplete-multipart-uploads",
		Status: "Enabled",
		AbortIncompleteMultipartUpload: lifecycle.AbortIncompleteMultipartUpload{
			DaysAfterInitiation: lifecycle.ExpirationDays(incompleteUploadExpiryDays),
		},
	}
	if err := s.client.SetBucketLifecycle(ctx, s.bucket, &lifecycle.Configuration{Rules: []lifecycle.Rule{rule}}); err != nil {
		return fmt.Errorf("configure minio lifecycle: %w", err)
	}
	return nil
}

func (s *MinIOStorage) PutObject(
	ctx context.Context,
	key string,
	reader io.Reader,
	size int64,
	contentType string,
) (ObjectInfo, error) {
	info, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("put object %q: %w", key, err)
	}
	return ObjectInfo{Key: key, ETag: info.ETag, Size: info.Size}, nil
}

func (s *MinIOStorage) NewMultipartUpload(ctx context.Context, key, contentType string) (string, error) {
	uploadID, err := s.core.NewMultipartUpload(ctx, s.bucket, key, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("start multipart upload %q: %w", key, err)
	}
	return uploadID, nil
}

func (s *MinIOStorage) PutObjectPart(
	ctx context.Context,
	key, uploadID string,
	partNumber int,
	reader io.Reader,
	size int64,
) (CompletedPart, error) {
	part, err := s.core.PutObjectPart(
		ctx,
		s.bucket,
		key,
		uploadID,
		partNumber,
		reader,
		size,
		minio.PutObjectPartOptions{},
	)
	if err != nil {
		return CompletedPart{}, fmt.Errorf("put object part %d for %q: %w", partNumber, key, err)
	}
	return CompletedPart{PartNumber: part.PartNumber, ETag: part.ETag}, nil
}

func (s *MinIOStorage) CompleteMultipartUpload(
	ctx context.Context,
	key, uploadID string,
	parts []CompletedPart,
	contentType string,
) (ObjectInfo, error) {
	completeParts := make([]minio.CompletePart, 0, len(parts))
	for _, part := range parts {
		completeParts = append(completeParts, minio.CompletePart{
			PartNumber: part.PartNumber,
			ETag:       part.ETag,
		})
	}
	info, err := s.core.CompleteMultipartUpload(
		ctx,
		s.bucket,
		key,
		uploadID,
		completeParts,
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("complete multipart upload %q: %w", key, err)
	}
	return ObjectInfo{Key: key, ETag: info.ETag, Size: info.Size}, nil
}

func (s *MinIOStorage) AbortMultipartUpload(ctx context.Context, key, uploadID string) error {
	if err := s.core.AbortMultipartUpload(ctx, s.bucket, key, uploadID); err != nil {
		return fmt.Errorf("abort multipart upload %q: %w", key, err)
	}
	return nil
}

func (s *MinIOStorage) RemoveObject(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove object %q: %w", key, err)
	}
	return nil
}

func (s *MinIOStorage) PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if key == "" {
		return "", nil
	}
	signedURL, err := s.publicClient.PresignedGetObject(ctx, s.bucket, key, expiry, url.Values{})
	if err != nil {
		return "", fmt.Errorf("presign object %q: %w", key, err)
	}
	return signedURL.String(), nil
}
