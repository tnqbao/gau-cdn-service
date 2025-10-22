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
	CloudflareR2 struct {
		Endpoint   string
		AccessKey  string
		SecretKey  string
		BucketName string
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

	// Cloudflare R2
	config.CloudflareR2.Endpoint = os.Getenv("CLOUDFLARE_R2_ENDPOINT")
	config.CloudflareR2.AccessKey = os.Getenv("CLOUDFLARE_R2_ACCESS_KEY_ID")
	config.CloudflareR2.SecretKey = os.Getenv("CLOUDFLARE_R2_SECRET_ACCESS_KEY")
	if bucketName := os.Getenv("CLOUDFLARE_R2_BUCKET_NAME"); bucketName != "" {
		config.CloudflareR2.BucketName = bucketName
	} else {
		config.CloudflareR2.BucketName = "default-bucket" // Default bucket name if not set
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
