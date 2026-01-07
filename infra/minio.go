package infra

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	appconfig "github.com/tnqbao/gau-cdn-service/config"
)

const (
	SmallFileSizeLimit = 50 * 1024 * 1024 // 50MB - cache in Redis
	StreamBufferSize   = 32 * 1024        // 32KB buffer for streaming
)

type MinioClient struct {
	Client     *minio.Client
	BucketName string
}

type ObjectInfo struct {
	Size        int64
	ContentType string
	ETag        string
}

func NewMinioClient(cfg *appconfig.EnvConfig) (*MinioClient, error) {
	endpoint := cfg.Minio.Endpoint
	if endpoint == "" {
		return nil, fmt.Errorf("MinIO endpoint is not configured")
	}

	accessKey := cfg.Minio.AccessKey
	if accessKey == "" {
		return nil, fmt.Errorf("MinIO access key is not configured")
	}

	secretKey := cfg.Minio.SecretKey
	if secretKey == "" {
		return nil, fmt.Errorf("MinIO secret key is not configured")
	}

	bucketName := cfg.Minio.BucketName
	useSSL := cfg.Minio.UseSSL

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	return &MinioClient{
		Client:     client,
		BucketName: bucketName,
	}, nil
}

// NewMinioClientWithCredentials creates a temporary MinIO client with custom credentials
func NewMinioClientWithCredentials(cfg *appconfig.EnvConfig, accessKey, secretKey string) (*MinioClient, error) {
	endpoint := cfg.Minio.Endpoint
	if endpoint == "" {
		return nil, fmt.Errorf("MinIO endpoint is not configured")
	}

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("access key and secret key cannot be empty")
	}

	bucketName := cfg.Minio.BucketName
	useSSL := cfg.Minio.UseSSL

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client with custom credentials: %w", err)
	}

	return &MinioClient{
		Client:     client,
		BucketName: bucketName,
	}, nil
}

// HeadObject gets object metadata without downloading content (for cache decision)
func (m *MinioClient) HeadObject(ctx context.Context, bucket, key string) (*ObjectInfo, error) {
	stat, err := m.Client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}

	return &ObjectInfo{
		Size:        stat.Size,
		ContentType: stat.ContentType,
		ETag:        stat.ETag,
	}, nil
}

// GetObjectStream returns a reader for streaming large files directly to client
func (m *MinioClient) GetObjectStream(ctx context.Context, bucket, key string, opts minio.GetObjectOptions) (io.ReadCloser, *ObjectInfo, error) {
	object, err := m.Client.GetObject(ctx, bucket, key, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object: %w", err)
	}

	stat, err := object.Stat()
	if err != nil {
		object.Close()
		return nil, nil, fmt.Errorf("failed to stat object: %w", err)
	}

	info := &ObjectInfo{
		Size:        stat.Size,
		ContentType: stat.ContentType,
		ETag:        stat.ETag,
	}

	return object, info, nil
}

// GetObjectWithRange supports range requests for video streaming and resume download
func (m *MinioClient) GetObjectWithRange(ctx context.Context, bucket, key string, start, end int64) (io.ReadCloser, *ObjectInfo, error) {
	opts := minio.GetObjectOptions{}
	if err := opts.SetRange(start, end); err != nil {
		return nil, nil, fmt.Errorf("failed to set range: %w", err)
	}

	return m.GetObjectStream(ctx, bucket, key, opts)
}

// GetSmallObject loads small files into memory for caching in Redis
func (m *MinioClient) GetSmallObject(ctx context.Context, bucket, key string, maxSize int64) ([]byte, string, error) {
	// First check size with HEAD request
	info, err := m.HeadObject(ctx, bucket, key)
	if err != nil {
		return nil, "", err
	}

	if info.Size > maxSize {
		return nil, "", fmt.Errorf("object too large for cache (%d bytes > %d limit)", info.Size, maxSize)
	}

	object, err := m.Client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get object: %w", err)
	}
	defer object.Close()

	// Use buffer with limit to prevent memory issues
	buf := make([]byte, info.Size)
	n, err := io.ReadFull(object, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, "", fmt.Errorf("failed to read object: %w", err)
	}

	return buf[:n], info.ContentType, nil
}

// StreamToWriter streams object directly to writer with buffer (for large files)
func (m *MinioClient) StreamToWriter(ctx context.Context, bucket, key string, w io.Writer, opts minio.GetObjectOptions) (int64, string, error) {
	if bucket == "" {
		return 0, "", fmt.Errorf("bucket cannot be empty")
	}
	if key == "" {
		return 0, "", fmt.Errorf("key cannot be empty")
	}

	reader, info, err := m.GetObjectStream(ctx, bucket, key, opts)
	if err != nil {
		return 0, "", err
	}
	defer reader.Close()

	// Use fixed buffer for efficient streaming
	buf := make([]byte, StreamBufferSize)
	written, err := io.CopyBuffer(w, reader, buf)
	if err != nil {
		return written, info.ContentType, fmt.Errorf("failed to stream object: %w", err)
	}

	return written, info.ContentType, nil
}

// IsAccessDeniedError checks if the error is an access denied error from MinIO
func IsAccessDeniedError(err error) bool {
	if err == nil {
		return false
	}
	return minio.ToErrorResponse(err).Code == "AccessDenied" ||
		minio.ToErrorResponse(err).Code == "InvalidAccessKeyId" ||
		minio.ToErrorResponse(err).Code == "SignatureDoesNotMatch"
}
