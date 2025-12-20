package config

import (
	"os"
	"strconv"
	"strings"
)

type EnvConfig struct {
	Redis struct {
		RedisHost string
		RedisPort string
		Password  string
		Database  int
	}

	Minio struct {
		Endpoint   string
		AccessKey  string
		SecretKey  string
		BucketName string
		UseSSL     bool
	}

	Limit struct {
		CacheTime int64
		CacheSize int64
	}

	Grafana struct {
		OTLPEndpoint string
		ServiceName  string
	}

	Environment struct {
		Mode  string
		Group string
	}
}

func LoadEnvConfig() *EnvConfig {
	var config EnvConfig

	// Redis
	config.Redis.RedisHost = os.Getenv("REDIS_HOST")
	config.Redis.RedisPort = os.Getenv("REDIS_PORT")
	config.Redis.Password = os.Getenv("REDIS_PASSWORD")
	config.Redis.Database, _ = strconv.Atoi(os.Getenv("REDIS_DB"))
	if config.Redis.Database == 0 {
		config.Redis.Database = 0 // Default to 0 if not set
	}

	// MinIO
	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	if minioEndpoint == "" {
		minioEndpoint = "localhost:9000"
	}
	// Strip protocol from endpoint - MinIO client doesn't accept full URLs
	if strings.HasPrefix(minioEndpoint, "https://") {
		config.Minio.Endpoint = strings.TrimPrefix(minioEndpoint, "https://")
		config.Minio.UseSSL = true
	} else if strings.HasPrefix(minioEndpoint, "http://") {
		config.Minio.Endpoint = strings.TrimPrefix(minioEndpoint, "http://")
		config.Minio.UseSSL = false
	} else {
		config.Minio.Endpoint = minioEndpoint
		config.Minio.UseSSL = os.Getenv("MINIO_USE_SSL") == "true"
	}
	config.Minio.AccessKey = os.Getenv("MINIO_ACCESS_KEY_ID")
	config.Minio.SecretKey = os.Getenv("MINIO_SECRET_ACCESS_KEY")
	config.Minio.BucketName = os.Getenv("MINIO_BUCKET_NAME")
	if config.Minio.BucketName == "" {
		config.Minio.BucketName = "cdn-files"
	}

	// Limit
	cacheTime, err := strconv.ParseInt(os.Getenv("CACHE_TIME"), 10, 64)
	if err != nil {
		cacheTime = 3600 // Default to 1 hour if not set or invalid
	}

	config.Limit.CacheTime = cacheTime
	cacheSize, err := strconv.ParseInt(os.Getenv("CACHE_SIZE"), 10, 64)
	if err != nil {
		cacheSize = 10 * 1024 * 1024 // 10 MB
	}
	config.Limit.CacheSize = cacheSize

	// Grafana/OpenTelemetry
	grafanaEndpoint := os.Getenv("GRAFANA_OTLP_ENDPOINT")
	if grafanaEndpoint == "" {
		grafanaEndpoint = "https://grafana.gauas.online"
	}
	// Remove protocol for OpenTelemetry client to avoid duplicate protocols
	if strings.HasPrefix(grafanaEndpoint, "https://") {
		config.Grafana.OTLPEndpoint = strings.TrimPrefix(grafanaEndpoint, "https://")
	} else if strings.HasPrefix(grafanaEndpoint, "http://") {
		config.Grafana.OTLPEndpoint = strings.TrimPrefix(grafanaEndpoint, "http://")
	} else {
		config.Grafana.OTLPEndpoint = grafanaEndpoint
	}
	config.Grafana.ServiceName = os.Getenv("SERVICE_NAME")
	if config.Grafana.ServiceName == "" {
		config.Grafana.ServiceName = "gau-account-service"
	}

	config.Environment.Mode = os.Getenv("DEPLOY_ENV")
	if config.Environment.Mode == "" {
		config.Environment.Mode = "development"
	}

	config.Environment.Group = os.Getenv("GROUP_NAME")
	if config.Environment.Group == "" {
		config.Environment.Group = "local"
	}

	return &config
}
