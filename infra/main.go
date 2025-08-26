package infra

import (
	"log"

	"github.com/tnqbao/gau-cdn-service/config"
)

type Infra struct {
	CloudflareR2Client *CloudflareR2Client
	RedisClient        *RedisClient
	Logger             *LoggerClient
}

func InitInfra(cfg *config.Config) *Infra {
	r2, err := NewCloudflareR2Client(cfg.EnvConfig)
	if err != nil {
		log.Fatalf("Failed to initialize Cloudflare R2 client: %v", err)
	}

	redisClient := InitRedisClient(cfg.EnvConfig)

	loggerClient := InitLoggerClient(cfg.EnvConfig)
	if loggerClient == nil {
		panic("Failed to create Logger client")
	}
	return &Infra{
		CloudflareR2Client: r2,
		RedisClient:        redisClient,
		Logger:             loggerClient,
	}
}
