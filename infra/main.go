package infra

import (
	"log"

	"github.com/tnqbao/gau-cdn-service/config"
)

type Infra struct {
	MinioClient *MinioClient
	RedisClient *RedisClient
	Logger      *LoggerClient
}

func InitInfra(cfg *config.Config) *Infra {
	minioClient, err := NewMinioClient(cfg.EnvConfig)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	redisClient := InitRedisClient(cfg.EnvConfig)

	loggerClient := InitLoggerClient(cfg.EnvConfig)
	if loggerClient == nil {
		panic("Failed to create Logger client")
	}
	return &Infra{
		MinioClient: minioClient,
		RedisClient: redisClient,
		Logger:      loggerClient,
	}
}
