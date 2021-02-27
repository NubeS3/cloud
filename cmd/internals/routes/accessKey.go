package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"time"
)

func AccessKeyRoutes(r *gin.Engine) {
	ar := r.Group("/accessKey", middlewares.UserAuthenticate)
	{
		ar.GET("/all/:bucket_id", func(c *gin.Context) {
			bucketId := c.Param("bucket_id")
			limit := c.DefaultQuery("limit", "0")
			offset := c.DefaultQuery("offset", "0")
			iLimit, err := strconv.Atoi(limit)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "wrong limit format",
				})

				return
			}

			iOffset, err := strconv.Atoi(offset)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "wrong offset format",
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

			accessKeys, err := arango.FindAccessKeyByUidBid(uid.(string), bucketId, iLimit, iOffset)
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

			accessKey, err := arango.FindAccessKeyByKey(key)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /accessKey/info/:access_key:")
				log.Println(err)
				return
			}

			if accessKey.Uid != uid.(string) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "invalid user ownership",
				})

				return
			}

			c.JSON(http.StatusOK, accessKey)
		})
		ar.POST("/create", func(c *gin.Context) {
			type createAKeyData struct {
				BucketId    string    `json:"bucket_id"`
				ExpiredDate time.Time `json:"expired_date"`
				Permissions []string  `json:"permissions"`
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

			res, err := arango.GenerateAccessKey(keyData.BucketId, uid.(string), keyData.Permissions, keyData.ExpiredDate)
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

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /buckets/create:")
				log.Println("uid not found in authenticated route")
				return
			}

			if err := arango.DeleteAccessKey(key, bucketId, uid.(string)); err != nil {
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
