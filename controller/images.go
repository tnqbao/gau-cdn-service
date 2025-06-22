package controller

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (ctrl *Controller) GetImage(c *gin.Context) {
	filepath := c.Param("filepath") // example: "/user123/avatar.jpg"
	if filepath == "" || filepath == "/" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
		return
	}

	key := filepath[1:] // Bỏ dấu slash đầu → "user123/avatar.jpg"
	compressedKey := "compressed:" + key
	ctx := context.Background()

	// redis cache check
	data, contentType, err := ctrl.Repository.GetImage(ctx, compressedKey)
	if err == nil && len(data) > 0 {
		c.Header("X-From-Cache", "true")
		c.Data(http.StatusOK, contentType, data)
		return
	}

	// fetch từ R2
	data, contentType, err = ctrl.Infra.CloudflareR2Client.GetObject(ctx, key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// compress if file size > 100KB
	isCompressed := false
	toCache := data

	if len(data) > 100*1024 {
		if compressed, err := compressToJPEGUnder100KB(data, 1024); err == nil {
			toCache = compressed
			contentType = "image/jpeg"
			isCompressed = true
		} else {
			fmt.Printf("⚠️ compress failed: %v\n", err)
		}
	}

	// cache
	if err := ctrl.Repository.SetImage(ctx, compressedKey, toCache, contentType); err != nil {
		fmt.Printf("⚠️ cache failed: %v\n", err)
	}

	// response image
	if isCompressed {
		c.Header("X-Compressed", "true")
	}
	c.Header("X-From-Cache", "false")
	c.Data(http.StatusOK, contentType, toCache)
}
