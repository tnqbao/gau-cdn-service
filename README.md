# CDN Service by Gau

## Introduction | Giới thiệu

**English:**  
This repository provides a CDN (Content Delivery Network) service written in Go, designed to serve images efficiently with caching capabilities. It supports image compression, Redis caching, and integrates with Cloudflare R2 for storage. The service optimizes image delivery with automatic compression and intelligent caching strategies, making it suitable for microservices architectures and can be deployed using Docker or Kubernetes.

**Tiếng Việt:**  
Repo này cung cấp dịch vụ CDN (Content Delivery Network) viết bằng Go, được thiết kế để phục vụ hình ảnh hiệu quả với khả năng cache. Hỗ trợ nén hình ảnh, cache Redis và tích hợp với Cloudflare R2 để lưu trữ. Dịch vụ tối ưu hóa việc phân phối hình ảnh với nén tự động và chiến lược cache thông minh, phù hợp với kiến trúc microservices và có thể triển khai bằng Docker hoặc Kubernetes.

---

## Directory Structure | Cấu trúc thư mục

```
gau-cdn-service/
├── Dockerfile
├── entrypoint.sh
├── go.mod
├── go.sum
├── main.go
├── README.md
├── config/
│   ├── env_config.go
│   └── main.go
├── controller/
│   ├── helper.go
│   ├── images.go
│   └── main.go
├── deploy/
│   └── k8s/
│       ├── production/
│       │   ├── apply.sh
│       │   ├── apply_envsubst.sh
│       │   ├── kustomization.yaml
│       │   ├── unapply.sh
│       │   ├── base/
│       │   └── template/
│       │       ├── configmap.yaml
│       │       ├── deployment.yaml
│       │       ├── hpa.yaml
│       │       ├── ingress.yaml
│       │       ├── secret.yaml
│       │       └── service.yaml
│       └── staging/
│           ├── apply.sh
│           ├── apply_envsubst.sh
│           ├── kustomization.yaml
│           ├── unapply.sh
│           ├── base/
│           └── template/
│               ├── configmap.yaml
│               ├── deployment.yaml
│               ├── hpa.yaml
│               ├── ingress.yaml
│               ├── secret.yaml
│               └── service.yaml
├── infra/
│   ├── cloudflare_r2.go
│   ├── logger.go
│   ├── main.go
│   └── redis.go
├── migrations/
├── provider/
│   ├── logger.go
│   └── main.go
├── repository/
│   ├── main.go
│   └── redis.go
├── routes/
│   └── routes.go
└── utils/
    └── response.go
```

### 📑 Directory Description | Mô tả thư mục

| Path                          | Description                                             | Mô tả                                  |
|-------------------------------|---------------------------------------------------------|----------------------------------------|
| `Dockerfile`, `entrypoint.sh` | Docker image build and startup script                   | File build và khởi động Docker         |
| `go.mod`, `go.sum`            | Go module definitions                                   | Định nghĩa module Go                   |
| `config/`                     | Environment loading and configuration logic             | Logic cấu hình và load môi trường      |
| `controller/`                 | HTTP handlers for image serving operations              | Xử lý HTTP để phục vụ hình ảnh         |
| `deploy/k8s/`                 | Kubernetes manifests and scripts for staging/production | Manifest và script triển khai trên K8s |
| `infra/`                      | Cloud storage (R2), Redis, and logging setup           | Thiết lập cloud storage, Redis và log  |
| `provider/`                   | Service providers and dependency injection              | Provider dịch vụ và dependency injection |
| `repository/`                 | Data access layer for caching operations               | Tầng truy cập dữ liệu cho cache        |
| `routes/`                     | API route definitions                                   | Định nghĩa route API                   |
| `utils/`                      | HTTP response utilities                                 | Tiện ích phản hồi HTTP                 |

---

## Features | Tính năng

### 🚀 Image Delivery | Phân phối hình ảnh

**English:**
- Fast image serving from Cloudflare R2 storage
- Automatic image compression for large files
- Intelligent caching with Redis for frequently accessed images
- Support for multiple image formats (JPEG, PNG, WebP)
- Optimized delivery with cache headers

**Tiếng Việt:**
- Phục vụ hình ảnh nhanh từ Cloudflare R2 storage
- Tự động nén hình ảnh cho file lớn
- Cache thông minh với Redis cho hình ảnh truy cập thường xuyên
- Hỗ trợ nhiều định dạng hình ảnh (JPEG, PNG, WebP)
- Phân phối tối ưu với cache headers

### ⚡ Performance | Hiệu suất

**English:**
- Redis caching layer for ultra-fast image delivery
- Automatic image compression when files exceed cache size limits
- Configurable cache size and compression quality
- Cache hit indicators in response headers

**Tiếng Việt:**
- Tầng cache Redis cho phân phối hình ảnh siêu nhanh
- Tự động nén hình ảnh khi file vượt quá giới hạn cache
- Kích thước cache và chất lượng nén có thể cấu hình
- Chỉ báo cache hit trong response headers

### 🔒 Reliability | Độ tin cậy

**English:**
- Robust error handling for missing or corrupted files
- Fallback mechanisms for cache misses
- Comprehensive logging with OpenTelemetry integration
- Health monitoring and observability

**Tiếng Việt:**
- Xử lý lỗi mạnh mẽ cho file thiếu hoặc bị hỏng
- Cơ chế fallback cho cache miss
- Logging toàn diện với tích hợp OpenTelemetry
- Giám sát sức khỏe và quan sát hệ thống

---

## API Endpoints | Điểm cuối API

### GET /images/{filePath}

**Request:**
```bash
curl -X GET http://localhost:8080/images/folder/image.jpg
```

**Response Headers:**
```
Content-Type: image/jpeg
X-From-Cache: true  # Present when served from Redis cache
Cache-Control: public, max-age=3600
```

**Response:**
- Returns the requested image file
- Automatically compresses large images if needed
- Serves from Redis cache when available
- Falls back to Cloudflare R2 storage

**Parameters:**
- `filePath`: Path to the image file in storage (supports nested paths)

**Status Codes:**
- `200`: Image found and served successfully
- `404`: Image not found or too large to serve

---

## Deployment | Triển khai

### 🐳 Docker

**English:**
1. Build the Docker image:
   ```bash
   docker build -t gau-cdn-service .
   ```
2. Run the container:
   ```bash
   docker run -p 8080:8080 \
     -e CLOUDFLARE_R2_ENDPOINT="https://<account_id>.r2.cloudflarestorage.com" \
     -e CLOUDFLARE_R2_ACCESS_KEY_ID="your_access_key" \
     -e CLOUDFLARE_R2_SECRET_ACCESS_KEY="your_secret_key" \
     -e REDIS_ADDRESS="redis:6379" \
     -e REDIS_PASSWORD="your_password" \
     -e REDIS_DB="cdn" \
     gau-cdn-service
   ```

**Tiếng Việt:**
1. Build image Docker:
   ```bash
   docker build -t gau-cdn-service .
   ```
2. Chạy container:
   ```bash
   docker run -p 8080:8080 \
     -e CLOUDFLARE_R2_ENDPOINT="https://<account_id>.r2.cloudflarestorage.com" \
     -e CLOUDFLARE_R2_ACCESS_KEY_ID="your_access_key" \
     -e CLOUDFLARE_R2_SECRET_ACCESS_KEY="your_secret_key" \
     -e REDIS_ADDRESS="redis:6379" \
     -e REDIS_PASSWORD="your_password" \
     -e REDIS_DB="cdn" \
     gau-cdn-service
   ```

---

### ☸ Kubernetes

**English:**
1. Edit environment variables in `deploy/k8s/staging/template/configmap.yaml` and `secret.yaml`.
2. Apply manifests:
   ```bash
   cd deploy/k8s/staging
   ./apply.sh
   ```
3. To remove:
   ```bash
   ./unapply.sh
   ```

**Tiếng Việt:**
1. Chỉnh sửa biến môi trường trong `deploy/k8s/staging/template/configmap.yaml` và `secret.yaml`.
2. Áp dụng manifest:
   ```bash
   cd deploy/k8s/staging
   ./apply.sh
   ```
3. Để xóa:
   ```bash
   ./unapply.sh
   ```

---

## Configuration | Cấu hình

### Environment Variables | Biến môi trường

| Variable | Description | Example | Required |
|----------|-------------|---------|----------|
| `CLOUDFLARE_R2_ENDPOINT` | Cloudflare R2 endpoint URL | `https://<account_id>.r2.cloudflarestorage.com` | Yes |
| `CLOUDFLARE_R2_ACCESS_KEY_ID` | R2 access key ID | `your_access_key_id` | Yes |
| `CLOUDFLARE_R2_SECRET_ACCESS_KEY` | R2 secret access key | `your_secret_access_key` | Yes |
| `REDIS_ADDRESS` | Redis server address | `redis:6379` | Yes |
| `REDIS_PASSWORD` | Redis authentication password | `your_redis_password` | Yes |
| `REDIS_DB` | Redis database name | `cdn` | Yes |

### Example Environment File | File môi trường mẫu

```shell
#!/bin/sh

export CLOUDFLARE_R2_ENDPOINT="https://<your_account_id>.r2.cloudflarestorage.com"
export CLOUDFLARE_R2_ACCESS_KEY_ID="your_access_key_id"
export CLOUDFLARE_R2_SECRET_ACCESS_KEY="your_secret_access_key"

export REDIS_ADDRESS="redis:6379"
export REDIS_PASSWORD="Qu_bao1604"
export REDIS_DB="cdn"
```

---

## Performance Optimization | Tối ưu hóa hiệu suất

### Caching Strategy | Chiến lược Cache

**English:**
- **First Level**: Redis cache for frequently accessed images
- **Second Level**: Cloudflare R2 storage as the source of truth
- **Compression**: Automatic image compression for large files
- **Cache Key**: Uses compressed prefix for optimized storage

**Tiếng Việt:**
- **Cấp độ 1**: Cache Redis cho hình ảnh truy cập thường xuyên
- **Cấp độ 2**: Cloudflare R2 storage làm nguồn dữ liệu gốc
- **Nén**: Tự động nén hình ảnh cho file lớn
- **Cache Key**: Sử dụng prefix compressed để tối ưu lưu trữ

---

## Contact | Liên hệ

Nếu bạn có bất kỳ câu hỏi hoặc đề xuất nào, vui lòng liên hệ qua:

* Github: [tnqbao](https://github.com/tnqbao)
* LinkedIn: [https://www.linkedin.com/in/tnqb2004/](https://www.linkedin.com/in/tnqb2004/)
