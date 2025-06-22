package repository

import (
	"github.com/redis/go-redis/v9"
	"github.com/tnqbao/gau-cdn-service/infra"
)

type Repository struct {
	cacheDb *redis.Client
}

var repository *Repository

func InitRepository(infra *infra.Infra) *Repository {
	repository = &Repository{
		cacheDb: infra.RedisClient.Client,
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
