package infra

import (
	"log"

	"github.com/tnqbao/gau-cdn-service/config"
)

type Infra struct {
	CloudflareR2Client *CloudflareR2Client
	RedisClient        *RedisClient
}

func InitInfra(cfg *config.Config) *Infra {
	r2, err := NewCloudflareR2Client(cfg.EnvConfig)
	redisClient := InitRedisClient(cfg.EnvConfig)
	if err != nil {
		log.Fatalf("Failed to initialize Cloudflare R2 client: %v", err)
	}

	return &Infra{
		CloudflareR2Client: r2,
		RedisClient:        redisClient,
	}
}
