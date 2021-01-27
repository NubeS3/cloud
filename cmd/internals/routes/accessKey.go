package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"log"
	"net/http"
)

func AccessKeyRoutes(r *gin.Engine) {
	ar := r.Group("/accessKey", middlewares.UserAuthenticate)
	{
		ar.GET("/all", func(c *gin.Context) {
			bucketId := c.Param("bucket_id")
			bid, err := gocql.ParseUUID(bucketId)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "bucket_id wrong format",
				})

				return
			}

			uid, ok := c.Get("id")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /buckets/all:")
				log.Println("uid not found in authenticated route")
				return
			}

			accessKeys, err := models.FindAccessKeyByUidBid(uid.(gocql.UUID), bid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /buckets/all:")
				log.Println(err)
				return
			}

			c.JSON(http.StatusOK, accessKeys)
		})
	}
}
