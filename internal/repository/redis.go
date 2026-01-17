package repository

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// Redis wraps the Redis client
type Redis struct {
	Client *redis.Client
}

// NewRedisClient creates a new Redis connection
func NewRedisClient(ctx context.Context, uri string) (*Redis, error) {
	opt, err := redis.ParseURL(uri)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Redis{
		Client: client,
	}, nil
}

// Ping checks the Redis connection
func (r *Redis) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (r *Redis) Close() error {
	return r.Client.Close()
}