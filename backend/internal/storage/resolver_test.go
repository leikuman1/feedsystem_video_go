package storage

import (
	"context"
	"testing"
)

func TestIsOwnedMediaObjectKey(t *testing.T) {
	if !IsOwnedMediaObjectKey("videos/42/20260621/demo.mp4", MediaVideo, 42) {
		t.Fatal("expected owned video key")
	}
	for _, key := range []string{
		"videos/7/20260621/demo.mp4",
		"covers/42/20260621/demo.webp",
		"/videos/42/demo.mp4",
		"videos/42/../7/demo.mp4",
		"videos/42/",
	} {
		if IsOwnedMediaObjectKey(key, MediaVideo, 42) {
			t.Fatalf("unexpected owned key %q", key)
		}
	}
}

func TestURLResolverFallsBackToLegacyURL(t *testing.T) {
	resolver := NewURLResolver(nil, 0)
	got, err := resolver.Resolve(context.Background(), "", "https://legacy.example/video.mp4")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != "https://legacy.example/video.mp4" {
		t.Fatalf("got %q", got)
	}
}
