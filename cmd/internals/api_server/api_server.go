package api_server

import (
	"fmt"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/gin-gonic/gin"
	"net/http"
)

func Run() {
	fmt.Println("Initialize DB connection")
	err := models.IninDb()
	if err != nil {
		panic(err)
	}
	fmt.Println("Starting Cloud Server")
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "OK!",
		})
	})

	_ = r.Run(":8080")
}
