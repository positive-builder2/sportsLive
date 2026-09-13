package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *goredis.Client
}

func Connect(addr string) *Client {
	return &Client{
		rdb: goredis.NewClient(&goredis.Options{Addr: addr}),
	}
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func matchKey(matchID string) string {
	return fmt.Sprintf("match:%s:state", matchID)
}

func (c *Client) SetMatchState(ctx context.Context, matchID string, state any) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, matchKey(matchID), data, 24*time.Hour).Err()
}

func (c *Client) GetMatchState(ctx context.Context, matchID string) ([]byte, error) {
	return c.rdb.Get(ctx, matchKey(matchID)).Bytes()
}

func (c *Client) SaveMatch(ctx context.Context, matchID string, metaData any) error {
	data, err := json.Marshal(metaData)
	if err != nil {
		return err
	}
	pipe := c.rdb.Pipeline()
	pipe.SAdd(ctx, "matches:index", matchID)
	pipe.Set(ctx, fmt.Sprintf("match:%s:meta", matchID), data, 0)
	_, err = pipe.Exec(ctx)
	return err
}

func (c *Client) GetMatches(ctx context.Context) ([]json.RawMessage, error) {
	ids, err := c.rdb.SMembers(ctx, "matches:index").Result()
	if err != nil && err != goredis.Nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []json.RawMessage{}, nil
	}

	var results []json.RawMessage
	for _, id := range ids {
		meta, err := c.rdb.Get(ctx, fmt.Sprintf("match:%s:meta", id)).Bytes()
		if err == nil {
			results = append(results, json.RawMessage(meta))
		}
	}
	return results, nil
}
