package repository

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

type feishuSetupStore struct{ redis *redis.Client }

func NewFeishuSetupStore(r *redis.Client) service.FeishuSetupStore { return &feishuSetupStore{r} }
func (s *feishuSetupStore) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := s.redis.Get(ctx, "feishu:setup:"+key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, service.ErrFeishuSetupStateNotFound
	}
	return value, err
}

// SaveDirectoryProgress fences an expired worker from overwriting a newer job.
func (s *feishuSetupStore) SaveDirectoryProgress(ctx context.Context, id string, data []byte, ttl time.Duration) (bool, error) {
	updated, err := s.redis.Eval(ctx, `local raw = redis.call('get', KEYS[1]); if not raw then return 0 end; local job = cjson.decode(raw); if job.id ~= ARGV[1] then return 0 end; redis.call('set', KEYS[1], ARGV[2], 'PX', ARGV[3]); return 1`, []string{"feishu:setup:directory:active"}, id, data, ttl.Milliseconds()).Int()
	return updated == 1, err
}
func (s *feishuSetupStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return s.redis.Set(ctx, "feishu:setup:"+key, value, ttl).Err()
}
func (s *feishuSetupStore) Lock(ctx context.Context) (func(), error) {
	nonce := make([]byte, 24)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(nonce)
	ok, err := s.redis.SetNX(ctx, "feishu:settings:lock", token, 2*time.Minute).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, infraerrors.Conflict("FEISHU_SETUP_BUSY", "Feishu configuration is busy; retry shortly")
	}
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.redis.Eval(ctx, `if redis.call('get', KEYS[1]) == ARGV[1] then return redis.call('del', KEYS[1]) else return 0 end`, []string{"feishu:settings:lock"}, token).Err()
	}, nil
}
