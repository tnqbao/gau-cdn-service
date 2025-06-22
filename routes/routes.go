package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/tnqbao/gau-cdn-service/controller"
)

func SetupRouter(ctrl *controller.Controller) *gin.Engine {
	r := gin.Default()

	//useMiddlewares, err := middlewares.NewMiddlewares(config.EnvConfig, svc)
	//if err != nil {
	//	panic(err)
	//}
	//
	//r.Use(useMiddlewares.CORSMiddleware)
	//apiRoutes := r.Group("/api/cdn/v2")
	//{
	//	identifierRoutes := apiRoutes.Group("/basic")
	//	{
	//		identifierRoutes.POST("/register", ctrl.RegisterWithIdentifierAndPassword)
	//		identifierRoutes.POST("/login", ctrl.LoginWithIdentifierAndPassword)
	//	}
	//
	//	profileRoutes := apiRoutes.Group("/profile")
	//	{
	//		profileRoutes.Use(useMiddlewares.AuthMiddleware)
	//		profileRoutes.GET("/", ctrl.GetAccountInfo)
	//		profileRoutes.PUT("/", ctrl.UpdateAccountInfo)
	//	}
	//
	//	apiRoutes.GET("/token", ctrl.RenewAccessToken)
	//	apiRoutes.POST("/logout", ctrl.Logout, useMiddlewares.AuthMiddleware)
	//
	//	ssoRoutes := apiRoutes.Group("/sso")
	//	{
	//		ssoRoutes.POST("/google", ctrl.LoginWithGoogle)
	//	}
	//	checkRoutes := apiRoutes.Group("/check")
	//	{
	//		//checkRoutes.GET("/deployment", ctrl.TestDeployment)
	//		checkRoutes.GET("/", ctrl.CheckHealth)
	//	}
	//}
	return r
}
