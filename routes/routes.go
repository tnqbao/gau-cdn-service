package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-cdn-service/controller"
)

func SetupRouter(ctrl *controller.Controller) *gin.Engine {
	r := gin.Default()
	apiRoutes := r.Group("/images")
	{
		//apiRoutes.POST("/upload", ctrl.UploadFile)
		apiRoutes.GET("/*filePath", ctrl.GetImage)
	}
	return r
}
