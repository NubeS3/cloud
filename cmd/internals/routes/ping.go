package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/google/uuid"
	"net/http"

	"github.com/gin-gonic/gin"
)

func PingRoute(r *gin.Engine) {
	r.GET("/ping", middlewares.ReqLogger("unauth", ""), func(c *gin.Context) {
		c.JSON(http.StatusOK, "pong")
	})

	r.GET("/version", middlewares.ReqLogger("unauth", ""), func(c *gin.Context) {
		c.JSON(http.StatusOK, "v2.0.8")
	})

	id, err := uuid.NewUUID()
	if err != nil {
		panic(err)
	}

	r.GET("/instance-id", middlewares.ReqLogger("unauth", ""), func(c *gin.Context) {
		c.JSON(http.StatusOK, "instance-id: "+id.String())
	})
}
