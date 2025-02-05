package database

import (
	"context"
	"encoding/json"
	"fmt"
	"streaming_video_service/pkg/logger"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// RedisRepository 定义接口
type RedisRepository[T any] interface {
	Set(ctx context.Context, key string, value T, ttl time.Duration) error
	Get(ctx context.Context, key string) (T, error)
	Del(ctx context.Context, key string) error
	GetTTL(ctx context.Context, key string) (int, error)
	ExtendTTL(ctx context.Context, key string, ttl time.Duration) error
}

// redisRepository 实现 RedisSessionRepository
type redisRepository[T any] struct {
	client *redis.Client
}

// NewRedisClient inti Redis Sentinel connection
func NewRedisClient(masterName string, sentinelAddrs []string, db int) (*redis.Client, error) {
	rdb := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    masterName,    // 哨兵主节点名称
		SentinelAddrs: sentinelAddrs, // 哨兵地址列表
		Password:      "",            // Redis 密码（如有需要）
		DB:            db,            // Redis 数据库编号
	})

	// 测试连接
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis sentinel: %w", err)
	}

	return rdb, nil
}

// NewRedisRepository init Redis repository (Set , Get, Del, GetTTL, ExtendTTL)
func NewRedisRepository[T any](masterName string, sentinelAddrs []string, db int) (RedisRepository[T], error) {
	rdb := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    masterName,    // 哨兵主节点名称
		SentinelAddrs: sentinelAddrs, // 哨兵地址列表
		Password:      "",            // Redis 密码（如有需要）
		DB:            db,            // Redis 数据库编号
	})

	// 测试连接
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis sentinel: %w", err)
	}

	return &redisRepository[T]{client: rdb}, nil
}

func (r *redisRepository[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	// 将 sessionData 序列化为 JSON
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal sessionData: %w", err)
	}

	// 存储到 Redis
	return r.client.Set(ctx, key, data, ttl).Err()
}

func (r *redisRepository[T]) Get(ctx context.Context, key string) (T, error) {
	var zeroValue T // 用于返回空值
	// 从 Redis 获取值
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return zeroValue, fmt.Errorf("redis.Nil")
	} else if err != nil {
		return zeroValue, fmt.Errorf("failed to get session: %w", err)
	}

	// 解析 JSON 数据
	var result T
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		logger.Log.Error("Logout err :", zap.String("err", fmt.Sprintf("failed to unmarshal session data: %v", err)))
		return zeroValue, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	return result, nil
}

func (r *redisRepository[T]) Del(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *redisRepository[T]) ExtendTTL(ctx context.Context, key string, ttl time.Duration) error {
	// 更新 Key 的过期时间
	return r.client.Expire(ctx, key, ttl).Err()
}

func (r *redisRepository[T]) GetTTL(ctx context.Context, key string) (int, error) {
	ttl, err := r.client.TTL(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	} else if err != nil {
		return 0, fmt.Errorf("failed to get TTL for key %s: %w", key, err)
	}

	if ttl < 0 {
		return 0, nil
	}

	return int(ttl.Seconds()), nil
}
