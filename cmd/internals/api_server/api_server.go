package api_server

import (
	"fmt"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/routes"
	"github.com/gin-gonic/gin"
	"net/http"
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
