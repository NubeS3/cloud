package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/gin-gonic/gin"
	"github.com/m1ome/randstr"
	"net/http"
	"strconv"
	"time"
)

func BucketRoutes(r *gin.Engine) {
	ar := r.Group("/auth/buckets", middlewares.UserAuthenticate)
	{
		ar.GET("/all", middlewares.ReqLogger("auth", "C"), func(c *gin.Context) {
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
			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("uid not found in authenticated route at /buckets/all:",
					"Unknown Error")
				return
			}

			//res, err := arango.FindBucketByUid(uid.(string), limit, offset)
			res, err := arango.FindDetailBucketByUid(uid.(string), limit, offset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")

				return
			}

			c.JSON(http.StatusOK, res)
		})
		ar.POST("/", middlewares.ReqLogger("auth", "C"), func(c *gin.Context) {
			type createBucket struct {
				Name string `json:"name" binding:"required"`
				//Region string `json:"region" binding:"required"`
				IsPublic     *bool `json:"is_public" binding:"required"`
				IsEncrypted  *bool `json:"is_encrypted" binding:"required"`
				IsObjectLock *bool `json:"is_object_lock" binding:"required"`

				Passphrase *string `json:"passphrase"`
				Duration   int     `json:"duration"`
			}

			var curCreateBucket createBucket
			if err := c.ShouldBind(&curCreateBucket); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			if ok, err := ultis.ValidateBucketName(curCreateBucket.Name); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Validate Error")
				return
			} else if !ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Bucket name must be 8-64 characters, contains only alphanumeric",
				})

				return
			}

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err := nats.SendErrorEvent("uid not found at  /auth/buckets/",
					"Unknown Error")
				print(err)

				return
			}

			bucket, err := arango.InsertBucket(uid.(string), curCreateBucket.Name, *curCreateBucket.IsPublic, *curCreateBucket.IsEncrypted, *curCreateBucket.IsObjectLock)
			if err != nil {
				if err.(*models.ModelError).ErrType == models.Duplicated {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "duplicated bucket name",
					})

					return
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")

				return
			}

			if curCreateBucket.IsEncrypted != nil {
				if *curCreateBucket.IsEncrypted {
					if curCreateBucket.Passphrase == nil {
						passph := randstr.GetString(16)
						curCreateBucket.Passphrase = &passph
					}
					_, err = arango.CreateEncrypt(*curCreateBucket.Passphrase, bucket.Id)
					//TODO temp ignore errors

				}
			}

			if curCreateBucket.IsObjectLock != nil {
				if *curCreateBucket.IsObjectLock {
					bucket, err = arango.UpdateHoldDuration(bucket.Id, time.Duration(curCreateBucket.Duration*1_000_000_000))
				} else {
					bucket, err = arango.UpdateHoldDuration(bucket.Id, 0)
				}
			}

			c.JSON(http.StatusOK, bucket)
		})
		ar.PUT("/:bucket_id", middlewares.ReqLogger("auth", "C"), func(c *gin.Context) {
			type updateBucket struct {
				//Region string `json:"region" binding:"required"`
				IsPublic     *bool `json:"is_public"`
				IsEncrypted  *bool `json:"is_encrypted"`
				IsObjectLock *bool `json:"is_object_lock"`

				Passphrase *string `json:"passphrase"`
				Duration   *int    `json:"duration"`
			}

			var curUpdateBucket updateBucket
			if err := c.ShouldBind(&curUpdateBucket); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			bid := c.Param("bucket_id")

			updateResult, err := arango.UpdateBucketById(bid, curUpdateBucket.IsPublic, curUpdateBucket.IsEncrypted, curUpdateBucket.IsObjectLock)
			if err != nil {
				if err.(*models.ModelError).ErrType == models.NotFound || err.(*models.ModelError).ErrType == models.DocumentNotFound {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "bucket not found",
					})

					return
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")

				return
			}

			if updateResult.Old.IsEncrypted != updateResult.New.IsEncrypted {
				if *curUpdateBucket.IsEncrypted {
					if curUpdateBucket.Passphrase == nil {
						passph := randstr.GetString(16)
						curUpdateBucket.Passphrase = &passph
					}
					_, err = arango.CreateEncrypt(*curUpdateBucket.Passphrase, bid)
					//TODO temp ignore errors

				} else {
					_, err = arango.UpdateToEncryptionInfoByBucketId(bid)
					//TODO temp ignore errors
				}
			}

			var bucket *arango.Bucket
			if updateResult.Old.IsObjectLock != updateResult.New.IsObjectLock {
				if *curUpdateBucket.IsObjectLock {
					bucket, err = arango.UpdateHoldDuration(updateResult.New.Id, time.Duration(*curUpdateBucket.Duration*1_000_000_000))
					//TODO temp ignore errors
				} else {
					bucket, err = arango.UpdateHoldDuration(updateResult.New.Id, 0)
					//TODO temp ignore errors
				}
			}
			if bucket != nil {
				bucket = &updateResult.New
			}

			c.JSON(http.StatusOK, bucket)
		})
		ar.DELETE("/:bucket_id", middlewares.ReqLogger("auth", "A"), func(c *gin.Context) {
			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err := nats.SendErrorEvent("uid not found in authenticated route at auth/buckets/:bucket_id",
					"Unknown Error")
				print(err)
				return
			}

			bucketId := c.Param("bucket_id")
			//id, err := gocql.ParseUUID(bucketId)
			//if err != nil {
			//	c.JSON(http.StatusInternalServerError, gin.H{
			//		"error": "something went wrong",
			//	})
			//
			//	log.Println("at /buckets/delete:")
			//	log.Println("parse bucket_id failed")
			//	return
			//}

			if err := arango.RemoveBucket(uid.(string), bucketId); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")

				return
			}
			c.JSON(http.StatusOK, gin.H{
				"message": "delete success.",
			})
		})

		ar.GET("/size/:bucket_id", middlewares.ReqLogger("auth", "A"), func(c *gin.Context) {
			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err := nats.SendErrorEvent("key not found at auth/buckets/size/:bucket_id",
					"Unknown Error")
				print(err)

				return
			}

			bucketId := c.Param("bucket_id")
			bucket, err := arango.FindBucketById(bucketId)
			if err != nil {
				if err, ok := err.(*models.ModelError); ok {
					if err.ErrType == models.NotFound || err.ErrType == models.DocumentNotFound {
						c.JSON(http.StatusNotFound, gin.H{
							"error": "bucket not found",
						})

						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			if bucket.Uid != uid {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "not your bucket",
				})

				return
			}

			bucketSize, err := arango.GetBucketSizeById(bucketId)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")

				return
			}

			c.JSON(http.StatusOK, bucketSize)
		})
		ar.GET("/object-count/:bucket_id", middlewares.ReqLogger("auth", "A"), func(c *gin.Context) {
			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err := nats.SendErrorEvent("uid not found at /auth/buckets/object-count/:bucket_id",
					"Unknown Error")
				print(err)
				return
			}

			bucketId := c.Param("bucket_id")
			bucket, err := arango.FindBucketById(bucketId)
			if err != nil {
				if err, ok := err.(*models.ModelError); ok {
					if err.ErrType == models.NotFound || err.ErrType == models.DocumentNotFound {
						c.JSON(http.StatusNotFound, gin.H{
							"error": "bucket not found",
						})

						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			if bucket.Uid != uid {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "not your bucket",
				})

				return
			}

			count, err := arango.CountMetadataByBucketId(bucket.Id)
			if err != nil {
				if err, ok := err.(*models.ModelError); ok {
					if err.ErrType == models.NotFound || err.ErrType == models.DocumentNotFound {
						c.JSON(http.StatusNotFound, gin.H{
							"error": "bucket not found",
						})

						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			c.JSON(http.StatusOK, count)
		})
	}

	kr := r.Group("/apiKey/buckets", middlewares.AccessKeyAuthenticate)
	{
		kr.GET("/", middlewares.ReqLogger("key", "C"), func(c *gin.Context) {
			k, ok := c.Get("key")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err := nats.SendErrorEvent("key not found at get /apiKey/buckets/",
					"Unknown Error")
				print(err)
				return
			}

			key := k.(*arango.AccessKey)
			hasPerm, err := CheckPerm(key, arango.ListBuckets)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Key Error")
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

			//res, err := arango.FindBucketByUid(uid.(string), limit, offset)
			res, err := arango.FindDetailBucketByUid(key.Uid, limit, offset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")

				return
			}

			c.JSON(http.StatusOK, res)
		})
		kr.POST("/", middlewares.ReqLogger("key", "C"), func(c *gin.Context) {
			k, ok := c.Get("key")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err := nats.SendErrorEvent("key not found at post /apiKey/buckets/",
					"Unknown Error")
				print(err)
				return
			}

			key := k.(*arango.AccessKey)
			hasPerm, err := CheckPerm(key, arango.WriteBucket)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Key Error")
				return
			}
			if !hasPerm {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "missing permission",
				})

				return
			}

			type createBucket struct {
				Name string `json:"name" binding:"required"`
				//Region string `json:"region" binding:"required"`
				IsPublic     *bool `json:"is_public" binding:"required"`
				IsEncrypted  *bool `json:"is_encrypted" binding:"required"`
				IsObjectLock *bool `json:"is_object_lock" binding:"required"`

				Passphrase *string `json:"passphrase"`
				Duration   int     `json:"duration"`
			}

			var curCreateBucket createBucket
			if err := c.ShouldBind(&curCreateBucket); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			if ok, err := ultis.ValidateBucketName(curCreateBucket.Name); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Validate Error")
				return
			} else if !ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Bucket name must be 8-64 characters, contains only alphanumeric",
				})

				return
			}

			bucket, err := arango.InsertBucket(key.Uid, curCreateBucket.Name, *curCreateBucket.IsPublic, *curCreateBucket.IsEncrypted, *curCreateBucket.IsObjectLock)
			if err != nil {
				if err.(*models.ModelError).ErrType == models.Duplicated {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "duplicated bucket name",
					})

					return
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")

				return
			}

			if curCreateBucket.IsEncrypted != nil {
				if *curCreateBucket.IsEncrypted {
					if curCreateBucket.Passphrase == nil {
						passph := randstr.GetString(16)
						curCreateBucket.Passphrase = &passph
					}
					_, err = arango.CreateEncrypt(*curCreateBucket.Passphrase, bucket.Id)
					//TODO temp ignore errors

				}
			}

			if curCreateBucket.IsObjectLock != nil {
				if *curCreateBucket.IsObjectLock {
					bucket, err = arango.UpdateHoldDuration(bucket.Id, time.Duration(curCreateBucket.Duration*1_000_000_000))
				} else {
					bucket, err = arango.UpdateHoldDuration(bucket.Id, 0)
				}
			}

			c.JSON(http.StatusOK, bucket)
		})
		kr.DELETE("/:bucket_id", middlewares.ReqLogger("key", "A"), func(c *gin.Context) {
			k, ok := c.Get("key")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err := nats.SendErrorEvent("key not found at /apiKey/buckets/:bucket_id",
					"Unknown Error")
				print(err)
				return
			}

			key := k.(*arango.AccessKey)
			hasPerm, err := CheckPerm(key, arango.DeleteBucket)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Key Error")
				return
			}
			if !hasPerm {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "missing permission",
				})

				return
			}

			bucketId := c.Param("bucket_id")
			//id, err := gocql.ParseUUID(bucketId)
			//if err != nil {
			//	c.JSON(http.StatusInternalServerError, gin.H{
			//		"error": "something went wrong",
			//	})
			//
			//	log.Println("at /buckets/delete:")
			//	log.Println("parse bucket_id failed")
			//	return
			//}

			if err := arango.RemoveBucket(key.Uid, bucketId); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")

				return
			}
			c.JSON(http.StatusOK, gin.H{
				"message": "delete success.",
			})
		})
	}
}
