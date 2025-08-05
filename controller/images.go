package controller

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-cdn-service/utils"
	"net/http"
	"strings"
)

func (ctrl *Controller) GetImage(c *gin.Context) {
	filepath := c.Param("filePath")
	if filepath == "" || filepath == "/" {
		utils.JSON400(c, "invalid file path")
		return
	}

	key := filepath[1:]
	compressedKey := "compressed:" + key
	ctx := context.Background()

	// Try to get compressed image from cache first
	data, contentType, err := ctrl.Repository.GetImage(ctx, compressedKey)
	if err == nil && len(data) > 0 {
		c.Header("X-From-Cache", "true")
		c.Data(http.StatusOK, contentType, data)
		return
	}

	// Fetch original image from Cloudflare R2
	data, contentType, err = ctrl.Infra.CloudflareR2Client.GetObjectWithLimit(ctx, key, ctrl.Config.EnvConfig.Limit.CacheSize)
	if err != nil {
		fmt.Printf("fetch failed: %v\n", err)
		utils.JSON404(c, "file not found or too large")
		return
	}

	toCache := data
	isCompressed := false

	// Compress large non-JPEG images
	if len(data) > 100*1024 && !strings.Contains(contentType, "jpeg") && !strings.Contains(contentType, "jpg") {
		if compressed, err := compressToJPEGUnder100KB(data, 1024); err == nil {
			toCache = compressed
			contentType = "image/jpeg"
			isCompressed = true
		} else {
			fmt.Printf("compress failed: %v\n", err)
		}
	}

	// Cache the processed image
	if err := ctrl.Repository.SetImage(ctx, compressedKey, toCache, contentType); err != nil {
		fmt.Printf("cache failed: %v\n", err)
	}

	// Set response headers
	if isCompressed {
		c.Header("X-Compressed", "true")
	}
	c.Header("X-From-Cache", "false")

	// Set cache control headers
	if ctrl.Config.EnvConfig.Limit.CacheSize > 0 {
		c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", ctrl.Config.EnvConfig.Limit.CacheTime))
	} else {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
	}

	c.Data(http.StatusOK, contentType, toCache)
}
