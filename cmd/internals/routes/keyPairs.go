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
		ar.GET("/all/:bucket_id", middlewares.ReqLogger("auth", "C"), func(c *gin.Context) {
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
		ar.GET("/info/:public_key", middlewares.ReqLogger("auth", "B"), func(c *gin.Context) {
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
		ar.GET("/use-count/all/:public", middlewares.ReqLogger("auth", "B"), func(c *gin.Context) {
			key := c.Param("public")

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKey/info/:access_key:",
				//	"Unknown Error")
				return
			}

			kp, err := arango.FindKeyPairByPublic(key)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent(err.Error()+" at /accessKey/info/:access_key:",
				//	"Db Error")
				return
			}

			if kp.GeneratorUid != uid.(string) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "invalid user ownership",
				})

				return
			}

			limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid limit format",
				})

				return
			}

			offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid offset format",
				})

				return
			}

			count, err := nats.CountSignedReqCount(kp.Public, int(limit), int(offset))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("count signed key usage err: "+err.Error(), "nats")
				return
			}

			c.JSON(http.StatusOK, count)
		})
		ar.GET("/use-count/date/:public", middlewares.ReqLogger("auth", "B"), func(c *gin.Context) {
			key := c.Param("public")

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKey/info/:access_key:",
					"Unknown Error")
				return
			}

			kp, err := arango.FindKeyPairByPublic(key)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent(err.Error()+" at /accessKey/info/:access_key:",
				//	"Db Error")
				return
			}

			if kp.GeneratorUid != uid.(string) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "invalid user ownership",
				})

				return
			}

			limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid limit format",
				})

				return
			}

			offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid offset format",
				})

				return
			}

			from, err := strconv.ParseInt(c.DefaultQuery("from", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid from format",
				})

				return
			}

			to, err := strconv.ParseInt(c.DefaultQuery("to", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid from format",
				})

				return
			}

			fromT := time.Unix(from, 0)
			toT := time.Unix(to, 0)

			count, err := nats.CountAccessKeyReqCountByDateRange(kp.Public, fromT, toT, int(limit), int(offset))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("count signed usage err: "+err.Error(), "nats")
				return
			}

			c.JSON(http.StatusOK, count)
		})
		ar.POST("/", middlewares.ReqLogger("auth", "C"), func(c *gin.Context) {
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

				//_ = nats.SendErrorEvent("uid not found in authenticated route at /keyPairs/create:",
				//	"Unknown Error")
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

		})
		ar.DELETE("/:bucket_id/:public_key", middlewares.ReqLogger("auth", "A"), func(c *gin.Context) {
			key := c.Param("public_key")
			bucketId := c.Param("bucket_id")

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent("uid not found in authenticated route at /keyPairs/delete:",
				//	"Unknown Error")
				return
			}

			if err := arango.RemoveKeyPair(key, bucketId, uid.(string)); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent(err.Error()+" at /keyPairs/delete:",
				//	"Db Error")

				return
			}

			c.JSON(http.StatusOK, key)
		})
	}
}
