package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

func FileRoutes(r *gin.Engine) {
	acr := r.Group("/files", middlewares.ApiKeyAuthenticate)
	{
		acr.GET("/all", func(c *gin.Context) {
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

			key, ok := c.Get("accessKey")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /files/all:")
				log.Println("accessKey not found in authenticate")
				return
			}
			accessKey := key.(*arango.AccessKey)
			var isUploadPerm bool
			for _, perm := range accessKey.Permissions {
				if perm == "GetFileList" {
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

			res, err := arango.FindMetadataByBid(accessKey.BucketId, limit, offset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				log.Println("at files/all:")
				log.Println(err)
				return
			}

			c.JSON(http.StatusOK, res)
		})

		acr.POST("/upload", func(c *gin.Context) {
			key, ok := c.Get("accessKey")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /files/upload:")
				log.Println("accessKey not found in authenticate")
				return
			}

			accessKey := key.(*arango.AccessKey)
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

			uploadFile, err := c.FormFile("file")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}
			path := c.DefaultPostForm("path", "/")
			//TODO Validate path format

			//END TODO

			fileName := c.DefaultPostForm("name", uploadFile.Filename)
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
			ttlStr := c.DefaultPostForm("ttl", "0")
			ttl, err := strconv.ParseInt(ttlStr, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
			}

			isHiddenStr := c.DefaultPostForm("hidden", "false")
			isHidden, err := strconv.ParseBool(isHiddenStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})

				return
			}

			cType, err := ultis.GetFileContentType(fileContent)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "unknown file content type",
				})

				return
			}

			res, err := arango.SaveFile(fileContent, accessKey.BucketId, path, fileName, isHidden,
				cType, fileSize, time.Duration(ttl))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, res)
		})

		acr.GET("/download/:file_id", func(c *gin.Context) {
			key, ok := c.Get("accessKey")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /files/download:")
				log.Println("accessKey not found in authenticate")
				return
			}
			accessKey := key.(*arango.AccessKey)
			var isDownloadPerm bool
			for _, perm := range accessKey.Permissions {
				if perm == "Download" {
					isDownloadPerm = true
					break
				}
			}
			if !isDownloadPerm {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "not have permission",
				})
				return
			}
			fid := c.Param("file_id")

			err := arango.GetFileByFid(fid, func(reader io.Reader, metadata *arango.FileMetadata) error {
				if metadata.BucketId != accessKey.BucketId {
					return &models.RouteError{
						Msg:     "invalid bucket",
						ErrType: models.InvalidBucket,
					}
				}

				extraHeaders := map[string]string{
					"Content-Disposition": `attachment; filename=` + metadata.Name,
				}

				c.DataFromReader(http.StatusOK, metadata.Size, metadata.ContentType, reader, extraHeaders)
				return nil
			})

			if err != nil {
				if e, ok := err.(*models.RouteError); ok {
					if e.ErrType == models.InvalidBucket {
						c.JSON(http.StatusForbidden, gin.H{
							"error": err.Error(),
						})

						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				log.Println("at /files/download:")
				log.Println("download failed: " + err.Error())
				return
			}
		})
	}

	ar := r.Group("/files/auth", middlewares.UserAuthenticate)
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
			bid := c.Query("bid")
			if bid == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "missing bid",
				})

				return
			}

			bucket, err := arango.FindBucketById(bid)
			if err != nil {
				if e, ok := err.(*models.ModelError); ok {
					if e.ErrType == models.DocumentNotFound {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": "bid invalid",
						})

						return
					}
					if e.ErrType == models.DbError {
						c.JSON(http.StatusInternalServerError, gin.H{
							"error": "something when wrong",
						})

						log.Println("at authenticated files/all:")
						log.Println(err)
						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				log.Println("at authenticated files/all:")
				log.Println(err)
				return
			}

			if uid, ok := c.Get("uid"); !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				log.Println("at authenticated files/all:")
				log.Println(err)
				return
			} else {
				if uid.(string) != bucket.Uid {
					c.JSON(http.StatusForbidden, gin.H{
						"error": "permission denied",
					})
					return
				}
			}

			res, err := arango.FindMetadataByBid(bid, limit, offset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				log.Println("at authenticated files/all:")
				log.Println(err)
				return
			}

			c.JSON(http.StatusOK, res)
		})
	}
}
