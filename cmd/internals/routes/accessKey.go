package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

func AccessKeyRoutes(r *gin.Engine) {
	uar := r.Group("/get-auth-token")
	{
		uar.POST("/", func(c *gin.Context) {
			type reqKey struct {
				KeyId string `json:"key_id" binding:"required"`
				Key   string `json:"key"  binding:"required"`
			}

			var keyData reqKey
			if err := c.ShouldBind(&keyData); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			key, err := arango.FindAccessKeyById(keyData.KeyId)
			if err != nil {
				if err.(*models.ModelError).ErrType == models.NotFound || err.(*models.ModelError).ErrType == models.DocumentNotFound {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "key not found",
					})

					return
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at key/resolve:",
					"Db Error")

				return
			}

			token, err := ultis.CreateKeyToken(key.Id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("at /get-auth-token: "+err.Error(), "unknown")
				return
			}

			key.Key = "*****"
			c.JSON(http.StatusOK, gin.H{
				"auth_token": token,
				"key_info":   key,
			})
		})
	}

	ar := r.Group("/auth/accessKey", middlewares.UserAuthenticate, middlewares.AuthReqCount)
	{
		//ar.GET("/all/:bucket_id", func(c *gin.Context) {
		//	bucketId := c.Param("bucket_id")
		//	limit := c.DefaultQuery("limit", "0")
		//	offset := c.DefaultQuery("offset", "0")
		//	iLimit, err := strconv.Atoi(limit)
		//	if err != nil {
		//		c.JSON(http.StatusBadRequest, gin.H{
		//			"error": "wrong limit format",
		//		})
		//
		//		return
		//	}
		//
		//	iOffset, err := strconv.Atoi(offset)
		//	if err != nil {
		//		c.JSON(http.StatusBadRequest, gin.H{
		//			"error": "wrong offset format",
		//		})
		//
		//		return
		//	}
		//
		//	uid, ok := c.Get("uid")
		//	if !ok {
		//		c.JSON(http.StatusInternalServerError, gin.H{
		//			"error": "something went wrong",
		//		})
		//
		//		_ = nats.SendErrorEvent("uid not found in authenticated route at /buckets/all:",
		//			"Unknown Error")
		//		return
		//	}
		//
		//	accessKeys, err := arango.FindAccessKeyByUidBid(uid.(string), bucketId, iLimit, iOffset)
		//	if err != nil {
		//		c.JSON(http.StatusInternalServerError, gin.H{
		//			"error": "something went wrong",
		//		})
		//
		//		_ = nats.SendErrorEvent(err.Error()+"at /buckets/all:",
		//			"Db Error")
		//		return
		//	}
		//
		//	c.JSON(http.StatusOK, accessKeys)
		//})
		//ar.GET("/info/:access_key", func(c *gin.Context) {
		//	key := c.Param("access_key")
		//
		//	uid, ok := c.Get("uid")
		//	if !ok {
		//		c.JSON(http.StatusInternalServerError, gin.H{
		//			"error": "something went wrong",
		//		})
		//
		//		_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKey/info/:access_key:",
		//			"Unknown Error")
		//		return
		//	}
		//
		//	accessKey, err := arango.FindAccessKeyByKey(key)
		//	if err != nil {
		//		c.JSON(http.StatusInternalServerError, gin.H{
		//			"error": "something went wrong",
		//		})
		//
		//		_ = nats.SendErrorEvent(err.Error()+" at /accessKey/info/:access_key:",
		//			"Db Error")
		//		return
		//	}
		//
		//	if accessKey.Uid != uid.(string) {
		//		c.JSON(http.StatusForbidden, gin.H{
		//			"error": "invalid user ownership",
		//		})
		//
		//		return
		//	}
		//
		//	c.JSON(http.StatusOK, accessKey)
		//})
		ar.GET("/", func(c *gin.Context) {
			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKey/info/:access_key:",
				//	"Unknown Error")
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

			keys, err := arango.GetAccessKeyByUid(uid.(string), int(limit), int(offset))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				//_ = nats.SendErrorEvent(err.Error()+" at get all key:",
				//	"Db Error")

				return
			}

			c.JSON(http.StatusOK, keys)
		})
		ar.GET("/use-count/all/:access_key", func(c *gin.Context) {
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

			count, err := nats.CountAccessKeyReqCount(accessKey.Key, int(limit), int(offset))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("count access key usage err: "+err.Error(), "nats")
				return
			}

			c.JSON(http.StatusOK, count)
		})
		ar.GET("/use-count/date/:access_key", func(c *gin.Context) {
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

			count, err := nats.CountAccessKeyReqCountByDateRange(accessKey.Key, fromT, toT, int(limit), int(offset))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("count access key usage err: "+err.Error(), "nats")
				return
			}

			c.JSON(http.StatusOK, count)
		})
		ar.POST("/master", func(c *gin.Context) {
			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKeys/create:",
					"Unknown Error")
				return
			}

			key, err := arango.FindMasterKeyByUid(uid.(string))
			if err == nil {
				err = arango.DeleteAccessKeyById(key.Id)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "something when wrong",
					})

					_ = nats.SendErrorEvent(err.Error()+" at buckets/create:",
						"Db Error")

					return
				}
			}

			res, err := arango.GenerateMasterKey(uid.(string))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})

				return
			}

			c.JSON(http.StatusOK, res.Key)
		})
		ar.POST("/app", func(c *gin.Context) {
			type createAKeyData struct {
				Name                   string    `json:"name" binding:"required"`
				BucketId               *string   `json:"bucket_id"`
				ExpiredDate            time.Time `json:"expired_date"`
				Permissions            []string  `json:"permissions"`
				FilenamePrefixRestrict *string   `json:"filename_prefix_restrict"`
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

			_, err := arango.FindAppKeyByNameAndUid(uid.(string), keyData.Name)
			if err == nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "name duplicated",
				})

				return
			}

			if *keyData.BucketId == "*" {
				keyData.BucketId = nil
			}

			res, err := arango.GenerateApplicationKey(keyData.Name, keyData.BucketId, uid.(string), keyData.Permissions, keyData.ExpiredDate, *keyData.FilenamePrefixRestrict)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})

				return
			}

			c.JSON(http.StatusOK, res.Key)
		})
		ar.DELETE("/:id", func(c *gin.Context) {
			id := c.Param("id")

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKeys/delete:",
				//	"Unknown Error")
				return
			}

			key, err := arango.FindAccessKeyById(id)
			if err != nil {
				if err.(*models.ModelError).ErrType == models.NotFound || err.(*models.ModelError).ErrType == models.DocumentNotFound {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "key not found",
					})

					return
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				//_ = nats.SendErrorEvent(err.Error()+" at key/delete:",
				//	"Db Error")

				return
			}

			if key.Uid != uid.(string) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "not your key",
				})

				return
			}

			err = arango.DeleteAccessKeyById(id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				//_ = nats.SendErrorEvent(err.Error()+" at key/delete:",
				//	"Db Error")

				return
			}

			c.JSON(http.StatusOK, gin.H{
				"key": key.Id,
			})
		})
		//ar.DELETE("/:bucket_id/:access_key", func(c *gin.Context) {
		//	key := c.Param("access_key")
		//	bucketId := c.Param("bucket_id")
		//
		//	uid, ok := c.Get("uid")
		//	if !ok {
		//		c.JSON(http.StatusInternalServerError, gin.H{
		//			"error": "something went wrong",
		//		})
		//
		//		_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKeys/delete:",
		//			"Unknown Error")
		//		return
		//	}
		//
		//	if err := arango.DeleteAccessKey(key, bucketId, uid.(string)); err != nil {
		//		c.JSON(http.StatusInternalServerError, gin.H{
		//			"error": "something went wrong",
		//		})
		//
		//		_ = nats.SendErrorEvent(err.Error()+" at /accessKeys/delete:",
		//			"Db Error")
		//
		//		return
		//	}
		//
		//	c.JSON(http.StatusOK, key)
		//})
	}

	kr := r.Group("/apiKey/accessKey", middlewares.AccessKeyAuthenticate, middlewares.AccessKeyReqCount)
	{
		kr.GET("/", func(c *gin.Context) {
			k, ok := c.Get("key")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKey/info/:access_key:",
				//	"Unknown Error")
				return
			}

			key := k.(*arango.AccessKey)
			hasPerm, err := CheckPerm(key, arango.ListKeys)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//TODO LOG wrong permission
				return
			}
			if !hasPerm {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "missing permission",
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

			keys, err := arango.GetAccessKeyByUid(key.Uid, int(limit), int(offset))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				//_ = nats.SendErrorEvent(err.Error()+" at get all key:",
				//	"Db Error")

				return
			}

			c.JSON(http.StatusOK, keys)
		})
		kr.POST("/app", func(c *gin.Context) {
			k, ok := c.Get("key")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKey/info/:access_key:",
				//	"Unknown Error")
				return
			}

			key := k.(*arango.AccessKey)
			hasPerm, err := CheckPerm(key, arango.WriteKey)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//TODO LOG wrong permission
				return
			}
			if !hasPerm {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "missing permission",
				})

				return
			}

			type createAKeyData struct {
				Name                   string    `json:"name" binding:"required"`
				BucketId               *string   `json:"bucket_id"`
				ExpiredDate            time.Time `json:"expired_date"`
				Permissions            []string  `json:"permissions"`
				FilenamePrefixRestrict *string   `json:"filename_prefix_restrict"`
			}

			var keyData createAKeyData
			if err := c.ShouldBind(&keyData); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			_, err = arango.FindAppKeyByNameAndUid(key.Uid, keyData.Name)
			if err == nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "name duplicated",
				})

				return
			}

			if *keyData.BucketId == "*" {
				keyData.BucketId = nil
			}

			res, err := arango.GenerateApplicationKey(keyData.Name, keyData.BucketId, key.Uid, keyData.Permissions, keyData.ExpiredDate, *keyData.FilenamePrefixRestrict)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})

				return
			}

			c.JSON(http.StatusOK, res.Key)
		})
		kr.DELETE("/:id", func(c *gin.Context) {
			id := c.Param("id")

			k, ok := c.Get("key")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKey/info/:access_key:",
				//	"Unknown Error")
				return
			}

			uKey := k.(*arango.AccessKey)
			hasPerm, err := CheckPerm(uKey, arango.DeleteKey)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//TODO LOG wrong permission
				return
			}
			if !hasPerm {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "missing permission",
				})

				return
			}

			key, err := arango.FindAccessKeyById(id)
			if err != nil {
				if err.(*models.ModelError).ErrType == models.NotFound || err.(*models.ModelError).ErrType == models.DocumentNotFound {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "key not found",
					})

					return
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				//_ = nats.SendErrorEvent(err.Error()+" at key/delete:",
				//	"Db Error")

				return
			}

			if key.Uid != uKey.Uid {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "not your key",
				})

				return
			}

			err = arango.DeleteAccessKeyById(id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				//_ = nats.SendErrorEvent(err.Error()+" at key/delete:",
				//	"Db Error")

				return
			}

			c.JSON(http.StatusOK, gin.H{
				"key": key.Id,
			})
		})
	}
}

func CheckPerm(key *arango.AccessKey, perm arango.Permission) (hasPerm bool, err error) {
	hasPerm = false
	err = nil
	for _, k := range key.Permissions {
		p, err := arango.ParsePerm(k)
		if err != nil {
			return false, err
		}
		if perm == p {
			hasPerm = true
			break
		}
	}
	return
}
