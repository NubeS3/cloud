package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/cassandra"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func BucketRoutes(r *gin.Engine) {
	ar := r.Group("/buckets", middlewares.UserAuthenticate)
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

				cassandra.ErrLog("uid not found in authenticated route at /buckets/all:",
					"Unknown Error")
				return
			}

			res, err := arango.FindBucketByUid(uid.(string), limit, offset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				cassandra.ErrLog(err.Error()+" at buckets/all:",
					"Db Error")

				return
			}

			c.JSON(http.StatusOK, res)
		})
		ar.POST("/create", func(c *gin.Context) {
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
			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				cassandra.ErrLog("uid not found in authenticated route at /buckets/create:",
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

				cassandra.ErrLog(err.Error()+" at buckets/create:",
					"Db Error")

				return
			}
			c.JSON(http.StatusOK, bucket)
		})
		ar.DELETE("/delete/:bucket_id", middlewares.UserAuthenticate, func(c *gin.Context) {
			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				cassandra.ErrLog("uid not found in authenticated route at /buckets/create:",
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

				cassandra.ErrLog(err.Error()+" at /buckets/delete:",
					"Db Error")

				return
			}
			c.JSON(http.StatusOK, gin.H{
				"message": "delete success.",
			})
		})
	}

}
