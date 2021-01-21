package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/linxGnu/goseaweedfs"
)

func TestRoute(route *gin.Engine) {
	testRoutesGroup := route.Group("/test")
	{
		testRoutesGroup.POST("/upload", func(c *gin.Context) {

		})
	}
}
