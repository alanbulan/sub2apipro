package repository

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

type feishuSetupStore struct{ redis *redis.Client }

func NewFeishuSetupStore(r *redis.Client) service.FeishuSetupStore { return &feishuSetupStore{r} }
func (s *feishuSetupStore) Get(ctx context.Context, key string) ([]byte, error) {
	return s.redis.Get(ctx, "feishu:setup:"+key).Bytes()
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
