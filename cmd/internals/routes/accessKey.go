package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

func AccessKeyRoutes(r *gin.Engine) {
	ar := r.Group("/auth/accessKey", middlewares.UserAuthenticate, middlewares.AuthReqCount)
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

				_ = nats.SendErrorEvent("uid not found in authenticated route at /buckets/all:",
					"Unknown Error")
				return
			}

			accessKeys, err := arango.FindAccessKeyByUidBid(uid.(string), bucketId, iLimit, iOffset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+"at /buckets/all:",
					"Db Error")
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

				_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKey/info/:access_key:",
					"Unknown Error")
				return
			}

			accessKey, err := arango.FindAccessKeyByKey(key)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at /accessKey/info/:access_key:",
					"Db Error")
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
		ar.POST("/", func(c *gin.Context) {
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

				_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKeys/create:",
					"Unknown Error")
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
		ar.DELETE("/:bucket_id/:access_key", func(c *gin.Context) {
			key := c.Param("access_key")
			bucketId := c.Param("bucket_id")

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKeys/delete:",
					"Unknown Error")
				return
			}

			if err := arango.DeleteAccessKey(key, bucketId, uid.(string)); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at /accessKeys/delete:",
					"Db Error")

				return
			}

			c.JSON(http.StatusOK, key)
		})
	}
}
