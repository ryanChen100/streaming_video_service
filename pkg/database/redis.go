package database

import (
	"context"
	"encoding/json"
	"fmt"
	"streaming_video_service/pkg/config"
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
func NewRedisClient(masterName, localUse string, sentinelAddrs []string, db int) (*redis.Client, error) {
	var rdb *redis.Client
	if config.IsLocal() {
		rdb = redis.NewClient(&redis.Options{
			Addr:         localUse,
			DB:           db,
			PoolSize:     10, // 连接池大小
			MinIdleConns: 10, // 最小空闲连接
		})
	} else {
		rdb = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    masterName,    // 哨兵主节点名称
			SentinelAddrs: sentinelAddrs, // 哨兵地址列表
			Password:      "",            // Redis 密码（如有需要）
			DB:            db,            // Redis 数据库编号
		})
	}

	// 测试连接
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis sentinel: %w", err)
	}

	return rdb, nil
}

// NewRedisRepository init Redis repository (Set , Get, Del, GetTTL, ExtendTTL)
func NewRedisRepository[T any](masterName, localUse string, sentinelAddrs []string, db int) (RedisRepository[T], error) {
	var rdb *redis.Client
	if config.IsLocal() {
		rdb = redis.NewClient(&redis.Options{
			Addr:         localUse,
			DB:           db,
			PoolSize:     10, // 连接池大小
			MinIdleConns: 10, // 最小空闲连接
			// Addr            string        // Redis 服务器地址，格式："host:port"
			// Network         string        // 连接类型（"tcp" 或 "unix"）
			// Dialer          func(ctx context.Context, network, addr string) (net.Conn, error) // 自定义拨号
			// OnConnect       func(ctx context.Context, cn *Conn) error // 连接建立时的回调

			// Password        string        // 连接 Redis 服务器的密码（默认 "" 表示无密码）
			// DB              int           // Redis 数据库编号（默认 0）

			// MaxRetries      int           // 最大重试次数（默认 3，负数表示不重试）
			// MinRetryBackoff time.Duration // 最小重试间隔（默认 8ms）
			// MaxRetryBackoff time.Duration // 最大重试间隔（默认 512ms）

			// DialTimeout  time.Duration // 连接超时时间（默认 5s）
			// ReadTimeout  time.Duration // 读操作超时时间（默认 3s，-1 代表不超时）
			// WriteTimeout time.Duration // 写操作超时时间（默认 3s，-1 代表不超时）

			// PoolFIFO           bool          // 是否使用 FIFO 连接池（默认 false，即 LIFO 连接池）
			// PoolSize           int           // 连接池最大连接数（默认等于 CPU 核心数 * 10）
			// MinIdleConns       int           // 连接池最小空闲连接数（默认 0）
			// MaxConnAge         time.Duration // 连接的最大存活时间（默认 0，表示不限制）
			// PoolTimeout        time.Duration // 等待连接的超时时间（默认等于 `ReadTimeout + 1s`）
			// IdleTimeout        time.Duration // 空闲连接最大存活时间（默认 5min，-1 代表不关闭空闲连接）
			// IdleCheckFrequency time.Duration // 检查空闲连接的频率（默认 1min）

			// TLSConfig *tls.Config // TLS 连接配置（默认 nil 表示不使用 TLS）
		})
	} else {
		rdb = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    masterName,    // 哨兵主节点名称
			SentinelAddrs: sentinelAddrs, // 哨兵地址列表
			Password:      "",            // Redis 密码（如有需要）
			DB:            db,            // Redis 数据库编号
		})
	}

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
