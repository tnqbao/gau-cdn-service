package repository

import (
	"context"
	"time"
)

func (r *Repository) GetImage(ctx context.Context, key string) ([]byte, string, error) {
	data, err := r.cacheDb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, "", err
	}
	ct, err := r.cacheDb.Get(ctx, key+":content-type").Result()
	if err != nil {
		ct = "application/octet-stream"
	}
	return data, ct, nil
}

func (r *Repository) SetImage(ctx context.Context, key string, data []byte, contentType string) error {
	pipe := r.cacheDb.TxPipeline()
	pipe.Set(ctx, key, data, time.Hour)
	pipe.Set(ctx, key+":content-type", contentType, time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *Repository) Set(key string, value string) error {
	return r.cacheDb.Set(context.Background(), key, value, 0).Err()
}

func (r *Repository) Get(key string) (string, error) {
	return r.cacheDb.Get(context.Background(), key).Result()
}

func (r *Repository) Delete(key string) error {
	return r.cacheDb.Del(context.Background(), key).Err()
}

func (r *Repository) GetBit(ctx context.Context, key string, offset int64) (int64, error) {
	return r.cacheDb.GetBit(ctx, key, offset).Result()
}
