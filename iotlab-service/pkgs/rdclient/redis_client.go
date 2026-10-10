// Package rdclient opens the go-redis connection and exposes it behind the RedisClient interface.
package rdclient

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient exposes the shared go-redis client and its shutdown hook.
type RedisClient interface {
	Client() redis.UniversalClient
	Close() error
}

type redisClient struct {
	_client redis.UniversalClient
}

func (r *redisClient) Client() redis.UniversalClient {
	return r._client
}

func (r *redisClient) Close() error {
	return r._client.Close()
}

// Config configures the Redis connection. Mode is "standalone" (default), "cluster" or "sentinel".
type Config struct {
	Mode       string
	Username   string
	Password   string
	MasterName string
	PoolSize   int
	Address    []string
}

// NewRedisClient connects to Redis in the mode named by cfg.
func NewRedisClient(cfg Config) (RedisClient, error) {
	if len(cfg.Address) == 0 {
		slog.Error("redis address list is empty")
		return nil, fmt.Errorf("redis address list is empty")
	}

	connOpts := redis.UniversalOptions{
		Addrs:        cfg.Address,
		Username:     cfg.Username,
		Password:     cfg.Password,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		MaxRetries:   3,
	}

	switch cfg.Mode {
	case "cluster": // Cluster mode relies on multiple node addresses
		if len(connOpts.Addrs) == 0 {
			slog.Error("redis cluster mode requires at least one node address")
			return nil, fmt.Errorf("redis cluster mode requires at least one node address")
		}
	case "sentinel": // Sentinel mode requires master name and sentinel addresses
		if cfg.MasterName == "" {
			slog.Error("redis sentinel mode requires a master name")
			return nil, fmt.Errorf("redis sentinel mode requires a master name")
		}
		connOpts.MasterName = cfg.MasterName
	default:
		if len(connOpts.Addrs) == 0 { // Treat any other value as standalone
			slog.Error("redis standalone mode requires a node address")
			return nil, fmt.Errorf("redis standalone mode requires a node address")
		}
		// Ensure only the primary address is used for standalone setups
		connOpts.Addrs = []string{connOpts.Addrs[0]}
	}

	client := redis.NewUniversalClient(&connOpts)

	// Test the connection with a ping to ensure it's working
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		slog.Error("redis client ping failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("redis client ping failed: %w", err)
	}

	slog.Info("redis connection created",
		slog.String("mode", cfg.Mode),
		slog.Any("addresses", cfg.Address),
		slog.String("master_name", cfg.MasterName),
	)

	return &redisClient{
		_client: client,
	}, nil
}
