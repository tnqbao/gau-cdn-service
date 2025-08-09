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

	data, contentType, err := ctrl.Repository.GetImage(ctx, compressedKey)
	if err == nil && len(data) > 0 {
		c.Header("X-From-Cache", "true")
		c.Data(http.StatusOK, contentType, data)
		return
	}

	data, contentType, err = ctrl.Infra.CloudflareR2Client.GetObjectWithLimit(ctx, key, ctrl.Config.EnvConfig.Limit.CacheSize)
	if err != nil {
		fmt.Printf("fetch failed: %v\n", err)
		utils.JSON404(c, "file not found or too large")
		return
	}

	toCache := data
	isCompressed := false

	if int64(len(data)) > ctrl.Config.EnvConfig.Limit.CacheSize && shouldCompressImage(contentType) {
		if compressed, err := compressImageInOriginalFormat(data, contentType, ctrl.Config.EnvConfig.Limit.CacheSize, 1024); err == nil {
			toCache = compressed
			isCompressed = true
			if strings.Contains(strings.ToLower(contentType), "image/webp") {
				contentType = "image/jpeg"
			}
		} else {
			fmt.Printf("compress failed: %v\n", err)
		}
	}

	if err := ctrl.Repository.SetImage(ctx, compressedKey, toCache, contentType); err != nil {
		fmt.Printf("cache failed: %v\n", err)
	}

	if isCompressed {
		c.Header("X-Compressed", "true")
	}
	c.Header("X-From-Cache", "false")
	if ctrl.Config.EnvConfig.Limit.CacheSize > 0 {
		c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", ctrl.Config.EnvConfig.Limit.CacheTime))
	} else {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
	}
	c.Data(http.StatusOK, contentType, toCache)
}
