package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/tnqbao/gau-cdn-service/infra"
	"github.com/tnqbao/gau-cdn-service/utils"
)

const (
// Timeouts
// OriginReadTimeout = 30 * time.Second
)

func (ctrl *Controller) GetFile(c *gin.Context) {
	ctx := c.Request.Context()

	bucket := c.Param("bucket")
	path := c.Param("path")

	// Validate parameters
	if bucket == "" {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[GetFile] Missing bucket parameter")
		utils.JSON400(c, "missing bucket parameter")
		return
	}

	// Clean path - remove leading slash
	key := strings.TrimPrefix(path, "/")
	if key == "" {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[GetFile] Invalid file path")
		utils.JSON400(c, "invalid file path")
		return
	}

	// Get optional access_key and secret_key from query params
	accessKey := c.Query("access_key")
	secretKey := c.Query("secret_key")

	// Determine which MinIO client to use
	var minioClient *infra.MinioClient
	var err error

	if accessKey != "" && secretKey != "" {
		// Create temporary client with custom credentials
		minioClient, err = infra.NewMinioClientWithCredentials(ctrl.Config.EnvConfig, accessKey, secretKey)
		if err != nil {
			ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[GetFile] Failed to create MinIO client with custom credentials")
			utils.JSON500(c, "failed to initialize storage client")
			return
		}
		ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[GetFile] Using custom credentials for bucket=%s, key=%s", bucket, key)
	} else {
		// Use default client from controller
		minioClient = ctrl.Infra.MinioClient
		ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[GetFile] Using default credentials for bucket=%s, key=%s", bucket, key)
	}

	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[GetFile] Request: bucket=%s, key=%s", bucket, key)

	// Check if download mode is requested
	downloadMode := c.Query("mode") == "download"
	if downloadMode {
		ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[GetFile] Download mode requested for bucket=%s, key=%s", bucket, key)
	}

	// Check for Range header (video streaming, resume download)
	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		ctrl.handleRangeRequest(c, ctx, minioClient, bucket, key, rangeHeader, downloadMode)
		return
	}

	// Get file metadata to determine size
	objInfo, err := minioClient.HeadObject(ctx, bucket, key)
	if err != nil {
		// Check if it's an Access Denied error
		if infra.IsAccessDeniedError(err) {
			ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[GetFile] Access denied for bucket=%s, key=%s", bucket, key)
			utils.JSON403(c, "Access Denied")
			return
		}
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[GetFile] HEAD request failed for bucket=%s, key=%s", bucket, key)
		utils.JSON404(c, "file not found")
		return
	}

	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[GetFile] File info: size=%d, type=%s", objInfo.Size, objInfo.ContentType)

	// For small files < 50MB, try cache first
	if objInfo.Size <= infra.SmallFileSizeLimit {
		cacheKey := fmt.Sprintf("cdn:%s:%s", bucket, key)
		if data, contentType, err := ctrl.Repository.GetImage(ctx, cacheKey); err == nil && len(data) > 0 {
			ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[GetFile] Cache hit for key: %s", cacheKey)
			ctrl.setCacheHeaders(c, true)

			// Set download headers if requested
			if downloadMode {
				ctrl.setDownloadHeaders(c, key)
			}

			c.Header("Content-Length", strconv.FormatInt(int64(len(data)), 10))
			c.Header("ETag", objInfo.ETag)
			c.Data(http.StatusOK, contentType, data)
			return
		}

		// Validate size before handling small file
		if objInfo.Size <= 0 {
			ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, nil, "[GetFile] Invalid file size detected before handling: bucket=%s, key=%s, size=%d", bucket, key, objInfo.Size)
			utils.JSON404(c, "file not found or has invalid size")
			return
		}

		// Cache miss, fetch and cache small file
		ctrl.handleSmallFileWithCache(c, ctx, minioClient, bucket, key, cacheKey, objInfo, downloadMode)
	} else {
		// Large file: stream directly without caching
		ctrl.handleLargeFileStream(c, ctx, minioClient, bucket, key, objInfo, downloadMode)
	}
}

// handleSmallFileWithCache streams small files and caches them in Redis
func (ctrl *Controller) handleSmallFileWithCache(c *gin.Context, ctx context.Context, minioClient *infra.MinioClient, bucket, key, cacheKey string, objInfo *infra.ObjectInfo, downloadMode bool) {
	// Validate size before allocating buffer
	if objInfo.Size <= 0 {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, nil, "[GetFile] Invalid file size: bucket=%s, key=%s, size=%d", bucket, key, objInfo.Size)
		utils.JSON500(c, "invalid file size")
		return
	}

	if objInfo.Size > infra.SmallFileSizeLimit {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, nil, "[GetFile] File too large for small file handler: bucket=%s, key=%s, size=%d", bucket, key, objInfo.Size)
		utils.JSON500(c, "file too large")
		return
	}

	// Get object stream from MinIO
	reader, _, err := minioClient.GetObjectStream(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		// Check if it's an Access Denied error
		if infra.IsAccessDeniedError(err) {
			ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[GetFile] Access denied for bucket=%s, key=%s", bucket, key)
			utils.JSON403(c, "Access Denied")
			return
		}
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[GetFile] Failed to get small object stream: bucket=%s, key=%s", bucket, key)
		utils.JSON500(c, "failed to fetch file")
		return
	}
	defer reader.Close()

	// Read into buffer for caching (small files only)
	data := make([]byte, objInfo.Size)
	n, err := io.ReadFull(reader, data)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[GetFile] Failed to read small object: bucket=%s, key=%s", bucket, key)
		utils.JSON500(c, "failed to read file")
		return
	}
	data = data[:n]

	// Cache in Redis for future requests (async, don't block response)
	go func() {
		if err := ctrl.Repository.SetImage(context.Background(), cacheKey, data, objInfo.ContentType); err != nil {
			ctrl.Provider.LoggerProvider.ErrorWithContextf(context.Background(), err, "[GetFile] Failed to cache file: %s", cacheKey)
		}
	}()

	// Set response headers and send data
	ctrl.setCacheHeaders(c, false)

	// Set download headers if requested
	if downloadMode {
		ctrl.setDownloadHeaders(c, key)
	}

	c.Header("Content-Length", strconv.FormatInt(int64(len(data)), 10))
	c.Header("ETag", objInfo.ETag)
	c.Data(http.StatusOK, objInfo.ContentType, data)

	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[GetFile] Served small file: bucket=%s, key=%s, size=%d", bucket, key, len(data))
}

// handleLargeFileStream streams large files directly from MinIO to client without loading into memory
func (ctrl *Controller) handleLargeFileStream(c *gin.Context, ctx context.Context, minioClient *infra.MinioClient, bucket, key string, objInfo *infra.ObjectInfo, downloadMode bool) {
	// Get object stream from MinIO
	reader, _, err := minioClient.GetObjectStream(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		// Check if it's an Access Denied error
		if infra.IsAccessDeniedError(err) {
			ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[GetFile] Access denied for bucket=%s, key=%s", bucket, key)
			utils.JSON403(c, "Access Denied")
			return
		}
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[GetFile] Failed to get object stream: bucket=%s, key=%s", bucket, key)
		utils.JSON500(c, "failed to fetch file")
		return
	}
	defer reader.Close()

	// Set headers before streaming
	c.Header("Content-Type", objInfo.ContentType)
	c.Header("Content-Length", strconv.FormatInt(objInfo.Size, 10))
	c.Header("ETag", objInfo.ETag)
	c.Header("Accept-Ranges", "bytes")

	// Set download headers if requested
	if downloadMode {
		ctrl.setDownloadHeaders(c, key)
	}

	ctrl.setCacheHeaders(c, false)
	c.Status(http.StatusOK)

	// Stream directly to response writer with buffer
	buf := make([]byte, infra.StreamBufferSize)
	written, err := io.CopyBuffer(c.Writer, reader, buf)
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[GetFile] Stream failed: bucket=%s, key=%s, written=%d", bucket, key, written)
		// Can't send error response as headers already sent
		return
	}

	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[GetFile] Streamed large file: bucket=%s, key=%s, size=%d", bucket, key, written)
}

// handleRangeRequest handles HTTP Range requests for video streaming and resume download
func (ctrl *Controller) handleRangeRequest(c *gin.Context, ctx context.Context, minioClient *infra.MinioClient, bucket, key, rangeHeader string, downloadMode bool) {
	// Get file metadata first
	objInfo, err := minioClient.HeadObject(ctx, bucket, key)
	if err != nil {
		// Check if it's an Access Denied error
		if infra.IsAccessDeniedError(err) {
			ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[GetFile] Access denied for bucket=%s, key=%s", bucket, key)
			utils.JSON403(c, "Access Denied")
			return
		}
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[GetFile] HEAD request failed for range request: bucket=%s, key=%s", bucket, key)
		utils.JSON404(c, "file not found")
		return
	}

	// Parse Range header: "bytes=start-end"
	start, end, err := parseRangeHeader(rangeHeader, objInfo.Size)
	if err != nil {
		ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[GetFile] Invalid range header: %s, error: %v", rangeHeader, err)
		c.Header("Content-Range", fmt.Sprintf("bytes */%d", objInfo.Size))
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	contentLength := end - start + 1

	// Set response headers for partial content
	c.Header("Content-Type", objInfo.ContentType)
	c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, objInfo.Size))
	c.Header("Accept-Ranges", "bytes")
	c.Header("ETag", objInfo.ETag)

	// Set download headers if requested
	if downloadMode {
		ctrl.setDownloadHeaders(c, key)
	}

	ctrl.setCacheHeaders(c, false)
	c.Status(http.StatusPartialContent)

	// Stream range to client
	reader, _, err := ctrl.Infra.MinioClient.GetObjectWithRange(ctx, bucket, key, start, end)
	if err != nil {
		// Check if it's an Access Denied error
		if infra.IsAccessDeniedError(err) {
			ctrl.Provider.LoggerProvider.WarningWithContextf(ctx, "[GetFile] Access denied for bucket=%s, key=%s", bucket, key)
			utils.JSON403(c, "Access Denied")
			return
		}
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[GetFile] Range request failed: bucket=%s, key=%s, range=%d-%d", bucket, key, start, end)
		return
	}
	defer reader.Close()

	buf := make([]byte, infra.StreamBufferSize)
	written, err := copyBufferWithLimit(c.Writer, reader, buf, contentLength)
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[GetFile] Range stream failed: bucket=%s, key=%s, written=%d", bucket, key, written)
		return
	}

	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[GetFile] Served range request: bucket=%s, key=%s, range=%d-%d, written=%d", bucket, key, start, end, written)
}

// setCacheHeaders sets appropriate cache control headers
func (ctrl *Controller) setCacheHeaders(c *gin.Context, fromCache bool) {
	if fromCache {
		c.Header("X-From-Cache", "true")
	} else {
		c.Header("X-From-Cache", "false")
	}

	cacheTime := ctrl.Config.EnvConfig.Limit.CacheTime
	if cacheTime > 0 {
		c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", cacheTime))
	} else {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
	}
}

// setDownloadHeaders sets Content-Disposition header to force file download
func (ctrl *Controller) setDownloadHeaders(c *gin.Context, key string) {
	// Extract filename from key path
	filename := filepath.Base(key)

	// Set Content-Disposition header with filename
	// Using attachment to force download, and properly quote the filename
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
}

// parseRangeHeader parses HTTP Range header and returns start, end positions
func parseRangeHeader(rangeHeader string, fileSize int64) (int64, int64, error) {
	// Format: bytes=start-end or bytes=start- or bytes=-suffix
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return 0, 0, fmt.Errorf("invalid range format")
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(rangeSpec, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range format")
	}

	var start, end int64
	var err error

	if parts[0] == "" {
		// Suffix range: bytes=-500 (last 500 bytes)
		suffix, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid suffix range")
		}
		start = fileSize - suffix
		if start < 0 {
			start = 0
		}
		end = fileSize - 1
	} else {
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid start position")
		}

		if parts[1] == "" {
			// Open-ended range: bytes=500-
			end = fileSize - 1
		} else {
			end, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("invalid end position")
			}
		}
	}

	// Validate range
	if start < 0 || start >= fileSize || end < start || end >= fileSize {
		return 0, 0, fmt.Errorf("range out of bounds")
	}

	return start, end, nil
}

// copyBufferWithLimit copies data with a buffer up to a limit
func copyBufferWithLimit(dst gin.ResponseWriter, src interface{ Read([]byte) (int, error) }, buf []byte, limit int64) (int64, error) {
	var written int64
	for written < limit {
		toRead := int64(len(buf))
		if remaining := limit - written; remaining < toRead {
			toRead = remaining
		}

		n, err := src.Read(buf[:toRead])
		if n > 0 {
			nw, errw := dst.Write(buf[:n])
			if nw > 0 {
				written += int64(nw)
			}
			if errw != nil {
				return written, errw
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return written, err
		}
	}
	return written, nil
}
