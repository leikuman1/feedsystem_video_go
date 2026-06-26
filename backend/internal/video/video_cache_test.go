package video

import (
	"encoding/json"
	"testing"
	"time"
)

func TestVideoCachePreservesObjectKeys(t *testing.T) {
	createdAt := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	src := &Video{
		ID:             1,
		AuthorID:       2,
		Username:       "interviewer",
		Title:          "demo",
		Description:    "minio video",
		PlayObjectKey:  "videos/2/20260626/video.mp4",
		CoverObjectKey: "covers/2/20260626/cover.jpg",
		CreateTime:     createdAt,
		LikesCount:     7,
		Popularity:     9,
	}

	b, err := MarshalVideoCache(src)
	if err != nil {
		t.Fatalf("marshal video cache: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal raw cache: %v", err)
	}
	if raw["play_object_key"] != src.PlayObjectKey {
		t.Fatalf("play_object_key not preserved: %#v", raw["play_object_key"])
	}
	if raw["cover_object_key"] != src.CoverObjectKey {
		t.Fatalf("cover_object_key not preserved: %#v", raw["cover_object_key"])
	}

	got, ok := UnmarshalVideoCache(b)
	if !ok {
		t.Fatal("expected valid cache payload")
	}
	if got.PlayObjectKey != src.PlayObjectKey || got.CoverObjectKey != src.CoverObjectKey {
		t.Fatalf("media keys lost after round trip: %#v", got)
	}
	if !got.CreateTime.Equal(createdAt) {
		t.Fatalf("create_time mismatch: got %s want %s", got.CreateTime, createdAt)
	}
}

func TestVideoCacheRejectsPayloadWithoutMediaReferences(t *testing.T) {
	b := []byte(`{"id":1,"author_id":2,"username":"interviewer","title":"bad-cache"}`)

	if got, ok := UnmarshalVideoCache(b); ok {
		t.Fatalf("expected old bad cache to be rejected, got %#v", got)
	}
}

func TestVideoCacheAcceptsLegacyURLs(t *testing.T) {
	b := []byte(`{
		"id":1,
		"author_id":2,
		"username":"interviewer",
		"title":"legacy-cache",
		"play_url":"https://legacy.example/video.mp4",
		"cover_url":"https://legacy.example/cover.jpg"
	}`)

	got, ok := UnmarshalVideoCache(b)
	if !ok {
		t.Fatal("expected legacy url cache to remain valid")
	}
	if got.PlayURL == "" || got.CoverURL == "" {
		t.Fatalf("legacy urls not preserved: %#v", got)
	}
}
