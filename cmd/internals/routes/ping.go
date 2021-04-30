package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
)

func PingRoute(r *gin.Engine) {
	r.GET("/ping", middlewares.UnauthReqCount, func(c *gin.Context) {
		c.JSON(http.StatusOK, "pong")
	})
}
