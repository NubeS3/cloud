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
)

func BucketRoutes(r *gin.Engine) {
	ar := r.Group("/auth/buckets", middlewares.UserAuthenticate, middlewares.AuthReqCount)
	{
		ar.GET("/all", func(c *gin.Context) {
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

			res, err := arango.FindBucketByUid(uid.(string), limit, offset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at buckets/all:",
					"Db Error")

				return
			}

			c.JSON(http.StatusOK, res)
		})
		ar.POST("/", func(c *gin.Context) {
			type createBucket struct {
				Name   string `json:"name" binding:"required"`
				Region string `json:"region" binding:"required"`
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

				_ = nats.SendErrorEvent("create bucket > "+err.Error(), "validate")
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

				_ = nats.SendErrorEvent("uid not found in authenticated route at /buckets/create:",
					"Unknown Error")

				return
			}

			bucket, err := arango.InsertBucket(uid.(string), curCreateBucket.Name, curCreateBucket.Region)
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

				_ = nats.SendErrorEvent(err.Error()+" at auth/buckets/create:",
					"Db Error")

				return
			}
			c.JSON(http.StatusOK, bucket)
		})
		ar.DELETE("/:bucket_id", func(c *gin.Context) {
			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("uid not found in authenticated route at auth/buckets/create:",
					"Unknown Error")

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

				_ = nats.SendErrorEvent(err.Error()+" at auth/buckets/delete:",
					"Db Error")

				return
			}
			c.JSON(http.StatusOK, gin.H{
				"message": "delete success.",
			})
		})

		ar.GET("/size/:bucket_id", func(c *gin.Context) {
			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("uid not found in authenticated route at /buckets/create:",
					"Unknown Error")

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

				_ = nats.SendErrorEvent("error at auth/bucket/size: "+err.Error(), "db error")
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

				_ = nats.SendErrorEvent("bucket size not found in authenticated route at /buckets/create: "+err.Error(),
					"Unknown Error")

				return
			}

			c.JSON(http.StatusOK, bucketSize)
		})
		ar.GET("/object-count/:bucket_id", func(c *gin.Context) {
			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("uid not found in authenticated route at /buckets/create:",
					"Unknown Error")

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

				_ = nats.SendErrorEvent("error at auth/bucket/size: "+err.Error(), "db error")
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

				_ = nats.SendErrorEvent("error at auth/bucket/size: "+err.Error(), "db error")
				return
			}

			c.JSON(http.StatusOK, count)
		})
	}

}
