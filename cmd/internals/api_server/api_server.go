package api_server

import (
	"fmt"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/cassandra"
	"net/http"

	"github.com/NubeS3/cloud/cmd/internals/routes"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}

func Routing(r *gin.Engine) {
	routes.TestRoute(r)
	routes.UserRoutes(r)
	routes.BucketRoutes(r)
	routes.AccessKeyRoutes(r)
	routes.FileRoutes(r)
}

func Run() {
	fmt.Println("Initializing utilities...")
	ultis.InitUtilities()

	fmt.Println("Initialize Log DB connection")
	err := cassandra.InitCassandraDb()
	if err != nil {
		panic(err)
	}

	fmt.Println("Initialize DB connection")
	err = arango.InitArangoDb()
	if err != nil {
		panic(err)
	}

	err = cassandra.InitFs()
	if err != nil {
		panic(err)
	}

	ultis.InitMailService()

	defer cassandra.CleanUp()

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
