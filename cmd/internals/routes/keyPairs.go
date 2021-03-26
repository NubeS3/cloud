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

func KeyPairsRoutes(r *gin.Engine) {
	ar := r.Group("/auth/keyPairs", middlewares.UserAuthenticate)
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

				_ = nats.SendErrorEvent("uid not found in authenticated route at keyPairs/buckets/all:",
					"Unknown Error")
				return
			}

			keyPairs, err := arango.FindKeyByUidBid(uid.(string), bucketId, iLimit, iOffset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+"at keyPairs/buckets/all:",
					"Db Error")
				return
			}

			c.JSON(http.StatusOK, keyPairs)
		})
		ar.GET("/info/:public_key", func(c *gin.Context) {
			key := c.Param("public_key")

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("uid not found in authenticated route at /keyPairs/info/:public_key:",
					"Unknown Error")
				return
			}

			keyPair, err := arango.FindKeyPairByPublic(key)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at /keyPairs/info/:public_key:",
					"Db Error")
				return
			}

			if keyPair.GeneratorUid != uid.(string) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "invalid user ownership",
				})

				return
			}

			c.JSON(http.StatusOK, keyPair)
		})
		ar.POST("/create", func(c *gin.Context) {
			type createKeyPairData struct {
				BucketId    string    `json:"bucket_id"`
				ExpiredDate time.Time `json:"expired_date"`
				Permissions []string  `json:"permissions"`
			}

			var keyData createKeyPairData
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

				_ = nats.SendErrorEvent("uid not found in authenticated route at /keyPairs/create:",
					"Unknown Error")
				return
			}

			res, err := arango.GenerateKeyPair(keyData.BucketId, uid.(string), keyData.ExpiredDate, keyData.Permissions)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})

				return
			}

			c.JSON(http.StatusOK, gin.H{
				"public_key":  res.Public,
				"private_key": res.Private,
			})

			ar.DELETE("/delete/:bucket_id/:public_key", func(c *gin.Context) {
				key := c.Param("public_key")
				bucketId := c.Param("bucket_id")

				uid, ok := c.Get("uid")
				if !ok {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "something went wrong",
					})

					_ = nats.SendErrorEvent("uid not found in authenticated route at /keyPairs/delete:",
						"Unknown Error")
					return
				}

				if err := arango.DeleteKeyPair(key, bucketId, uid.(string)); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "something went wrong",
					})

					_ = nats.SendErrorEvent(err.Error()+" at /keyPairs/delete:",
						"Db Error")

					return
				}

				c.JSON(http.StatusOK, key)
			})
		})
	}
}
