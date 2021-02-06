package api_server

import (
	"fmt"
	"net/http"

	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/routes"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/gin-gonic/gin"
)

func Routing(r *gin.Engine) {
	routes.TestRoute(r)
	routes.UserRoutes(r)
}

func Run() {
	fmt.Println("Initialize DB connection")
	err := models.InitDb()
	if err != nil {
		panic(err)
	}

	err = models.InitFs()
	if err != nil {
		panic(err)
	}

	ultis.InitMailService()

	defer models.CleanUp()

	fmt.Println("Starting Cloud Server")
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "OK!",
		})
	})
	Routing(r)

	_ = r.Run(":6160")
}
