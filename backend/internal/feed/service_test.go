package feed

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	rediscache "feedsystem_video_go/internal/middleware/redis"
	"feedsystem_video_go/internal/storage"
	"feedsystem_video_go/internal/video"

	miniredis "github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

type fakeFeedStorage struct{}

func (fakeFeedStorage) EnsureReady(context.Context) error { return nil }

func (fakeFeedStorage) PutObject(context.Context, string, io.Reader, int64, string) (storage.ObjectInfo, error) {
	return storage.ObjectInfo{}, nil
}

func (fakeFeedStorage) NewMultipartUpload(context.Context, string, string) (string, error) {
	return "", nil
}

func (fakeFeedStorage) PutObjectPart(context.Context, string, string, int, io.Reader, int64) (storage.CompletedPart, error) {
	return storage.CompletedPart{}, nil
}

func (fakeFeedStorage) CompleteMultipartUpload(context.Context, string, string, []storage.CompletedPart, string) (storage.ObjectInfo, error) {
	return storage.ObjectInfo{}, nil
}

func (fakeFeedStorage) AbortMultipartUpload(context.Context, string, string) error { return nil }

func (fakeFeedStorage) RemoveObject(context.Context, string) error { return nil }

func (fakeFeedStorage) PresignedGetURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "signed://" + key, nil
}

func TestCachedFollowingFeedRegeneratesSignedURLs(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	cache := rediscache.NewClient(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}), "v1:")
	defer cache.Close()

	ctx := context.Background()
	media := &video.Video{
		ID:             1,
		AuthorID:       2,
		Username:       "interviewer",
		Title:          "demo",
		PlayObjectKey:  "videos/2/20260626/video.mp4",
		CoverObjectKey: "covers/2/20260626/cover.jpg",
		CreateTime:     time.Now(),
	}
	b, err := video.MarshalVideoCache(media)
	if err != nil {
		t.Fatalf("marshal video cache: %v", err)
	}
	if err := cache.SetBytes(ctx, cache.Key("video:entity:%d", media.ID), b, time.Hour); err != nil {
		t.Fatalf("set video entity cache: %v", err)
	}

	service := &FeedService{
		likeRepo:      video.NewLikeRepository(nil),
		rediscache:    cache,
		cacheTTL:      time.Hour,
		mediaResolver: storage.NewURLResolver(fakeFeedStorage{}, time.Hour),
	}
	cacheKey := cache.Key("feed:listByFollowing:v2:test")
	service.setCachedFollowingFeed(ctx, cacheKey, followingFeedCache{
		Version:  followingFeedCacheVersion,
		VideoIDs: []uint{media.ID},
		NextTime: 123,
		HasMore:  false,
	})

	resp, ok, err := service.getCachedFollowingFeed(ctx, cacheKey, 0)
	if err != nil {
		t.Fatalf("get following cache: %v", err)
	}
	if !ok {
		t.Fatal("expected following cache hit")
	}
	if len(resp.VideoList) != 1 {
		t.Fatalf("expected 1 video, got %d", len(resp.VideoList))
	}
	if resp.VideoList[0].PlayURL != "signed://"+media.PlayObjectKey {
		t.Fatalf("unexpected play_url: %s", resp.VideoList[0].PlayURL)
	}
	if resp.VideoList[0].CoverURL != "signed://"+media.CoverObjectKey {
		t.Fatalf("unexpected cover_url: %s", resp.VideoList[0].CoverURL)
	}
}

func TestCachedFollowingFeedRejectsOldResponseCache(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	cache := rediscache.NewClient(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}), "v1:")
	defer cache.Close()

	ctx := context.Background()
	cacheKey := cache.Key("feed:listByFollowing:v2:old-response")
	oldResponse := ListByFollowingResponse{
		VideoList: []FeedVideoItem{{ID: 1, PlayURL: ""}},
		NextTime:  123,
		HasMore:   false,
	}
	b, err := json.Marshal(oldResponse)
	if err != nil {
		t.Fatalf("marshal old response: %v", err)
	}
	if err := cache.SetBytes(ctx, cacheKey, b, time.Hour); err != nil {
		t.Fatalf("set old response cache: %v", err)
	}

	service := &FeedService{rediscache: cache, cacheTTL: time.Hour}
	if _, ok, err := service.getCachedFollowingFeed(ctx, cacheKey, 0); err != nil || ok {
		t.Fatalf("expected old response cache miss, ok=%v err=%v", ok, err)
	}
	if mr.Exists(cacheKey) {
		t.Fatal("expected invalid old response cache to be deleted")
	}
}
