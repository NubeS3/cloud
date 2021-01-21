package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/gin-gonic/gin"
	"net/http"
)

func ApiKeyRoute(route *gin.Engine) {
	apiKeyRoutesGroup := route.Group("/accessKey")
	{
		apiKeyRoutesGroup.POST("/create", middlewares.UserAuthenticate, func(c *gin.Context) {
			accessKey, err := models.GenerateAccessKey()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})
				return
			}

			c.JSON(http.StatusOK, accessKey)
		})

		apiKeyRoutesGroup.GET("/bucket", middlewares.UserAuthenticate, func(c *gin.Context) {

		})
	}
}
