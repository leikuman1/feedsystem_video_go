package storage

import (
	"context"
	"net/url"
	"testing"
	"time"

	"feedsystem_video_go/internal/config"
)

func testMinIOConfig() config.MinIOConfig {
	return config.MinIOConfig{
		Endpoint:               "minio:9000",
		PublicEndpoint:         "media.example.com",
		AccessKey:              "access",
		SecretKey:              "secret",
		Bucket:                 "feedsystem-media",
		Region:                 "us-east-1",
		UseSSL:                 false,
		PublicUseSSL:           true,
		SignedURLExpirySeconds: 7200,
	}
}

func TestNewMinIOStorageRejectsInvalidConfig(t *testing.T) {
	cfg := testMinIOConfig()
	cfg.Bucket = ""

	if _, err := NewMinIOStorage(cfg); err == nil {
		t.Fatal("expected invalid config error")
	}
}

func TestPresignedGetURLUsesPublicEndpoint(t *testing.T) {
	store, err := NewMinIOStorage(testMinIOConfig())
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}

	rawURL, err := store.PresignedGetURL(context.Background(), "videos/1/demo.mp4", 2*time.Hour)
	if err != nil {
		t.Fatalf("presign: %v", err)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if parsed.Scheme != "https" {
		t.Fatalf("scheme = %q, want https", parsed.Scheme)
	}
	if parsed.Host != "media.example.com" {
		t.Fatalf("host = %q, want media.example.com", parsed.Host)
	}
	if parsed.Path != "/feedsystem-media/videos/1/demo.mp4" {
		t.Fatalf("path = %q", parsed.Path)
	}
	if parsed.Query().Get("X-Amz-Signature") == "" {
		t.Fatal("missing signature")
	}
}
