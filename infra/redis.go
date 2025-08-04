package infra

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/tnqbao/gau-cdn-service/config"
	"log"
)

type RedisClient struct {
	Client *redis.Client
}

func InitRedisClient(cfg *config.EnvConfig) *RedisClient {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.RedisHost + ":" + cfg.Redis.RedisPort,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.Database,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	log.Println("Connected to Redis:", cfg.Redis.RedisPort+" on "+cfg.Redis.RedisHost)

	return &RedisClient{Client: client}
}
