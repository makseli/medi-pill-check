package database

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/makseli/medi-pill-check/internal/config"
)

var RedisClient *redis.Client

func InitRedis(cfg *config.Config) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPass,
		DB:       0,
	})
	return RedisClient.Ping(context.Background()).Err()
}
