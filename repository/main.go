package repository

import (
	"github.com/redis/go-redis/v9"
	"github.com/tnqbao/gau-cdn-service/config"
	"github.com/tnqbao/gau-cdn-service/infra"
)

type Repository struct {
	envConfig *config.EnvConfig
	cacheDb   *redis.Client
}

var repository *Repository

func InitRepository(infra *infra.Infra, config *config.EnvConfig) *Repository {
	repository = &Repository{
		envConfig: config,
		cacheDb:   infra.RedisClient.Client,
	}
	if repository.cacheDb == nil {
		panic("database connection is nil")
	}
	return repository
}

func GetRepository() *Repository {
	if repository == nil {
		panic("repository not initialized")
	}
	return repository
}
