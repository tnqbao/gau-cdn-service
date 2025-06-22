package config

import (
	"os"
	"strconv"
)

type EnvConfig struct {
	//JWT struct {
	//	SecretKey string
	//	Algorithm string
	//	Expire    int
	//}
	Redis struct {
		Address  string
		Password string
		Database int
	}
	CloudflareR2 struct {
		Endpoint   string
		AccessKey  string
		SecretKey  string
		BucketName string
	}
}

func LoadEnvConfig() *EnvConfig {
	var config EnvConfig

	//// JWT
	//config.JWT.SecretKey = os.Getenv("JWT_SECRET_KEY")
	//config.JWT.Algorithm = os.Getenv("JWT_ALGORITHM")
	//
	//if val := os.Getenv("JWT_EXPIRE"); val != "" {
	//	fmt.Sscanf(val, "%d", &config.JWT.Expire)
	//} else {
	//	config.JWT.Expire = 3600 * 24 * 7
	//}

	// Redis
	config.Redis.Address = os.Getenv("REDIS_ADDRESS")
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

	return &config
}
