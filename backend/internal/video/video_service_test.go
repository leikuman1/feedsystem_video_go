package video

import (
	"context"
	"testing"
	"time"

	rediscache "feedsystem_video_go/internal/middleware/redis"

	miniredis "github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func TestVideoServiceGetDetailKeepsCachedObjectKeys(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	defer mr.Close()

	cache := rediscache.NewClient(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}), "v1:")
	defer cache.Close()

	ctx := context.Background()
	v := &Video{
		ID:             1,
		AuthorID:       2,
		Username:       "interviewer",
		Title:          "demo",
		PlayObjectKey:  "videos/2/20260626/video.mp4",
		CoverObjectKey: "covers/2/20260626/cover.jpg",
		CreateTime:     time.Now(),
	}
	b, err := MarshalVideoCache(v)
	if err != nil {
		t.Fatalf("marshal cache: %v", err)
	}
	if err := cache.SetBytes(ctx, cache.Key("video:detail:id=%d", v.ID), b, time.Hour); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	service := &VideoService{cache: cache, cacheTTL: time.Hour}
	got, err := service.GetDetail(ctx, v.ID)
	if err != nil {
		t.Fatalf("get detail: %v", err)
	}
	if got.PlayObjectKey != v.PlayObjectKey || got.CoverObjectKey != v.CoverObjectKey {
		t.Fatalf("cached media keys not preserved: %#v", got)
	}
}
