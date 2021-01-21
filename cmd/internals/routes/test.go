package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/gin-gonic/gin"
	"net/http"
)

func TestRoute(r *gin.Engine) {
	r.GET("/testInsertDB", func(c *gin.Context) {
		res := models.TestDb()
		c.JSON(http.StatusOK, res)
	})
	r.GET("/testFs", func(c *gin.Context) {

	})
}
