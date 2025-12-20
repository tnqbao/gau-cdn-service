package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/tnqbao/gau-cdn-service/infra"
	"github.com/tnqbao/gau-cdn-service/utils"
)

const (
	// Timeouts
	OriginReadTimeout = 30 * time.Second
)

func (ctrl *Controller) GetFile(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), OriginReadTimeout)
	defer cancel()

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

	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[GetFile] Request: bucket=%s, key=%s", bucket, key)

	// Check for Range header (video streaming, resume download)
	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		ctrl.handleRangeRequest(c, ctx, bucket, key, rangeHeader)
		return
	}

	// Try to get from cache first (only for small files)
	cacheKey := fmt.Sprintf("cdn:%s:%s", bucket, key)
	if data, contentType, err := ctrl.Repository.GetImage(ctx, cacheKey); err == nil && len(data) > 0 {
		ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[GetFile] Cache hit for key: %s", cacheKey)
		ctrl.setCacheHeaders(c, true)
		c.Data(http.StatusOK, contentType, data)
		return
	}

	// HEAD request to get file metadata for cache decision
	objInfo, err := ctrl.Infra.MinioClient.HeadObject(ctx, bucket, key)
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[GetFile] HEAD request failed for bucket=%s, key=%s", bucket, key)
		utils.JSON404(c, "file not found")
		return
	}

	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[GetFile] File info: size=%d, type=%s", objInfo.Size, objInfo.ContentType)

	// Decide strategy based on file size
	if objInfo.Size <= infra.SmallFileSizeLimit {
		// Small file: load to memory and cache in Redis
		ctrl.handleSmallFile(c, ctx, bucket, key, cacheKey, objInfo)
	} else {
		// Large file: stream directly without caching in Redis
		ctrl.handleLargeFile(c, ctx, bucket, key, objInfo)
	}
}

// handleSmallFile loads small files into memory and caches in Redis
func (ctrl *Controller) handleSmallFile(c *gin.Context, ctx context.Context, bucket, key, cacheKey string, objInfo *infra.ObjectInfo) {
	data, contentType, err := ctrl.Infra.MinioClient.GetSmallObject(ctx, bucket, key, infra.SmallFileSizeLimit)
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[GetFile] Failed to get small object: bucket=%s, key=%s", bucket, key)
		utils.JSON500(c, "failed to fetch file")
		return
	}

	// Cache in Redis for future requests
	if err := ctrl.Repository.SetImage(ctx, cacheKey, data, contentType); err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[GetFile] Failed to cache file: %s", cacheKey)
		// Continue serving even if cache fails
	}

	ctrl.setCacheHeaders(c, false)
	c.Header("Content-Length", strconv.FormatInt(int64(len(data)), 10))
	c.Header("ETag", objInfo.ETag)
	c.Data(http.StatusOK, contentType, data)

	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[GetFile] Served small file: bucket=%s, key=%s, size=%d", bucket, key, len(data))
}

// handleLargeFile streams large files directly to client using io.CopyBuffer
func (ctrl *Controller) handleLargeFile(c *gin.Context, ctx context.Context, bucket, key string, objInfo *infra.ObjectInfo) {
	// Set headers before streaming
	c.Header("Content-Type", objInfo.ContentType)
	c.Header("Content-Length", strconv.FormatInt(objInfo.Size, 10))
	c.Header("ETag", objInfo.ETag)
	c.Header("Accept-Ranges", "bytes")
	ctrl.setCacheHeaders(c, false)
	c.Status(http.StatusOK)

	// Stream directly to response writer
	written, _, err := ctrl.Infra.MinioClient.StreamToWriter(ctx, bucket, key, c.Writer, minio.GetObjectOptions{})
	if err != nil {
		ctrl.Provider.LoggerProvider.ErrorWithContextf(ctx, err, "[GetFile] Stream failed: bucket=%s, key=%s, written=%d", bucket, key, written)
		// Can't send error response as headers already sent
		return
	}

	ctrl.Provider.LoggerProvider.InfoWithContextf(ctx, "[GetFile] Streamed large file: bucket=%s, key=%s, size=%d", bucket, key, written)
}

// handleRangeRequest handles HTTP Range requests for video streaming and resume download
func (ctrl *Controller) handleRangeRequest(c *gin.Context, ctx context.Context, bucket, key, rangeHeader string) {
	// Get file metadata first
	objInfo, err := ctrl.Infra.MinioClient.HeadObject(ctx, bucket, key)
	if err != nil {
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
	ctrl.setCacheHeaders(c, false)
	c.Status(http.StatusPartialContent)

	// Stream range to client
	reader, _, err := ctrl.Infra.MinioClient.GetObjectWithRange(ctx, bucket, key, start, end)
	if err != nil {
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
