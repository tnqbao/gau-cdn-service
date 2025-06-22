package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-cdn-service/controller"
)

func SetupRouter(ctrl *controller.Controller) *gin.Engine {
	r := gin.Default()
	apiRoutes := r.Group("/uploads")
	{
		//apiRoutes.POST("/upload", ctrl.UploadFile)
		apiRoutes.GET("/:fileName", ctrl.GetImage)
	}
	return r
}
