package storage

import (
	"strings"
	"testing"
	"time"
)

func TestNewMediaObjectKey(t *testing.T) {
	now := time.Date(2026, time.June, 21, 12, 0, 0, 0, time.UTC)

	videoKey, err := NewMediaObjectKey(MediaVideo, 42, ".MP4", now)
	if err != nil {
		t.Fatalf("video key: %v", err)
	}
	if !strings.HasPrefix(videoKey, "videos/42/20260621/") || !strings.HasSuffix(videoKey, ".mp4") {
		t.Fatalf("unexpected video key %q", videoKey)
	}

	avatarKey, err := NewMediaObjectKey(MediaAvatar, 42, ".webp", now)
	if err != nil {
		t.Fatalf("avatar key: %v", err)
	}
	if avatarKey != "avatars/42/current.webp" {
		t.Fatalf("avatar key = %q", avatarKey)
	}
}

func TestNewMediaObjectKeyRejectsInvalidInput(t *testing.T) {
	if _, err := NewMediaObjectKey(MediaKind("unknown"), 1, ".mp4", time.Now()); err == nil {
		t.Fatal("expected unsupported kind error")
	}
	if _, err := NewMediaObjectKey(MediaVideo, 1, "../mp4", time.Now()); err == nil {
		t.Fatal("expected invalid extension error")
	}
}
