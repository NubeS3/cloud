package api_server

import (
	"fmt"
	"github.com/NubeS3/cloud/cmd/internals/cron"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/NubeS3/cloud/cmd/internals/models/seaweedfs"
	"github.com/NubeS3/cloud/cmd/internals/routes"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"golang.org/x/crypto/acme/autocert"
	"log"
	"net/http"
	"strconv"
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
	//routes.TestRoute(r)
	routes.PingRoute(r)
	routes.UserRoutes(r)
	routes.BucketRoutes(r)
	routes.AccessKeyRoutes(r)
	//routes.KeyPairsRoutes(r)
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

	fmt.Println("Initialize Nats connection")
	err = nats.InitNats()
	if err != nil {
		panic(err)
	}
	defer nats.CleanUp()

	//ultis.InitMailService()

	defer cron.CleanUp()

	fmt.Println("Starting Cloud Server")
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20 // 8MiB
	//r.Use(cors.New(cors.Config{
	//	AllowOrigins:     []string{"*"},
	//	AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
	//	AllowHeaders:     []string{"Origin", "Authorization", "Refresh", "Content-Length", "Content-Type"},
	//	ExposeHeaders:    []string{"Content-Length", "AccessToken", "RefreshToken", "Content-Type"},
	//	AllowCredentials: true,
	//	MaxAge:           12 * time.Hour,
	//}))

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "OK!",
		})
	})
	Routing(r)

	isProd := viper.GetBool("IS_PROD")
	if isProd {
		host := viper.GetString("HOST")
		m := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(host),
			Cache:      autocert.DirCache("/var/www/.cache"),
		}

		log.Fatal(autotls.RunWithManager(r, &m))
	} else {
		port := viper.GetInt("PORT")
		if port == 0 {
			port = 6160
		}
		println("Listening for request at port 6160")
		log.Fatal(r.Run(":" + strconv.Itoa(port)))
	}
}
