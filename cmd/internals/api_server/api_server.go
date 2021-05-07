package api_server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/NubeS3/cloud/cmd/internals/models/seaweedfs"
	"github.com/NubeS3/cloud/cmd/internals/routes"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"golang.org/x/crypto/acme/autocert"
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
	routes.PingRoute(r)
	routes.UserRoutes(r)
	routes.BucketRoutes(r)
	routes.AccessKeyRoutes(r)
	routes.KeyPairsRoutes(r)
	routes.FileRoutes(r)
	routes.FolderRoutes(r)
	routes.AdminRoutes(r)
}

func Run() {
	fmt.Println("Initializing utilities...")
	ultis.InitUtilities()

	// fmt.Println("Initialize Log DB connection")
	// err := cassandra.InitCassandraDb()
	// if err != nil {
	// 	panic(err)
	// }

	fmt.Println("Initialize DB connection")
	err := arango.InitArangoDb()
	if err != nil {
		panic(err)
	}

	fmt.Println("Initialize SeaweedFS connection")
	err = seaweedfs.InitFs()
	if err != nil {
		panic(err)
	}
	defer seaweedfs.CleanUp()

	err = nats.InitNats()
	if err != nil {
		panic(err)
	}
	defer nats.CleanUp()

	ultis.InitMailService()

	fmt.Println("Starting Cloud Server")
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Refresh", "Content-Length", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length", "AccessToken", "RefreshToken", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "OK!",
		})
	})
	Routing(r)

	m := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("nubes3.xyz"),
		Cache:      autocert.DirCache("/var/www/.cache"),
	}

	log.Fatal(autotls.RunWithManager(r, &m))
	//r.Run(":6160")
}
