package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"log"
	"net/http"
	"time"
)

func AccessKeyRoutes(r *gin.Engine) {
	ar := r.Group("/accessKey", middlewares.UserAuthenticate)
	{
		ar.GET("/all/:bucket_id", func(c *gin.Context) {
			bucketId := c.Param("bucket_id")
			bid, err := gocql.ParseUUID(bucketId)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "bucket_id wrong format",
				})

				return
			}

			uid, ok := c.Get("uid")
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
		ar.GET("/info/:access_key", func(c *gin.Context) {
			key := c.Param("access_key")

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /accessKey/info/:access_key:")
				log.Println("uid not found in authenticated route")
				return
			}

			accessKey, err := models.FindAccessKeyByKey(key)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /accessKey/info/:access_key:")
				log.Println(err)
				return
			}

			if accessKey.Uid != uid.(gocql.UUID) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "not your key",
				})

				return
			}

			c.JSON(http.StatusOK, accessKey)
		})
		ar.POST("/create", func(c *gin.Context) {
			type createAKeyData struct {
				BucketId    gocql.UUID `json:"bucket_id"`
				ExpiredDate time.Time  `json:"expired_date"`
				Permissions []string   `json:"permissions"`
			}

			var keyData createAKeyData
			if err := c.ShouldBind(&keyData); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /buckets/create:")
				log.Println("uid not found in authenticated route")
				return
			}

			res, err := models.InsertAccessKey(keyData.BucketId, uid.(gocql.UUID), keyData.Permissions, keyData.ExpiredDate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})

				return
			}

			c.JSON(http.StatusOK, res.Key)
		})
		ar.DELETE("/delete/:bucket_id/:access_key", func(c *gin.Context) {
			key := c.Param("access_key")
			bucketId := c.Param("bucket_id")
			bid, err := gocql.ParseUUID(bucketId)

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "bucket_id mismatch",
				})

				return
			}

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /buckets/create:")
				log.Println("uid not found in authenticated route")
				return
			}

			if err := models.DeleteAccessKey(uid.(gocql.UUID), bid, key); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /buckets/create:")
				log.Println(err)

				return
			}

			c.JSON(http.StatusOK, key)
		})
	}
}
