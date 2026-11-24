//go:build unit

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFeishuSetupStoreLockAcrossReplicasAndExpiry(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer func() { _ = client.Close() }()
	first := NewFeishuSetupStore(client)
	second := NewFeishuSetupStore(client)
	ctx := context.Background()
	release, err := first.Lock(ctx)
	require.NoError(t, err)
	_, err = second.Lock(ctx)
	require.Error(t, err)
	// An expired lock's owner cannot release a subsequent owner's lease.
	server.FastForward(3 * time.Minute)
	next, err := second.Lock(ctx)
	require.NoError(t, err)
	release()
	_, err = first.Lock(ctx)
	require.Error(t, err)
	next()
	require.NoError(t, first.Set(ctx, "opaque", []byte("encrypted-state"), time.Minute))
	value, err := second.Get(ctx, "opaque")
	require.NoError(t, err)
	require.Equal(t, []byte("encrypted-state"), value)
	server.FastForward(2 * time.Minute)
	_, err = first.Get(ctx, "opaque")
	require.Error(t, err)
}

func TestFeishuDirectoryProgressCannotReplaceNewerWorker(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer func() { _ = client.Close() }()
	store := &feishuSetupStore{redis: client}
	ctx := context.Background()
	require.NoError(t, store.Set(ctx, "directory:active", []byte(`{"id":"new","status":"reading"}`), time.Hour))
	ok, err := store.SaveDirectoryProgress(ctx, "old", []byte(`{"id":"old","status":"succeeded"}`), time.Hour)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = store.SaveDirectoryProgress(ctx, "new", []byte(`{"id":"new","status":"succeeded"}`), time.Hour)
	require.NoError(t, err)
	require.True(t, ok)
	value, err := store.Get(ctx, "directory:active")
	require.NoError(t, err)
	require.JSONEq(t, `{"id":"new","status":"succeeded"}`, string(value))
}
