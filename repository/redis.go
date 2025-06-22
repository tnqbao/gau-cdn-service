package repository

import "context"

func (r *Repository) Set(key string, value string) error {
	return r.cacheDb.Set(context.Background(), key, value, 0).Err()
}

func (r *Repository) Get(key string) (string, error) {
	return r.cacheDb.Get(context.Background(), key).Result()
}

func (r *Repository) Delete(key string) error {
	return r.cacheDb.Del(context.Background(), key).Err()
}
