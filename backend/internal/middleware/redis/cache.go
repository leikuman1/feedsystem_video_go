package redis

import (
	"context"
	"errors"
	"time"
)

func (c *Client) GetBytes(ctx context.Context, key string) ([]byte, error) {
	if c == nil || c.rdb == nil {
		return nil, errors.New("redis client not initialized")
	}
	return c.rdb.Get(ctx, key).Bytes()
}

func (c *Client) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c == nil || c.rdb == nil {
		return errors.New("redis client not initialized")
	}
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

func (c *Client) Del(ctx context.Context, key string) error {
	if c == nil || c.rdb == nil {
		return errors.New("redis client not initialized")
	}
	return c.rdb.Del(ctx, key).Err()
}

func (c *Client) SAdd(ctx context.Context, key string, members ...any) error {
	if c == nil || c.rdb == nil {
		return errors.New("redis client not initialized")
	}
	return c.rdb.SAdd(ctx, key, members...).Err()
}

func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	if c == nil || c.rdb == nil {
		return nil, errors.New("redis client not initialized")
	}
	return c.rdb.SMembers(ctx, key).Result()
}

func (c *Client) SCard(ctx context.Context, key string) (int64, error) {
	if c == nil || c.rdb == nil {
		return 0, errors.New("redis client not initialized")
	}
	return c.rdb.SCard(ctx, key).Result()
}

func (c *Client) HSet(ctx context.Context, key string, values ...any) error {
	if c == nil || c.rdb == nil {
		return errors.New("redis client not initialized")
	}
	return c.rdb.HSet(ctx, key, values...).Err()
}

func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if c == nil || c.rdb == nil {
		return nil, errors.New("redis client not initialized")
	}
	return c.rdb.HGetAll(ctx, key).Result()
}

func (c *Client) DelByPattern(ctx context.Context, pattern string) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	iter := c.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		_ = c.rdb.Del(ctx, iter.Val())
	}
	return iter.Err()
}

func (c *Client) MGet(cacheCtx context.Context, cacheKeys ...string) ([]interface{}, error) {
	if c == nil || c.rdb == nil {
		return nil, errors.New("redis client not initialized")
	}
	return c.rdb.MGet(cacheCtx, cacheKeys...).Result()
}
