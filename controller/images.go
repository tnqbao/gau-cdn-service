package controller

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-cdn-service/utils"
	"net/http"
	"strings"
)

const maxFetchSize = 10 * 1024 * 1024 // 10 MB max R2 image size

func (ctrl *Controller) GetImage(c *gin.Context) {
	filepath := c.Param("filepath")
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

	data, contentType, err = ctrl.Infra.CloudflareR2Client.GetObjectWithLimit(ctx, key, maxFetchSize)
	if err != nil {
		utils.JSON404(c, "file not found or too large")
		return
	}

	toCache := data
	isCompressed := false

	if len(data) > 100*1024 && !strings.HasSuffix(contentType, "jpeg") {
		if compressed, err := compressToJPEGUnder100KB(data, 1024); err == nil {
			toCache = compressed
			contentType = "image/jpeg"
			isCompressed = true
		} else {
			fmt.Printf("⚠️ compress failed: %v\n", err)
		}
	}

	if err := ctrl.Repository.SetImage(ctx, compressedKey, toCache, contentType); err != nil {
		fmt.Printf("⚠️ cache failed: %v\n", err)
	}

	if isCompressed {
		c.Header("X-Compressed", "true")
	}
	c.Header("X-From-Cache", "false")
	c.Data(http.StatusOK, contentType, toCache)
}
