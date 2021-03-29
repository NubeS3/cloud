package routes

import (
	"archive/zip"
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strconv"
	"time"
)

func FileRoutes(r *gin.Engine) {
	acr := r.Group("/accessKey/files", middlewares.ApiKeyAuthenticate)
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

				_ = nats.SendErrorEvent("accessKey not found in authenticate at /files/all:",
					"Unknown Error")
				return
			}
			accessKey := key.(*arango.AccessKey)
			var isGetFileListPerm bool
			for _, perm := range accessKey.Permissions {
				if perm == "GetFileList" {
					isGetFileListPerm = true
					break
				}
			}

			if !isGetFileListPerm {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "not have permission",
				})
				return
			}

			res, err := arango.FindMetadataByBid(accessKey.BucketId, limit, offset, false)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at files/all:",
					"Db Error")
				return
			}

			c.JSON(http.StatusOK, res)
		})

		acr.GET("/hidden/all", func(c *gin.Context) {
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

				_ = nats.SendErrorEvent("accessKey not found in authenticate at /files/hidden/all:",
					"Unknown Error")
				return
			}
			accessKey := key.(*arango.AccessKey)
			var isGetFileListHiddenPerm bool
			for _, perm := range accessKey.Permissions {
				if perm == "GetFileListHidden" {
					isGetFileListHiddenPerm = true
					break
				}
			}

			if !isGetFileListHiddenPerm {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "not have permission",
				})
				return
			}

			res, err := arango.FindMetadataByBid(accessKey.BucketId, limit, offset, true)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at authenticated files/hidden/all:",
					"Db Error")
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

				_ = nats.SendErrorEvent("accessKey not found in authenticate at /files/upload:",
					"Unknown Error")
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

			bucket, err := arango.FindBucketById(accessKey.BucketId)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
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
			queryPath := c.DefaultPostForm("path", "/")
			path := ultis.StandardizedPath("/"+bucket.Name+"/"+queryPath, true)

			fileName := c.DefaultPostForm("name", uploadFile.Filename)
			//newPath := bucket.Name + path + fileName

			fileContent, err := uploadFile.Open()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("open file failed at /files/upload:",
					"File Error")
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
				cType, fileSize, time.Duration(ttl)*time.Second)
			if err != nil {
				if e, ok := err.(*models.ModelError); ok {
					if e.ErrType == models.Duplicated {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": err.Error(),
						})

						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent("db error at /files/upload: "+err.Error(),
					"File Error")
				return
			}

			//LOG
			_ = nats.SendUploadFileEvent(*res)

			c.JSON(http.StatusOK, res)
		})

		acr.GET("/download", func(c *gin.Context) {
			key, ok := c.Get("accessKey")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("accessKey not found in authenticate at /files/download:",
					"Unknown Error")
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
			fid := c.DefaultQuery("fileId", "")

			fileMeta, err := arango.FindMetadataById(fid)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "file not found",
				})

				return
			}

			if fileMeta.IsHidden {
				var isDownloadHiddenPerm bool
				for _, perm := range accessKey.Permissions {
					if perm == "DownloadHidden" {
						isDownloadHiddenPerm = true
						break
					}
				}
				if !isDownloadHiddenPerm {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "file not found",
					})

					return
				}
			}

			err = arango.GetFileByFidIgnoreQueryMetadata(fileMeta.FileId, func(reader io.Reader) error {
				if fileMeta.BucketId != accessKey.BucketId {
					return &models.RouteError{
						Msg:     "invalid bucket",
						ErrType: models.InvalidBucket,
					}
				}

				extraHeaders := map[string]string{
					"Content-Disposition": `attachment; filename=` + fileMeta.Name,
				}

				c.DataFromReader(http.StatusOK, fileMeta.Size, fileMeta.ContentType, reader, extraHeaders)

				//LOG
				_ = nats.SendDownloadFileEvent(*fileMeta)

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

				if e, ok := err.(*models.ModelError); ok {
					if e.ErrType == models.DocumentNotFound {
						c.JSON(http.StatusNotFound, gin.H{
							"error": "file not found",
						})

						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("download failed: "+err.Error()+" at /files/download:",
					"File Error")
				return
			}
		})

		acr.GET("/download/*fullpath", func(c *gin.Context) {
			fullpath := c.Param("fullpath")
			fullpath = ultis.StandardizedPath(fullpath, true)
			bucketName := ultis.GetBucketName(fullpath)
			parentPath := ultis.GetParentPath(fullpath)
			fileName := ultis.GetFileName(fullpath)

			key, ok := c.Get("accessKey")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("accessKey not found in authenticate at /files/upload:",
					"Unknown Error")
				return
			}
			accessKey := key.(*arango.AccessKey)

			bucket, err := arango.FindBucketById(accessKey.BucketId)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "bucket not found",
				})

				return
			}

			if bucket.Name != bucketName {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid bucket name",
				})

				return
			}

			fileMeta, err := arango.FindMetadataByFilename(parentPath, fileName, bucket.Id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "file not found",
				})

				return
			}

			if fileMeta.IsHidden {
				var isDownloadHiddenPerm bool
				for _, perm := range accessKey.Permissions {
					if perm == "DownloadHidden" {
						isDownloadHiddenPerm = true
						break
					}
				}
				if !isDownloadHiddenPerm {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "file not found",
					})

					return
				}
			}

			err = arango.GetFileByFidIgnoreQueryMetadata(fileMeta.FileId, func(reader io.Reader) error {
				if fileMeta.BucketId != accessKey.BucketId {
					return &models.RouteError{
						Msg:     "invalid bucket",
						ErrType: models.InvalidBucket,
					}
				}

				extraHeaders := map[string]string{
					"Content-Disposition": `attachment; filename=` + fileMeta.Name,
				}

				c.DataFromReader(http.StatusOK, fileMeta.Size, fileMeta.ContentType, reader, extraHeaders)

				//LOG
				_ = nats.SendDownloadFileEvent(*fileMeta)

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

				if e, ok := err.(*models.ModelError); ok {
					if e.ErrType == models.DocumentNotFound {
						c.JSON(http.StatusNotFound, gin.H{
							"error": "file not found",
						})

						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("download failed: "+err.Error()+" at /files/download:",
					"File Error")
				return
			}
		})

		acr.POST("/hidden", func(c *gin.Context) {
			qIsHidden := c.DefaultQuery("hidden", "false")
			qName := c.DefaultQuery("name", "")
			qPath := c.DefaultQuery("path", "")

			key, ok := c.Get("accessKey")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("accessKey not found in authenticate at /accessKey/files/hidden:",
					"Unknown Error")
				return
			}
			accessKey := key.(*arango.AccessKey)

			var isMarkHiddenPerm bool
			for _, perm := range accessKey.Permissions {
				if perm == "MarkHidden" {
					isMarkHiddenPerm = true
					break
				}
			}

			if !isMarkHiddenPerm {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "not have permission",
				})
				return
			}

			fm, err := arango.FindMetadataByFilename(qPath, qName, accessKey.BucketId)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("find file failed at /accessKey/files/hidden:",
					"File Error")
				return
			}

			if accessKey.BucketId != fm.BucketId {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "permission denied",
				})
				return
			}

			isHidden, err := strconv.ParseBool(qIsHidden)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("parse failed at /accessKey/files/hidden:",
					"File Error")
				return
			}
			file, err := arango.ToggleHidden(fm.Path, isHidden)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("toggle failed at /accessKey/files/hidden:",
					"File Error")
				return
			}

			c.JSON(http.StatusOK, file)
		})
	}

	ar := r.Group("/auth/files", middlewares.UserAuthenticate)
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
			bid := c.DefaultQuery("bucketId", "")
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

						_ = nats.SendErrorEvent(err.Error()+" at authenticated files/auth/all:",
							"Db Error")
						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at authenticated files/auth/all:",
					"Db Error")
				return
			}

			if uid, ok := c.Get("uid"); !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent("uid not found in authenticate at /files/auth/all",
					"Unknown Error")
				return
			} else {
				if uid.(string) != bucket.Uid {
					c.JSON(http.StatusForbidden, gin.H{
						"error": "permission denied",
					})
					return
				}
			}

			res, err := arango.FindMetadataByBid(bid, limit, offset, false)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at authenticated files/auth/all:",
					"Db Error")
				return
			}

			c.JSON(http.StatusOK, res)
		})

		ar.POST("/upload", func(c *gin.Context) {
			bid := c.DefaultPostForm("bucket_id", "")
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

						_ = nats.SendErrorEvent(err.Error()+" at authenticated files/auth/upload:",
							"Db Error")
						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at authenticated files/auth/upload:",
					"Db Error")
				return
			}

			if uid, ok := c.Get("uid"); !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at authenticated files/auth/upload:",
					"Unknown Error")
				return
			} else {
				if uid.(string) != bucket.Uid {
					c.JSON(http.StatusForbidden, gin.H{
						"error": "permission denied",
					})
					return
				}
			}

			uploadFile, err := c.FormFile("file")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}
			queryPath := c.DefaultPostForm("path", "/")
			path := ultis.StandardizedPath(queryPath, true)

			fileName := c.DefaultPostForm("name", uploadFile.Filename)
			//newPath := bucket.Name + path + fileName

			fileContent, err := uploadFile.Open()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("open file failed at /files/auth/upload:",
					"File Error")
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

			res, err := arango.SaveFile(fileContent, bid, path, fileName, isHidden,
				cType, fileSize, time.Duration(ttl))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			//LOG
			_ = nats.SendUploadFileEvent(*res)

			c.JSON(http.StatusOK, res)
		})

		ar.GET("/download", func(c *gin.Context) {
			fid := c.DefaultQuery("fileId", "")
			bid := c.DefaultQuery("bucketId", "")

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

						_ = nats.SendErrorEvent(err.Error()+" at authenticated files/auth/download",
							"Db Error")
						return
					}
				}
			}

			if uid, ok := c.Get("uid"); !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent("uid not found at authenticated files/auth/download",
					"Unknown Error")
				return
			} else {
				if uid.(string) != bucket.Uid {
					c.JSON(http.StatusForbidden, gin.H{
						"error": "permission denied",
					})
					return
				}
			}

			err = arango.GetFileByFid(fid, func(reader io.Reader, metadata *arango.FileMetadata) error {
				if metadata.BucketId != bid {
					return &models.RouteError{
						Msg:     "invalid bucket",
						ErrType: models.InvalidBucket,
					}
				}

				extraHeaders := map[string]string{
					"Content-Disposition": `attachment; filename=` + metadata.Name,
				}

				c.DataFromReader(http.StatusOK, metadata.Size, metadata.ContentType, reader, extraHeaders)

				//LOG
				_ = nats.SendDownloadFileEvent(*metadata)

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

				_ = nats.SendErrorEvent(err.Error()+" at /files/auth/download:",
					"File Error")
				return
			}
		})

		ar.GET("/downloadFiles", func(c *gin.Context) {
			type fileList struct {
				FileIds  []string `json:"file_ids"`
				BucketId string   `json:"bucket_id"`
			}

			var curFileList fileList
			if err := c.ShouldBind(&curFileList); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
			}

			bucket, err := arango.FindBucketById(curFileList.BucketId)
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

						_ = nats.SendErrorEvent(err.Error()+" at authenticated /auth/files/download",
							"Db Error")
						return
					}
				}
			}

			if uid, ok := c.Get("uid"); !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent("uid not found at authenticated /auth/files/downloadFiles",
					"Unknown Error")
				return
			} else {
				if uid.(string) != bucket.Uid {
					c.JSON(http.StatusForbidden, gin.H{
						"error": "permission denied",
					})
					return
				}
			}

			listFileMetadata := []arango.FileMetadata{}
			for _, fid := range curFileList.FileIds {
				fm, err := arango.FindMetadataByFid(fid)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": err.Error(),
					})
					return
				}
				listFileMetadata = append(listFileMetadata, *fm)
			}

			validListFileMetadata := []arango.FileMetadata{}
			for _, fm := range listFileMetadata {
				if fm.BucketId == curFileList.BucketId {
					validListFileMetadata = append(validListFileMetadata, fm)
				}
			}

			var totalSize int64
			for _, fm := range validListFileMetadata {
				totalSize += fm.Size
			}

			const TotalSizeLimit = 10 << (10 * 3)
			if totalSize > TotalSizeLimit {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Total File Size Over Limit (10GB)",
				})
				return
			}

			c.Writer.Header().Set("Content-type", "application/octet-stream")
			zipw := zip.NewWriter(c.Writer)
			defer zipw.Close()

			for _, fm := range validListFileMetadata {
				err := arango.GetFileByFidIgnoreQueryMetadata(fm.FileId, func(reader io.Reader) error {
					if err = appendReaderToZip(reader, fm.Name, zipw); err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": err.Error(),
					})
					return
				}
			}
		})

		ar.POST("/hidden", func(c *gin.Context) {
			qIsHidden := c.DefaultQuery("hidden", "false")
			qName := c.DefaultQuery("name", "")
			qPath := c.DefaultQuery("path", "")
			qBid := c.DefaultQuery("bucketId", "")

			fm, err := arango.FindMetadataByFilename(qPath, qName, qBid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("find file failed at auth/files/toggle/hidden:",
					"File Error")
				return
			}

			bucket, err := arango.FindBucketById(fm.BucketId)
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

						_ = nats.SendErrorEvent(err.Error()+" at authenticated files/auth/upload:",
							"Db Error")
						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at authenticated files/auth/upload:",
					"Db Error")
				return
			}

			if uid, ok := c.Get("uid"); !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at authenticated files/auth/upload:",
					"Unknown Error")
				return
			} else {
				if uid.(string) != bucket.Uid {
					c.JSON(http.StatusForbidden, gin.H{
						"error": "permission denied",
					})
					return
				}
			}

			isHidden, err := strconv.ParseBool(qIsHidden)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("parse failed at auth/files/toggle/hidden:",
					"File Error")
				return
			}
			file, err := arango.ToggleHidden(fm.Path, isHidden)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("toggle failed at auth/files/toggle/hidden:",
					"File Error")
				return
			}

			c.JSON(http.StatusOK, file)
		})
	}

	kpr := r.Group("/signed/files", middlewares.CheckSigned)
	{
		kpr.GET("/all", func(c *gin.Context) {
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

			key, ok := c.Get("keyPair")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("keyPair not found in authenticate at /signed/files/all:",
					"Unknown Error")
				return
			}
			keyPair := key.(*arango.KeyPair)

			var isUploadPerm bool
			for _, perm := range keyPair.Permissions {
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

			res, err := arango.FindMetadataByBid(keyPair.BucketId, limit, offset, false)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at /signed/files/all:",
					"Db Error")
				return
			}

			c.JSON(http.StatusOK, res)
		})

		kpr.GET("/hidden/all", func(c *gin.Context) {
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

			key, ok := c.Get("keyPair")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("keyPair not found in authenticate at /signed/files/all:",
					"Unknown Error")
				return
			}
			keyPair := key.(*arango.KeyPair)

			var isUploadPerm bool
			for _, perm := range keyPair.Permissions {
				if perm == "GetFileListHidden" {
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

			res, err := arango.FindMetadataByBid(keyPair.BucketId, limit, offset, true)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at authenticated files/hidden/all:",
					"Db Error")
				return
			}

			c.JSON(http.StatusOK, res)
		})

		kpr.POST("/upload", func(c *gin.Context) {
			key, ok := c.Get("keyPair")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("keyPair not found in authenticate at /signed/files/upload:",
					"Unknown Error")
				return
			}

			keyPair := key.(*arango.KeyPair)
			var isUploadPerm bool
			for _, perm := range keyPair.Permissions {
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

			bucket, err := arango.FindBucketById(keyPair.BucketId)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
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
			queryPath := c.DefaultPostForm("path", "/")
			path := ultis.StandardizedPath("/"+bucket.Name+"/"+queryPath, true)

			fileName := c.DefaultPostForm("name", uploadFile.Filename)
			//newPath := bucket.Name + path + fileName

			fileContent, err := uploadFile.Open()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("open file failed at /signed/files/upload:",
					"File Error")
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

			res, err := arango.SaveFile(fileContent, keyPair.BucketId, path, fileName, isHidden,
				cType, fileSize, time.Duration(ttl)*time.Second)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			//LOG
			_ = nats.SendUploadFileEvent(*res)

			c.JSON(http.StatusOK, res)
		})

		kpr.GET("/download", func(c *gin.Context) {
			key, ok := c.Get("keyPair")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("keyPair not found in authenticate at signed/files/download:",
					"Unknown Error")
				return
			}
			keyPair := key.(*arango.KeyPair)
			var isDownloadPerm bool
			for _, perm := range keyPair.Permissions {
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
			fid := c.DefaultQuery("fileId", "")

			fileMeta, err := arango.FindMetadataById(fid)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "file not found",
				})

				return
			}

			if fileMeta.IsHidden {
				var isDownloadHiddenPerm bool
				for _, perm := range keyPair.Permissions {
					if perm == "DownloadHidden" {
						isDownloadHiddenPerm = true
						break
					}
				}
				if !isDownloadHiddenPerm {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "file not found",
					})

					return
				}
			}

			err = arango.GetFileByFidIgnoreQueryMetadata(fileMeta.FileId, func(reader io.Reader) error {
				if fileMeta.BucketId != keyPair.BucketId {
					return &models.RouteError{
						Msg:     "invalid bucket",
						ErrType: models.InvalidBucket,
					}
				}

				extraHeaders := map[string]string{
					"Content-Disposition": `attachment; filename=` + fileMeta.Name,
				}

				c.DataFromReader(http.StatusOK, fileMeta.Size, fileMeta.ContentType, reader, extraHeaders)

				//LOG
				_ = nats.SendDownloadFileEvent(*fileMeta)

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

				_ = nats.SendErrorEvent("download failed: "+err.Error()+" at signed/files/download:",
					"File Error")
				return
			}
		})

		kpr.GET("/download/*fullpath", func(c *gin.Context) {
			fullpath := c.Param("fullpath")
			fullpath = ultis.StandardizedPath(fullpath, true)
			bucketName := ultis.GetBucketName(fullpath)
			parentPath := ultis.GetParentPath(fullpath)
			fileName := ultis.GetFileName(fullpath)

			key, ok := c.Get("keyPair")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("accessKey not found in authenticate at /files/upload:",
					"Unknown Error")
				return
			}
			keyPair := key.(*arango.KeyPair)

			bucket, err := arango.FindBucketById(keyPair.BucketId)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "bucket not found",
				})

				return
			}

			if bucket.Name != bucketName {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid bucket name",
				})

				return
			}

			fileMeta, err := arango.FindMetadataByFilename(parentPath, fileName, bucket.Id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "file not found",
				})

				return
			}

			if fileMeta.IsHidden {
				var isDownloadHiddenPerm bool
				for _, perm := range keyPair.Permissions {
					if perm == "DownloadHidden" {
						isDownloadHiddenPerm = true
						break
					}
				}
				if !isDownloadHiddenPerm {
					c.JSON(http.StatusNotFound, gin.H{
						"error": "file not found",
					})

					return
				}
			}

			err = arango.GetFileByFidIgnoreQueryMetadata(fileMeta.FileId, func(reader io.Reader) error {
				if fileMeta.BucketId != keyPair.BucketId {
					return &models.RouteError{
						Msg:     "invalid bucket",
						ErrType: models.InvalidBucket,
					}
				}

				extraHeaders := map[string]string{
					"Content-Disposition": `attachment; filename=` + fileMeta.Name,
				}

				c.DataFromReader(http.StatusOK, fileMeta.Size, fileMeta.ContentType, reader, extraHeaders)

				//LOG
				_ = nats.SendDownloadFileEvent(*fileMeta)

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

				if e, ok := err.(*models.ModelError); ok {
					if e.ErrType == models.DocumentNotFound {
						c.JSON(http.StatusNotFound, gin.H{
							"error": "file not found",
						})

						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("download failed: "+err.Error()+" at /files/download:",
					"File Error")
				return
			}
		})

		kpr.POST("/hidden", func(c *gin.Context) {
			qIsHidden := c.DefaultQuery("hidden", "false")
			qName := c.DefaultQuery("name", "")
			qPath := c.DefaultQuery("path", "")

			key, ok := c.Get("keyPair")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("keyPair not found in authenticate at /signed/files/all:",
					"Unknown Error")
				return
			}
			keyPair := key.(*arango.KeyPair)

			var isMarkHiddenPerm bool
			for _, perm := range keyPair.Permissions {
				if perm == "MarkHidden" {
					isMarkHiddenPerm = true
					break
				}
			}

			if !isMarkHiddenPerm {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "not have permission",
				})
				return
			}

			fm, err := arango.FindMetadataByFilename(qPath, qName, keyPair.BucketId)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("find file failed at /signed/files/hidden:",
					"File Error")
				return
			}

			if keyPair.BucketId != fm.BucketId {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "permission denied",
				})
				return
			}

			isHidden, err := strconv.ParseBool(qIsHidden)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("parse failed at /signed/files/hidden:",
					"File Error")
				return
			}
			file, err := arango.ToggleHidden(fm.Path, isHidden)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("toggle failed at /signed/files/hidden:",
					"File Error")
				return
			}

			c.JSON(http.StatusOK, file)
		})
	}
}

func appendReaderToZip(fileReader io.Reader, filename string, zipw *zip.Writer) error {
	wr, err := zipw.Create(filename)
	if err != nil {
		return err
	}
	if _, err = io.Copy(wr, fileReader); err != nil {
		return err
	}

	return nil
}
