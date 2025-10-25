package redis

import (
	"context"
	"fmt"
	"time"
	"ulak/internal/config"
	"ulak/internal/store/keyval"

	"github.com/redis/go-redis/v9"
)

type redisStore struct {
	rdb *redis.Client
}

func buildKey(prefix, key string) string {
	return prefix + "." + key
}

func (rs *redisStore) GetMessageDelivered(ctx context.Context, messageId string) (string, error) {
	key := buildKey("message-delivered", messageId)

	val, err := rs.rdb.Get(ctx, key).Result()

	if err != nil {
		return "", err
	}

	return val, nil
}

func (rs *redisStore) SetMessageDelivered(ctx context.Context, messageId, value string, ttl int) error {
	key := buildKey("message-delivered", messageId)
	err := rs.rdb.Set(ctx, key, value, time.Duration(ttl)*time.Second).Err()
	return err
}

func NewRedisStore(cfg config.Redis) (keyval.KeyValueStore, error) {
	c := redis.NewClient(&redis.Options{
		Addr:            cfg.Address,
		Username:        cfg.Username,
		Password:        cfg.Password,
		DB:              cfg.DB,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConnections,
		ConnMaxLifetime: cfg.MaxConnectionAge,
		ConnMaxIdleTime: cfg.IdleTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	if err := c.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &redisStore{
		rdb: c,
	}, nil
}
