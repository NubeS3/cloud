package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
	"time"
)

func FileRoutes(r *gin.Engine) {
	ar := r.Group("/files", middlewares.ApiKeyAuthenticate)
	{
		ar.POST("/upload", func(c *gin.Context) {
			key, ok := c.Get("accessKey")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /files/upload:")
				log.Println("accessKey not found in authenticate")
				return
			}
			accessKey := key.(models.AccessKey)
			var isUploadPerm bool
			for _, perm := range accessKey.Permissions {
				if perm == "Upload" {
					isUploadPerm = true
					break
				}
			}
			if !isUploadPerm {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "not have permission",
				})
				return
			}

			uploadFile, err := c.FormFile("upload_file")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}
			path := c.PostForm("path")
			//TODO Validate path format

			//END TODO
			fileName := c.PostForm("name")
			bucket, err := models.FindBucketById(accessKey.Uid, accessKey.BucketId)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /files/upload:")
				log.Println("bucket not found")
				return
			}
			//newPath := bucket.Name + path + fileName
			fileContent, err := uploadFile.Open()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /files/upload:")
				log.Println("open file failed")
				return
			}
			fileSize := uploadFile.Size
			ttl_str := c.PostForm("ttl")
			ttl, err := strconv.ParseInt(ttl_str, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
			}

			res, err := models.SaveFile(fileContent, bucket.Id,
				bucket.Name, path, fileName, false,
				uploadFile.Header.Get("Content-type"),
				fileSize, time.Duration(ttl))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "update successful.",
				"file":    res,
			})
		})
	}
}
