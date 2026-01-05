package flags

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	indexKey    = "flags:index"
	valuePrefix = "flags:"
)

var keyRe = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,128}$`)

type Store struct {
	client redis.Cmdable
}

func NewStore(client redis.Cmdable) (*Store, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	return &Store{client: client}, nil
}

func ValidateKey(key string) error {
	if !keyRe.MatchString(key) {
		return fmt.Errorf("invalid flag key")
	}
	return nil
}

func (s *Store) Upsert(ctx context.Context, key string, value bool) (*Flag, error) {
	if err := ValidateKey(key); err != nil {
		return nil, err
	}

	flag := &Flag{Key: key, Value: value, UpdatedAt: time.Now().UTC()}
	b, err := json.Marshal(flag)
	if err != nil {
		return nil, fmt.Errorf("marshal flag: %w", err)
	}

	pipe := s.client.TxPipeline()
	pipe.Set(ctx, flagKey(key), b, 0)
	pipe.SAdd(ctx, indexKey, key)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("upsert flag: %w", err)
	}

	return flag, nil
}

func (s *Store) Get(ctx context.Context, key string) (*Flag, error) {
	if err := ValidateKey(key); err != nil {
		return nil, err
	}

	val, err := s.client.Get(ctx, flagKey(key)).Result()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get flag: %w", err)
	}

	var f Flag
	if err := json.Unmarshal([]byte(val), &f); err != nil {
		return nil, fmt.Errorf("unmarshal flag: %w", err)
	}
	return &f, nil
}

func (s *Store) List(ctx context.Context) ([]*Flag, error) {
	keys, err := s.client.SMembers(ctx, indexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("list flags index: %w", err)
	}
	if len(keys) == 0 {
		return []*Flag{}, nil
	}

	redisKeys := make([]string, 0, len(keys))
	for _, k := range keys {
		if err := ValidateKey(k); err != nil {
			continue
		}
		redisKeys = append(redisKeys, flagKey(k))
	}
	if len(redisKeys) == 0 {
		return []*Flag{}, nil
	}

	vals, err := s.client.MGet(ctx, redisKeys...).Result()
	if err != nil {
		return nil, fmt.Errorf("mget flags: %w", err)
	}

	out := make([]*Flag, 0, len(vals))
	for _, v := range vals {
		if v == nil {
			continue
		}
		s, ok := v.(string)
		if !ok {
			continue
		}
		var f Flag
		if err := json.Unmarshal([]byte(s), &f); err != nil {
			continue
		}
		out = append(out, &f)
	}

	return out, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	if err := ValidateKey(key); err != nil {
		return err
	}

	pipe := s.client.TxPipeline()
	pipe.Del(ctx, flagKey(key))
	pipe.SRem(ctx, indexKey, key)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("delete flag: %w", err)
	}

	return nil
}

func flagKey(key string) string {
	return valuePrefix + key
}
