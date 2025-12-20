package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-cdn-service/controller"
)

func SetupRouter(ctrl *controller.Controller) *gin.Engine {
	r := gin.Default()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// CDN file serving with flexible URL patterns:
	// - /:bucket/filename.ext
	// - /:bucket/folder/filename.ext
	// - /:bucket/folder1/folder2/filename.ext
	r.GET("/:bucket/*path", ctrl.GetFile)

	return r
}
