package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/routes/adminHandler"
	"github.com/gin-gonic/gin"
)

func AdminRoutes(route *gin.Engine) {
	adminRoutesGroup := route.Group("/admin")
	{
		adminRoutesGroup.POST("/signin", adminHandler.AdminSigninHandler)

		aar := adminRoutesGroup.Group("/auth", middlewares.AdminAuthenticate)
		{
			aar.GET("/test", adminHandler.AdminTestHandler)
			aar.POST("/mod", adminHandler.AdminCreateMod)
			aar.PATCH("/disable-mod", adminHandler.AdminModDisable)
			aar.PATCH("/ban-user", adminHandler.AdminBanUser)
		}
	}
}
