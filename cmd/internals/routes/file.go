package routes

import (
	"archive/zip"
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/blend/go-sdk/crypto"
	"github.com/gin-gonic/gin"
	"github.com/minio/sio"
	"io"
	"net/http"
	"strconv"
	"time"
)

func FileRoutes(r *gin.Engine) {
	ar := r.Group("/auth/files", middlewares.UserAuthenticate)
	{
		ar.GET("/all", middlewares.ReqLogger("auth", "B"), func(c *gin.Context) {
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

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			if uid, ok := c.Get("uid"); !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err := nats.SendErrorEvent("uid not found at /auth/files/all",
					"Unknown Error")
				print(err)
				return
			} else {
				if uid.(string) != bucket.Uid {
					c.JSON(http.StatusForbidden, gin.H{
						"error": "permission denied",
					})
					return
				}
			}

			res, err := arango.FindMetadataByBid(bid, limit, offset, true)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			c.JSON(http.StatusOK, res)
		})

		ar.POST("/upload", middlewares.ReqLogger("auth", "A"), func(c *gin.Context) {
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

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			if uid, ok := c.Get("uid"); !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err := nats.SendErrorEvent("uid not found at /auth/files/upload",
					"Unknown Error")
				print(err)
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
			path := ultis.StandardizedPath(bucket.Name+"/"+queryPath, true)

			fileName := c.DefaultPostForm("name", uploadFile.Filename)
			//newPath := bucket.Name + path + fileName

			if ok, err := ultis.ValidateFileName(fileName); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Validate Error")
				return
			} else if !ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "File should not contain special characters, from 1-255 characters",
				})

				return
			}

			fileContent, err := uploadFile.Open()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "File Error")
				return
			}

			fileSize := uploadFile.Size
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

			var holdDuration time.Duration
			if bucket.IsObjectLock {
				holdDuration = bucket.HoldDuration
			} else {
				holdDuration = 0
			}

			var encryptionInfo *arango.EncryptionInfo
			var res *arango.FileMetadata
			var isEncrypted bool
			var r io.Reader

			if bucket.IsEncrypted {
				encryptionInfo, err = arango.FindLatestEncryptionInfoByBucketId(bucket.Id)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "something went wrong",
					})

					err = nats.SendErrorEvent(err.Error(), "Db Error")
					return
				}

				if encryptionInfo.To == nil || encryptionInfo.To.After(time.Now()) {
					isEncrypted = true
					r, err = ultis.EncryptReader(fileContent, encryptionInfo.Passphrase)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{
							"error": "something went wrong",
						})

						err = nats.SendErrorEvent(err.Error(), "Encrypt Error")
						return
					}

				} else {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "something went wrong",
					})

					return
				}
			} else {
				isEncrypted = false
				r = fileContent
			}

			res, err = arango.SaveFile(r, bid, bucket.Uid, path, fileName, isHidden,
				cType, fileSize, isEncrypted, holdDuration)
			if err != nil {
				if err, ok := err.(*models.ModelError); ok {
					if err.ErrType == models.NotFound {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": err.Msg,
						})
					}

					if err.ErrType == models.Duplicated {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": err.Msg,
						})
					}
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": err.Error(),
					})
				}
				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			if isEncrypted {
				er := r.(*crypto.StreamEncrypter)
				meta := er.Meta()
				res, err = arango.UpdateFileEncryptData(res.Id, meta.IV, meta.Hash)
				if err != nil {
					if err, ok := err.(*models.ModelError); ok {
						if err.ErrType == models.NotFound {
							c.JSON(http.StatusBadRequest, gin.H{
								"error": err.Msg,
							})
						}
					} else {
						c.JSON(http.StatusInternalServerError, gin.H{
							"error": err.Error(),
						})
					}
					return
				}
			}

			//LOG
			_ = nats.SendUploadFileEvent(res.Id, res.FileId, res.Name, res.Size,
				res.BucketId, res.UploadedDate, bucket.Uid)

			c.JSON(http.StatusOK, res)
		})

		ar.GET("/detail/*fullpath", middlewares.ReqLogger("auth", "B"), func(c *gin.Context) {
			fullpath := c.Param("fullpath")
			fullpath = ultis.StandardizedPath(fullpath, true)
			bucketName := ultis.GetBucketName(fullpath)
			parentPath := ultis.GetParentPath(fullpath)
			fileName := ultis.GetFileName(fullpath)

			bucket, err := arango.FindBucketByName(bucketName)
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

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			var userId string
			if uid, ok := c.Get("uid"); !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err := nats.SendErrorEvent("uid not found at /auth/files/detail/*fullpath",
					"Unknown Error")
				print(err)
				return
			} else {
				userId = uid.(string)
				if userId != bucket.Uid {
					c.JSON(http.StatusForbidden, gin.H{
						"error": "permission denied",
					})
					return
				}
			}

			fileMeta, err := arango.FindMetadataByFilename(parentPath, fileName, bucket.Id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "file not found",
				})

				return
			}

			c.JSON(http.StatusOK, fileMeta)
		})

		ar.GET("/download", middlewares.ReqLogger("auth", "B"), func(c *gin.Context) {
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

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}
			}

			var userId string
			if uid, ok := c.Get("uid"); !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err := nats.SendErrorEvent("key not found at /auth/files/download",
					"Unknown Error")
				print(err)
				return
			} else {
				userId = uid.(string)
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

				var encryptInfo *arango.EncryptionInfo
				var r io.Reader
				if metadata.IsEncrypted {
					encryptInfo, err = arango.FindEncryptionInfoInDate(bucket.Id, metadata.UploadedDate)
					if err != nil {
						return err
					}

					r, err = ultis.DecryptReader(reader, crypto.StreamMeta{
						IV:   metadata.EncryptData.IV,
						Hash: metadata.EncryptData.Hash,
					}, encryptInfo.Passphrase)
					if err != nil {
						return err
					}
				} else {
					r = reader
				}

				extraHeaders := map[string]string{}

				teeReader := io.TeeReader(r, &ultis.DownloadBandwidthLogger{
					Uid:        userId,
					From:       userId,
					BucketId:   bucket.Id,
					SourceType: "auth",
				})

				c.DataFromReader(http.StatusOK, metadata.Size, metadata.ContentType, teeReader, extraHeaders)

				//LOG
				_ = nats.SendDownloadFileEvent(metadata.Id, metadata.FileId, metadata.Name, metadata.Size,
					metadata.BucketId, metadata.UploadedDate, userId)

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

				err = nats.SendErrorEvent(err.Error()+" at /files/auth/download:",
					"File Error")
				return
			}
		})

		ar.GET("/download/*fullpath", middlewares.ReqLogger("auth", "B"), func(c *gin.Context) {
			fullpath := c.Param("fullpath")
			fullpath = ultis.StandardizedPath(fullpath, true)
			bucketName := ultis.GetBucketName(fullpath)
			parentPath := ultis.GetParentPath(fullpath)
			fileName := ultis.GetFileName(fullpath)

			bucket, err := arango.FindBucketByName(bucketName)
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

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			var userId string
			if uid, ok := c.Get("uid"); !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err := nats.SendErrorEvent("uid not found at /auth/files/download/*fullpath",
					"Unknown Error")
				print(err)
				return
			} else {
				userId = uid.(string)
				if userId != bucket.Uid {
					c.JSON(http.StatusForbidden, gin.H{
						"error": "permission denied",
					})
					return
				}
			}

			fileMeta, err := arango.FindMetadataByFilename(parentPath, fileName, bucket.Id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "file not found",
				})

				return
			}

			err = arango.GetFileByFidIgnoreQueryMetadata(fileMeta.FileId, func(reader io.Reader) error {
				if fileMeta.BucketId != bucket.Id {
					return &models.RouteError{
						Msg:     "invalid bucket",
						ErrType: models.InvalidBucket,
					}
				}

				var encryptInfo *arango.EncryptionInfo
				var r io.Reader
				if fileMeta.IsEncrypted {
					encryptInfo, err = arango.FindEncryptionInfoInDate(bucket.Id, fileMeta.UploadedDate)
					if err != nil {
						return err
					}

					r, err = ultis.DecryptReader(reader, crypto.StreamMeta{
						IV:   fileMeta.EncryptData.IV,
						Hash: fileMeta.EncryptData.Hash,
					}, encryptInfo.Passphrase)
					if err != nil {
						return err
					}
				} else {
					r = reader
				}

				extraHeaders := map[string]string{}

				teeReader := io.TeeReader(r, &ultis.DownloadBandwidthLogger{
					Uid:        userId,
					From:       userId,
					BucketId:   bucket.Id,
					SourceType: "auth",
				})

				c.DataFromReader(http.StatusOK, fileMeta.Size, fileMeta.ContentType, teeReader, extraHeaders)

				//LOG
				_ = nats.SendDownloadFileEvent(fileMeta.Id, fileMeta.FileId, fileMeta.Name, fileMeta.Size,
					fileMeta.BucketId, fileMeta.UploadedDate, userId)

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

				err = nats.SendErrorEvent("download failed: "+err.Error()+" at auth/files/download/*fullpath",
					"File Error")
				return
			}
		})

		//ar.GET("/downloadFiles", func(c *gin.Context) {
		//	type fileList struct {
		//		FileIds  []string `json:"file_ids"`
		//		BucketId string   `json:"bucket_id"`
		//	}
		//
		//	var curFileList fileList
		//	if err := c.ShouldBind(&curFileList); err != nil {
		//		c.JSON(http.StatusInternalServerError, gin.H{
		//			"error": err.Error(),
		//		})
		//	}
		//
		//	bucket, err := arango.FindBucketById(curFileList.BucketId)
		//	if err != nil {
		//		if e, ok := err.(*models.ModelError); ok {
		//			if e.ErrType == models.DocumentNotFound {
		//				c.JSON(http.StatusBadRequest, gin.H{
		//					"error": "bid invalid",
		//				})
		//
		//				return
		//			}
		//			if e.ErrType == models.DbError {
		//				c.JSON(http.StatusInternalServerError, gin.H{
		//					"error": "something when wrong",
		//				})
		//
		//				_ = nats.SendErrorEvent(err.Error()+" at authenticated /auth/files/download",
		//					"Db Error")
		//				return
		//			}
		//		}
		//	}
		//
		//	if uid, ok := c.Get("uid"); !ok {
		//		c.JSON(http.StatusInternalServerError, gin.H{
		//			"error": "something when wrong",
		//		})
		//
		//		_ = nats.SendErrorEvent("uid not found at authenticated /auth/files/downloadFiles",
		//			"Unknown Error")
		//		return
		//	} else {
		//		if uid.(string) != bucket.Uid {
		//			c.JSON(http.StatusForbidden, gin.H{
		//				"error": "permission denied",
		//			})
		//			return
		//		}
		//	}
		//
		//	listFileMetadata := []arango.FileMetadata{}
		//	for _, fid := range curFileList.FileIds {
		//		fm, err := arango.FindMetadataByFid(fid)
		//		if err != nil {
		//			c.JSON(http.StatusInternalServerError, gin.H{
		//				"error": err.Error(),
		//			})
		//			return
		//		}
		//		listFileMetadata = append(listFileMetadata, *fm)
		//	}
		//
		//	validListFileMetadata := []arango.FileMetadata{}
		//	for _, fm := range listFileMetadata {
		//		if fm.BucketId == curFileList.BucketId {
		//			validListFileMetadata = append(validListFileMetadata, fm)
		//		}
		//	}
		//
		//	var totalSize int64
		//	for _, fm := range validListFileMetadata {
		//		totalSize += fm.Size
		//	}
		//
		//	const TotalSizeLimit = 10 << (10 * 3)
		//	if totalSize > TotalSizeLimit {
		//		c.JSON(http.StatusBadRequest, gin.H{
		//			"error": "Total File Size Over Limit (10GB)",
		//		})
		//		return
		//	}
		//
		//	c.Writer.Header().Set("Content-type", "application/octet-stream")
		//	zipw := zip.NewWriter(c.Writer)
		//	defer zipw.Close()
		//
		//	for _, fm := range validListFileMetadata {
		//		err := arango.GetFileByFidIgnoreQueryMetadata(fm.FileId, func(reader io.Reader) error {
		//			if err = appendReaderToZip(reader, fm.Name, zipw); err != nil {
		//				return err
		//			}
		//			return nil
		//		})
		//		if err != nil {
		//			c.JSON(http.StatusInternalServerError, gin.H{
		//				"error": err.Error(),
		//			})
		//			return
		//		}
		//	}
		//})

		ar.POST("/hidden", middlewares.ReqLogger("auth", "A"), func(c *gin.Context) {
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

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			if uid, ok := c.Get("uid"); !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err := nats.SendErrorEvent("key not found at auth/files/hidden",
					"Unknown Error")
				print(err)
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

				err = nats.SendErrorEvent("parse failed at auth/files/toggle/hidden:",
					"File Error")
				return
			}
			file, err := arango.ToggleHidden(fm.Path, isHidden)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent("toggle failed at auth/files/toggle/hidden:",
					"File Error")
				return
			}

			c.JSON(http.StatusOK, file)
		})

		ar.DELETE("/*fullpath", middlewares.ReqLogger("auth", "A"), func(c *gin.Context) {
			fullpath := c.Param("fullpath")
			fullpath = ultis.StandardizedPath(fullpath, true)
			bucketName := ultis.GetBucketName(fullpath)
			parentPath := ultis.GetParentPath(fullpath)
			fileName := ultis.GetFileName(fullpath)

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

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}
			}

			if uid, ok := c.Get("uid"); !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err := nats.SendErrorEvent("key not found at delete auth/files/*full_path",
					"Unknown Error")
				print(err)
				return
			} else {
				if uid.(string) != bucket.Uid {
					c.JSON(http.StatusForbidden, gin.H{
						"error": "permission denied",
					})
					return
				}
			}

			if bucket.Name != bucketName {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid bucket name",
				})

				return
			}

			err = arango.MarkDeleteFile(parentPath, fileName, bucket.Id)
			if err != nil {
				if e, ok := err.(*models.ModelError); ok {
					if e.ErrType == models.DocumentNotFound {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": "file not found",
						})

						return
					}
					if e.ErrType == models.Locked {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": e.Msg,
						})

						return
					}
					if e.ErrType == models.DbError {
						c.JSON(http.StatusInternalServerError, gin.H{
							"error": "something when wrong",
						})

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "successfully deleted",
			})
		})
	}

	kr := r.Group("/accessKey/files", middlewares.AccessKeyAuthenticate)
	{
		kr.GET("/", middlewares.ReqLogger("key", "B"), func(c *gin.Context) {
			k, ok := c.Get("key")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err := nats.SendErrorEvent("key not found at get auth/files/accessKey/files",
					"Unknown Error")
				print(err)
				return
			}

			key := k.(*arango.AccessKey)
			hasPerm, err := CheckPerm(key, arango.ListFiles)
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

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			if !ultis.CheckBucketPerm(key.BucketId, bucket.Id) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "this key is not associated with this bucket",
				})

				return
			}

			if uid, ok := c.Get("uid"); !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err := nats.SendErrorEvent("uid not found at get auth/files/accessKey/files",
					"Unknown Error")
				print(err)
				return
			} else {
				if uid.(string) != bucket.Uid {
					c.JSON(http.StatusForbidden, gin.H{
						"error": "permission denied",
					})
					return
				}
			}

			res, err := arango.FindMetadataByBid(bid, limit, offset, true)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			c.JSON(http.StatusOK, res)
		})

		kr.POST("/upload", middlewares.ReqLogger("key", "A"), func(c *gin.Context) {
			k, ok := c.Get("key")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err := nats.SendErrorEvent("key not found at post /accessKey/files/upload",
					"Unknown Error")
				print(err)
				return
			}

			key := k.(*arango.AccessKey)
			hasPerm, err := CheckPerm(key, arango.WriteFiles)
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

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			if !ultis.CheckBucketPerm(key.BucketId, bucket.Id) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "this key is not associated with this bucket",
				})

				return
			}

			//if uid, ok := c.Get("uid"); !ok {
			//	c.JSON(http.StatusInternalServerError, gin.H{
			//		"error": "something when wrong",
			//	})
			//
			//	//_ = nats.SendErrorEvent(err.Error()+" at authenticated files/auth/upload:",
			//	//	"Unknown Error")
			//	return
			//} else {
			if key.Uid != bucket.Uid {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "permission denied",
				})
				return
			}
			//}

			uploadFile, err := c.FormFile("file")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}
			queryPath := c.DefaultPostForm("path", "/")
			path := ultis.StandardizedPath(bucket.Name+"/"+queryPath, true)

			fileName := c.DefaultPostForm("name", uploadFile.Filename)

			if ok, err := ultis.ValidateFileName(fileName); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Validate Error")
				return
			} else if !ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "File should not contain special characters, from 1-255 characters",
				})

				return
			}

			fileContent, err := uploadFile.Open()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent("open file failed at accessKey/files/upload",
					"File Error")
				return
			}

			fileSize := uploadFile.Size
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

			var holdDuration time.Duration
			if bucket.IsObjectLock {
				holdDuration = bucket.HoldDuration
			} else {
				holdDuration = 0
			}

			var encryptionInfo *arango.EncryptionInfo
			var res *arango.FileMetadata
			var isEncrypted bool
			var r io.Reader

			if bucket.IsEncrypted {
				encryptionInfo, err = arango.FindLatestEncryptionInfoByBucketId(bucket.Id)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "something went wrong",
					})

					err = nats.SendErrorEvent(err.Error(), "Db Error")
					return
				}

				if encryptionInfo.To == nil || encryptionInfo.To.After(time.Now()) {
					isEncrypted = true
					r, err = ultis.EncryptReader(fileContent, encryptionInfo.Passphrase)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{
							"error": "something went wrong",
						})

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "something went wrong",
					})

					return
				}
			} else {
				isEncrypted = false
				r = fileContent
			}

			res, err = arango.SaveFile(r, bid, key.Uid, path, fileName, isHidden,
				cType, fileSize, isEncrypted, holdDuration)
			if err != nil {
				if err, ok := err.(*models.ModelError); ok {
					if err.ErrType == models.NotFound {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": err.Msg,
						})
					}
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": err.Error(),
					})
				}
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			if isEncrypted {
				er := r.(*crypto.StreamEncrypter)
				meta := er.Meta()
				res, err = arango.UpdateFileEncryptData(res.Id, meta.IV, meta.Hash)
				if err != nil {
					if err, ok := err.(*models.ModelError); ok {
						if err.ErrType == models.NotFound {
							c.JSON(http.StatusBadRequest, gin.H{
								"error": err.Msg,
							})
						}
					} else {
						c.JSON(http.StatusInternalServerError, gin.H{
							"error": err.Error(),
						})
					}
					err = nats.SendErrorEvent(err.Error(), "Db Error")
					return
				}
			}

			//LOG
			_ = nats.SendUploadFileEvent(res.Id, res.FileId, res.Name, res.Size,
				res.BucketId, res.UploadedDate, key.Uid)

			c.JSON(http.StatusOK, res)
		})

		kr.DELETE("/*fullpath", middlewares.ReqLogger("key", "A"), func(c *gin.Context) {
			k, ok := c.Get("key")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err := nats.SendErrorEvent("key not found at delete /accessKey/files/*fullpath",
					"Unknown Error")
				print(err)
				return
			}

			key := k.(*arango.AccessKey)
			hasPerm, err := CheckPerm(key, arango.DeleteFiles)
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

			fullpath := c.Param("fullpath")
			fullpath = ultis.StandardizedPath(fullpath, true)
			bucketName := ultis.GetBucketName(fullpath)
			parentPath := ultis.GetParentPath(fullpath)
			fileName := ultis.GetFileName(fullpath)

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
					if e.ErrType == models.Locked {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": e.Msg,
						})

						return
					}
					if e.ErrType == models.DbError {
						c.JSON(http.StatusInternalServerError, gin.H{
							"error": "something when wrong",
						})

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				return
			}

			if !ultis.CheckBucketPerm(key.BucketId, bucket.Id) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "this key is not associated with this bucket",
				})

				return
			}

			if uid, ok := c.Get("uid"); !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err := nats.SendErrorEvent("uid not found at delete /accessKey/files/*fullpath",
					"Unknown Error")
				print(err)
				return
			} else {
				if uid.(string) != bucket.Uid {
					c.JSON(http.StatusForbidden, gin.H{
						"error": "permission denied",
					})
					return
				}
			}

			if bucket.Name != bucketName {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid bucket name",
				})

				return
			}

			err = arango.MarkDeleteFile(parentPath, fileName, bucket.Id)
			if err != nil {
				if e, ok := err.(*models.ModelError); ok {
					if e.ErrType == models.DocumentNotFound {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": "file not found",
						})

						return
					}
					if e.ErrType == models.DbError {
						c.JSON(http.StatusInternalServerError, gin.H{
							"error": "something when wrong",
						})

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "successfully deleted",
			})
		})
	}

	skr := r.Group("/key/files", middlewares.CheckBucketPublic, middlewares.SkipableAccessKeyAuthenticate)
	{
		skr.GET("/download", middlewares.ReqLogger("key", "B"), func(c *gin.Context) {
			k, ok := c.Get("key")
			isPublic := c.GetBool("is_public")
			if !ok && !isPublic {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err := nats.SendErrorEvent("key not found at get /key/files/download",
					"Unknown Error")
				print(err)
				return
			}

			key := k.(*arango.AccessKey)

			if !isPublic {
				hasPerm, err := CheckPerm(key, arango.ReadFiles)
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
			}

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

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				return
			}

			if !isPublic && !ultis.CheckBucketPerm(key.BucketId, bucket.Id) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "this key is not associated with this bucket",
				})

				return
			}

			userId := key.Uid
			if userId != bucket.Uid {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "permission denied",
				})
				return
			}

			err = arango.GetFileByFid(fid, func(reader io.Reader, metadata *arango.FileMetadata) error {
				if metadata.BucketId != bid {
					return &models.RouteError{
						Msg:     "invalid bucket",
						ErrType: models.InvalidBucket,
					}
				}

				var encryptInfo *arango.EncryptionInfo
				var r io.Reader
				if metadata.IsEncrypted {
					encryptInfo, err = arango.FindEncryptionInfoInDate(bucket.Id, metadata.UploadedDate)
					if err != nil {
						return err
					}

					r, err = ultis.DecryptReader(reader, crypto.StreamMeta{
						IV:   metadata.EncryptData.IV,
						Hash: metadata.EncryptData.Hash,
					}, encryptInfo.Passphrase)
					if err != nil {
						return err
					}
				} else {
					r = reader
				}

				extraHeaders := map[string]string{}

				teeReader := io.TeeReader(r, &ultis.DownloadBandwidthLogger{
					Uid:        userId,
					From:       key.Id,
					BucketId:   bucket.Id,
					SourceType: "key",
				})

				c.DataFromReader(http.StatusOK, metadata.Size, metadata.ContentType, teeReader, extraHeaders)

				//LOG
				_ = nats.SendDownloadFileEvent(metadata.Id, metadata.FileId, metadata.Name, metadata.Size,
					metadata.BucketId, metadata.UploadedDate, key.Uid)

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

				err = nats.SendErrorEvent(err.Error()+" at /key/files/download:",
					"File Error")
				return
			}
		})

		skr.GET("/download/*fullpath", middlewares.ReqLogger("key", "B"), func(c *gin.Context) {
			k, ok := c.Get("key")
			isPublic := c.GetBool("is_public")
			if !ok && !isPublic {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err := nats.SendErrorEvent("key not found at get /key/files/download/*fullpath",
					"Unknown Error")
				print(err)
				return
			}

			key := k.(*arango.AccessKey)

			if !isPublic {
				hasPerm, err := CheckPerm(key, arango.ReadFiles)
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
			}

			fullpath := c.Param("fullpath")
			fullpath = ultis.StandardizedPath(fullpath, true)
			bucketName := ultis.GetBucketName(fullpath)
			parentPath := ultis.GetParentPath(fullpath)
			fileName := ultis.GetFileName(fullpath)

			bucket, err := arango.FindBucketByName(bucketName)
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

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			if !isPublic && !ultis.CheckBucketPerm(key.BucketId, bucket.Id) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "this key is not associated with this bucket",
				})

				return
			}

			userId := key.Uid
			if userId != bucket.Uid {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "permission denied",
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

			err = arango.GetFileByFidIgnoreQueryMetadata(fileMeta.FileId, func(reader io.Reader) error {
				if fileMeta.BucketId != bucket.Id {
					return &models.RouteError{
						Msg:     "invalid bucket",
						ErrType: models.InvalidBucket,
					}
				}

				var encryptInfo *arango.EncryptionInfo
				var r io.Reader
				if fileMeta.IsEncrypted {
					encryptInfo, err = arango.FindEncryptionInfoInDate(bucket.Id, fileMeta.UploadedDate)
					if err != nil {
						return err
					}

					r, err = ultis.DecryptReader(reader, crypto.StreamMeta{
						IV:   fileMeta.EncryptData.IV,
						Hash: fileMeta.EncryptData.Hash,
					}, encryptInfo.Passphrase)
					if err != nil {
						return err
					}
				} else {
					r = reader
				}

				extraHeaders := map[string]string{
					//"Content-Disposition": `attachment; filename=` + fileMeta.Name,
				}

				teeReader := io.TeeReader(r, &ultis.DownloadBandwidthLogger{
					Uid:        userId,
					From:       key.Id,
					BucketId:   bucket.Id,
					SourceType: "key",
				})

				c.DataFromReader(http.StatusOK, fileMeta.Size, fileMeta.ContentType, teeReader, extraHeaders)

				//LOG
				_ = nats.SendDownloadFileEvent(fileMeta.Id, fileMeta.FileId, fileMeta.Name, fileMeta.Size,
					fileMeta.BucketId, fileMeta.UploadedDate, key.Uid)

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

				err = nats.SendErrorEvent("download failed: "+err.Error()+" at key/files/download/*fullpath",
					"File Error")
				return
			}
		})
	}

	sqkr := r.Group("/key-query/files", middlewares.CheckBucketPublic, middlewares.SkipableAccessKeyAuthenticateQuery)
	{
		sqkr.GET("/download", middlewares.ReqLogger("key", "B"), func(c *gin.Context) {
			k, ok := c.Get("key")
			isPublic := c.GetBool("is_public")
			if !ok && !isPublic {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err := nats.SendErrorEvent("key not found at get /key-query/files/download",
					"Unknown Error")
				print(err)
				return
			}

			key := k.(*arango.AccessKey)

			if !isPublic {
				hasPerm, err := CheckPerm(key, arango.ReadFiles)
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
			}

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

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				return
			}

			if !isPublic && !ultis.CheckBucketPerm(key.BucketId, bucket.Id) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "this key is not associated with this bucket",
				})

				return
			}

			userId := key.Uid
			if userId != bucket.Uid {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "permission denied",
				})
				return
			}

			err = arango.GetFileByFid(fid, func(reader io.Reader, metadata *arango.FileMetadata) error {
				if metadata.BucketId != bid {
					return &models.RouteError{
						Msg:     "invalid bucket",
						ErrType: models.InvalidBucket,
					}
				}

				var encryptInfo *arango.EncryptionInfo
				var r io.Reader
				if metadata.IsEncrypted {
					encryptInfo, err = arango.FindEncryptionInfoInDate(bucket.Id, metadata.UploadedDate)
					if err != nil {
						return err
					}

					r, err = ultis.DecryptReader(reader, crypto.StreamMeta{
						IV:   metadata.EncryptData.IV,
						Hash: metadata.EncryptData.Hash,
					}, encryptInfo.Passphrase)
					if err != nil {
						return err
					}
				} else {
					r = reader
				}

				extraHeaders := map[string]string{}

				teeReader := io.TeeReader(r, &ultis.DownloadBandwidthLogger{
					Uid:        userId,
					From:       key.Id,
					BucketId:   bucket.Id,
					SourceType: "key",
				})

				c.DataFromReader(http.StatusOK, metadata.Size, metadata.ContentType, teeReader, extraHeaders)

				//LOG
				_ = nats.SendDownloadFileEvent(metadata.Id, metadata.FileId, metadata.Name, metadata.Size,
					metadata.BucketId, metadata.UploadedDate, key.Uid)

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

				_ = nats.SendErrorEvent(err.Error()+" at key-query/files/download",
					"File Error")
				return
			}
		})

		sqkr.GET("/download/*fullpath", middlewares.ReqLogger("key", "B"), func(c *gin.Context) {
			k, ok := c.Get("key")
			isPublic := c.GetBool("is_public")
			if !ok && !isPublic {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err := nats.SendErrorEvent("key not found at get /key-query/files/download/*fullpath",
					"Unknown Error")
				print(err)
				return
			}

			key := k.(*arango.AccessKey)

			if !isPublic {
				hasPerm, err := CheckPerm(key, arango.ReadFiles)
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
			}

			fullpath := c.Param("fullpath")
			fullpath = ultis.StandardizedPath(fullpath, true)
			bucketName := ultis.GetBucketName(fullpath)
			parentPath := ultis.GetParentPath(fullpath)
			fileName := ultis.GetFileName(fullpath)

			bucket, err := arango.FindBucketByName(bucketName)
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

						err = nats.SendErrorEvent(err.Error(), "Db Error")
						return
					}
				}
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			if !isPublic && !ultis.CheckBucketPerm(key.BucketId, bucket.Id) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "this key is not associated with this bucket",
				})

				return
			}

			userId := key.Uid
			if userId != bucket.Uid {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "permission denied",
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

			err = arango.GetFileByFidIgnoreQueryMetadata(fileMeta.FileId, func(reader io.Reader) error {
				if fileMeta.BucketId != bucket.Id {
					return &models.RouteError{
						Msg:     "invalid bucket",
						ErrType: models.InvalidBucket,
					}
				}

				var encryptInfo *arango.EncryptionInfo
				var r io.Reader
				if fileMeta.IsEncrypted {
					encryptInfo, err = arango.FindEncryptionInfoInDate(bucket.Id, fileMeta.UploadedDate)
					if err != nil {
						return err
					}

					r, err = ultis.DecryptReader(reader, crypto.StreamMeta{
						IV:   fileMeta.EncryptData.IV,
						Hash: fileMeta.EncryptData.Hash,
					}, encryptInfo.Passphrase)
					if err != nil {
						return err
					}
				} else {
					r = reader
				}

				extraHeaders := map[string]string{
					//"Content-Disposition": `attachment; filename=` + fileMeta.Name,
				}

				teeReader := io.TeeReader(r, &ultis.DownloadBandwidthLogger{
					Uid:        userId,
					From:       key.Id,
					BucketId:   bucket.Id,
					SourceType: "key",
				})

				c.DataFromReader(http.StatusOK, fileMeta.Size, fileMeta.ContentType, teeReader, extraHeaders)

				//LOG
				_ = nats.SendDownloadFileEvent(fileMeta.Id, fileMeta.FileId, fileMeta.Name, fileMeta.Size,
					fileMeta.BucketId, fileMeta.UploadedDate, key.Uid)

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

				if _, ok := err.(*sio.Error); ok {
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("download failed: "+err.Error()+" at /key-query/files/download:",
					"File Error")
				return
			}
		})
	}

	//kpr := r.Group("/signed/files", middlewares.CheckSigned, middlewares.SignedReqCount)
	//{
	//	kpr.GET("/all", func(c *gin.Context) {
	//		limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	//		if err != nil {
	//			c.JSON(http.StatusBadRequest, gin.H{
	//				"error": "invalid limit format",
	//			})
	//
	//			return
	//		}
	//		offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	//		if err != nil {
	//			c.JSON(http.StatusBadRequest, gin.H{
	//				"error": "invalid offset format",
	//			})
	//
	//			return
	//		}
	//
	//		key, ok := c.Get("keyPair")
	//		if !ok {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something went wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("keyPair not found in authenticate at /signed/files/all:",
	//			//	"Unknown Error")
	//			return
	//		}
	//		keyPair := key.(*arango.KeyPair)
	//
	//		var isUploadPerm bool
	//		for _, perm := range keyPair.Permissions {
	//			if perm == "GetFileList" {
	//				isUploadPerm = true
	//				break
	//			}
	//		}
	//
	//		if !isUploadPerm {
	//			c.JSON(http.StatusForbidden, gin.H{
	//				"error": "not have permission",
	//			})
	//			return
	//		}
	//
	//		res, err := arango.FindMetadataByBid(keyPair.BucketId, limit, offset, false)
	//		if err != nil {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something when wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent(err.Error()+" at /signed/files/all:",
	//			//	"Db Error")
	//			return
	//		}
	//
	//		c.JSON(http.StatusOK, res)
	//	})
	//
	//	kpr.GET("/hidden/all", func(c *gin.Context) {
	//		limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	//		if err != nil {
	//			c.JSON(http.StatusBadRequest, gin.H{
	//				"error": "invalid limit format",
	//			})
	//
	//			return
	//		}
	//		offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	//		if err != nil {
	//			c.JSON(http.StatusBadRequest, gin.H{
	//				"error": "invalid offset format",
	//			})
	//
	//			return
	//		}
	//
	//		key, ok := c.Get("keyPair")
	//		if !ok {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something went wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("keyPair not found in authenticate at /signed/files/all:",
	//			//	"Unknown Error")
	//			return
	//		}
	//		keyPair := key.(*arango.KeyPair)
	//
	//		var isUploadPerm bool
	//		for _, perm := range keyPair.Permissions {
	//			if perm == "GetFileListHidden" {
	//				isUploadPerm = true
	//				break
	//			}
	//		}
	//
	//		if !isUploadPerm {
	//			c.JSON(http.StatusForbidden, gin.H{
	//				"error": "not have permission",
	//			})
	//			return
	//		}
	//
	//		res, err := arango.FindMetadataByBid(keyPair.BucketId, limit, offset, true)
	//		if err != nil {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something when wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent(err.Error()+" at authenticated files/hidden/all:",
	//			//	"Db Error")
	//			return
	//		}
	//
	//		c.JSON(http.StatusOK, res)
	//	})
	//
	//	kpr.POST("/upload", func(c *gin.Context) {
	//		key, ok := c.Get("keyPair")
	//		if !ok {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something went wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("keyPair not found in authenticate at /signed/files/upload:",
	//			//	"Unknown Error")
	//			return
	//		}
	//
	//		keyPair := key.(*arango.KeyPair)
	//		var isUploadPerm bool
	//		for _, perm := range keyPair.Permissions {
	//			if perm == "Upload" {
	//				isUploadPerm = true
	//				break
	//			}
	//		}
	//
	//		if !isUploadPerm {
	//			c.JSON(http.StatusForbidden, gin.H{
	//				"error": "not have permission",
	//			})
	//			return
	//		}
	//
	//		bucket, err := arango.FindBucketById(keyPair.BucketId)
	//		if err != nil {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": err.Error(),
	//			})
	//			return
	//		}
	//
	//		uploadFile, err := c.FormFile("file")
	//		if err != nil {
	//			c.JSON(http.StatusBadRequest, gin.H{
	//				"error": err.Error(),
	//			})
	//			return
	//		}
	//		queryPath := c.DefaultPostForm("path", "/")
	//		path := ultis.StandardizedPath("/"+bucket.Name+"/"+queryPath, true)
	//
	//		fileName := c.DefaultPostForm("name", uploadFile.Filename)
	//		//newPath := bucket.Name + path + fileName
	//
	//		if ok, err := ultis.ValidateFileName(fileName); err != nil {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something went wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("upload file signed > "+err.Error(), "validate")
	//			return
	//		} else if !ok {
	//			c.JSON(http.StatusBadRequest, gin.H{
	//				"error": "File should not contain special characters, from 1-255 characters",
	//			})
	//
	//			return
	//		}
	//
	//		fileContent, err := uploadFile.Open()
	//		if err != nil {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something went wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("open file failed at /signed/files/upload:",
	//			//	"File Error")
	//			return
	//		}
	//
	//		fileSize := uploadFile.Size
	//		ttlStr := c.DefaultPostForm("ttl", "0")
	//		ttl, err := strconv.ParseInt(ttlStr, 10, 64)
	//		if err != nil {
	//			c.JSON(http.StatusBadRequest, gin.H{
	//				"error": err.Error(),
	//			})
	//		}
	//
	//		isHiddenStr := c.DefaultPostForm("hidden", "false")
	//		isHidden, err := strconv.ParseBool(isHiddenStr)
	//		if err != nil {
	//			c.JSON(http.StatusBadRequest, gin.H{
	//				"error": err.Error(),
	//			})
	//
	//			return
	//		}
	//
	//		cType, err := ultis.GetFileContentType(fileContent)
	//		if err != nil {
	//			c.JSON(http.StatusBadRequest, gin.H{
	//				"error": "unknown file content type",
	//			})
	//
	//			return
	//		}
	//
	//		res, err := arango.SaveFile(fileContent, keyPair.BucketId, path, fileName, isHidden,
	//			cType, fileSize, time.Duration(ttl)*time.Second)
	//		if err != nil {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": err.Error(),
	//			})
	//			return
	//		}
	//
	//		//LOG
	//		_ = nats.SendUploadFileEvent(res.Id, res.FileId, res.Name, res.Size,
	//			res.BucketId, res.ContentType, res.UploadedDate, res.Path, res.IsHidden)
	//
	//		c.JSON(http.StatusOK, res)
	//	})
	//
	//	kpr.GET("/download", func(c *gin.Context) {
	//		key, ok := c.Get("keyPair")
	//		if !ok {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something went wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("keyPair not found in authenticate at signed/files/download:",
	//			//	"Unknown Error")
	//			return
	//		}
	//		keyPair := key.(*arango.KeyPair)
	//		var isDownloadPerm bool
	//		for _, perm := range keyPair.Permissions {
	//			if perm == "Download" {
	//				isDownloadPerm = true
	//				break
	//			}
	//		}
	//		if !isDownloadPerm {
	//			c.JSON(http.StatusForbidden, gin.H{
	//				"error": "not have permission",
	//			})
	//			return
	//		}
	//		fid := c.DefaultQuery("fileId", "")
	//
	//		fileMeta, err := arango.FindMetadataById(fid)
	//		if err != nil {
	//			c.JSON(http.StatusNotFound, gin.H{
	//				"error": "file not found",
	//			})
	//
	//			return
	//		}
	//
	//		if fileMeta.IsHidden {
	//			var isDownloadHiddenPerm bool
	//			for _, perm := range keyPair.Permissions {
	//				if perm == "DownloadHidden" {
	//					isDownloadHiddenPerm = true
	//					break
	//				}
	//			}
	//			if !isDownloadHiddenPerm {
	//				c.JSON(http.StatusNotFound, gin.H{
	//					"error": "file not found",
	//				})
	//
	//				return
	//			}
	//		}
	//
	//		err = arango.GetFileByFidIgnoreQueryMetadata(fileMeta.FileId, func(reader io.Reader) error {
	//			if fileMeta.BucketId != keyPair.BucketId {
	//				return &models.RouteError{
	//					Msg:     "invalid bucket",
	//					ErrType: models.InvalidBucket,
	//				}
	//			}
	//
	//			extraHeaders := map[string]string{
	//				//"Content-Disposition": `attachment; filename=` + fileMeta.Name,
	//			}
	//
	//			teeReader := io.TeeReader(reader, &ultis.DownloadBandwidthLogger{
	//				Uid:        keyPair.GeneratorUid,
	//				From:       keyPair.Public,
	//				BucketId:   keyPair.BucketId,
	//				SourceType: "signed",
	//			})
	//
	//			c.DataFromReader(http.StatusOK, fileMeta.Size, fileMeta.ContentType, teeReader, extraHeaders)
	//
	//			//LOG
	//			_ = nats.SendDownloadFileEvent(fileMeta.Id, fileMeta.FileId, fileMeta.Name, fileMeta.Size,
	//				fileMeta.BucketId, fileMeta.ContentType, fileMeta.UploadedDate, fileMeta.Path, fileMeta.IsHidden)
	//
	//			return nil
	//		})
	//
	//		if err != nil {
	//			if e, ok := err.(*models.RouteError); ok {
	//				if e.ErrType == models.InvalidBucket {
	//					c.JSON(http.StatusForbidden, gin.H{
	//						"error": err.Error(),
	//					})
	//
	//					return
	//				}
	//			}
	//
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something went wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("download failed: "+err.Error()+" at signed/files/download:",
	//			//	"File Error")
	//			return
	//		}
	//	})
	//
	//	kpr.GET("/download/*fullpath", func(c *gin.Context) {
	//		fullpath := c.Param("fullpath")
	//		fullpath = ultis.StandardizedPath(fullpath, true)
	//		bucketName := ultis.GetBucketName(fullpath)
	//		parentPath := ultis.GetParentPath(fullpath)
	//		fileName := ultis.GetFileName(fullpath)
	//
	//		key, ok := c.Get("keyPair")
	//		if !ok {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something went wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("keypair not found in authenticate at /files/upload:",
	//			//	"Unknown Error")
	//			return
	//		}
	//		keyPair := key.(*arango.KeyPair)
	//
	//		bucket, err := arango.FindBucketById(keyPair.BucketId)
	//		if err != nil {
	//			c.JSON(http.StatusNotFound, gin.H{
	//				"error": "bucket not found",
	//			})
	//
	//			return
	//		}
	//
	//		if bucket.Name != bucketName {
	//			c.JSON(http.StatusBadRequest, gin.H{
	//				"error": "invalid bucket name",
	//			})
	//
	//			return
	//		}
	//
	//		fileMeta, err := arango.FindMetadataByFilename(parentPath, fileName, bucket.Id)
	//		if err != nil {
	//			c.JSON(http.StatusNotFound, gin.H{
	//				"error": "file not found",
	//			})
	//
	//			return
	//		}
	//
	//		if fileMeta.IsHidden {
	//			var isDownloadHiddenPerm bool
	//			for _, perm := range keyPair.Permissions {
	//				if perm == "DownloadHidden" {
	//					isDownloadHiddenPerm = true
	//					break
	//				}
	//			}
	//			if !isDownloadHiddenPerm {
	//				c.JSON(http.StatusNotFound, gin.H{
	//					"error": "file not found",
	//				})
	//
	//				return
	//			}
	//		}
	//
	//		err = arango.GetFileByFidIgnoreQueryMetadata(fileMeta.FileId, func(reader io.Reader) error {
	//			if fileMeta.BucketId != keyPair.BucketId {
	//				return &models.RouteError{
	//					Msg:     "invalid bucket",
	//					ErrType: models.InvalidBucket,
	//				}
	//			}
	//
	//			extraHeaders := map[string]string{
	//				//"Content-Disposition": `attachment; filename=` + fileMeta.Name,
	//			}
	//
	//			teeReader := io.TeeReader(reader, &ultis.DownloadBandwidthLogger{
	//				Uid:        keyPair.GeneratorUid,
	//				From:       keyPair.Public,
	//				BucketId:   keyPair.BucketId,
	//				SourceType: "signed",
	//			})
	//
	//			c.DataFromReader(http.StatusOK, fileMeta.Size, fileMeta.ContentType, teeReader, extraHeaders)
	//
	//			//LOG
	//			_ = nats.SendDownloadFileEvent(fileMeta.Id, fileMeta.FileId, fileMeta.Name, fileMeta.Size,
	//				fileMeta.BucketId, fileMeta.ContentType, fileMeta.UploadedDate, fileMeta.Path, fileMeta.IsHidden)
	//
	//			return nil
	//		})
	//
	//		if err != nil {
	//			if e, ok := err.(*models.RouteError); ok {
	//				if e.ErrType == models.InvalidBucket {
	//					c.JSON(http.StatusForbidden, gin.H{
	//						"error": err.Error(),
	//					})
	//
	//					return
	//				}
	//			}
	//
	//			if e, ok := err.(*models.ModelError); ok {
	//				if e.ErrType == models.DocumentNotFound {
	//					c.JSON(http.StatusNotFound, gin.H{
	//						"error": "file not found",
	//					})
	//
	//					return
	//				}
	//			}
	//
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something went wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("download failed: "+err.Error()+" at /files/download:",
	//			//	"File Error")
	//			return
	//		}
	//	})
	//
	//	kpr.POST("/hidden", func(c *gin.Context) {
	//		qIsHidden := c.DefaultQuery("hidden", "false")
	//		qName := c.DefaultQuery("name", "")
	//		qPath := c.DefaultQuery("path", "")
	//
	//		key, ok := c.Get("keyPair")
	//		if !ok {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something went wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("keyPair not found in authenticate at /signed/files/all:",
	//			//	"Unknown Error")
	//			return
	//		}
	//		keyPair := key.(*arango.KeyPair)
	//
	//		var isMarkHiddenPerm bool
	//		for _, perm := range keyPair.Permissions {
	//			if perm == "MarkHidden" {
	//				isMarkHiddenPerm = true
	//				break
	//			}
	//		}
	//
	//		if !isMarkHiddenPerm {
	//			c.JSON(http.StatusForbidden, gin.H{
	//				"error": "not have permission",
	//			})
	//			return
	//		}
	//
	//		fm, err := arango.FindMetadataByFilename(qPath, qName, keyPair.BucketId)
	//		if err != nil {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something went wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("find file failed at /signed/files/hidden:",
	//			//	"File Error")
	//			return
	//		}
	//
	//		if keyPair.BucketId != fm.BucketId {
	//			c.JSON(http.StatusForbidden, gin.H{
	//				"error": "permission denied",
	//			})
	//			return
	//		}
	//
	//		isHidden, err := strconv.ParseBool(qIsHidden)
	//		if err != nil {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something went wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("parse failed at /signed/files/hidden:",
	//			//	"File Error")
	//			return
	//		}
	//		file, err := arango.ToggleHidden(fm.Path, isHidden)
	//		if err != nil {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something went wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("toggle failed at /signed/files/hidden:",
	//			//	"File Error")
	//			return
	//		}
	//
	//		c.JSON(http.StatusOK, file)
	//	})
	//
	//	kpr.DELETE("/*fullpath", func(c *gin.Context) {
	//		fullpath := c.Param("fullpath")
	//		fullpath = ultis.StandardizedPath(fullpath, true)
	//		bucketName := ultis.GetBucketName(fullpath)
	//		parentPath := ultis.GetParentPath(fullpath)
	//		fileName := ultis.GetFileName(fullpath)
	//
	//		var keyPair *arango.KeyPair
	//		if kp, ok := c.Get("keyPair"); !ok {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something when wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("uid not found at authenticated auth/files/download",
	//			//	"Unknown Error")
	//			return
	//		} else {
	//			keyPair = kp.(*arango.KeyPair)
	//
	//			var isDeletePerm bool
	//			for _, perm := range keyPair.Permissions {
	//				if perm == "DeleteFile" {
	//					isDeletePerm = true
	//					break
	//				}
	//			}
	//			if !isDeletePerm {
	//				c.JSON(http.StatusForbidden, gin.H{
	//					"error": "permission denied",
	//				})
	//				return
	//			}
	//		}
	//
	//		bid := keyPair.BucketId
	//		bucket, err := arango.FindBucketById(bid)
	//		if err != nil {
	//			if e, ok := err.(*models.ModelError); ok {
	//				if e.ErrType == models.DocumentNotFound {
	//					c.JSON(http.StatusBadRequest, gin.H{
	//						"error": "bid invalid",
	//					})
	//
	//					return
	//				}
	//				if e.ErrType == models.DbError {
	//					c.JSON(http.StatusInternalServerError, gin.H{
	//						"error": "something when wrong",
	//					})
	//
	//					//_ = nats.SendErrorEvent(err.Error()+" at authenticated auth/files/download",
	//					//	"Db Error")
	//					return
	//				}
	//			}
	//		}
	//
	//		if bucket.Name != bucketName {
	//			c.JSON(http.StatusBadRequest, gin.H{
	//				"error": "invalid bucket name",
	//			})
	//
	//			return
	//		}
	//
	//		err = arango.MarkDeleteFile(parentPath, fileName, bucket.Id)
	//		if err != nil {
	//			if e, ok := err.(*models.ModelError); ok {
	//				if e.ErrType == models.DocumentNotFound {
	//					c.JSON(http.StatusBadRequest, gin.H{
	//						"error": "file not found",
	//					})
	//
	//					return
	//				}
	//				if e.ErrType == models.DbError {
	//					c.JSON(http.StatusInternalServerError, gin.H{
	//						"error": "something when wrong",
	//					})
	//
	//					//_ = nats.SendErrorEvent(err.Error()+" at signed files/auth/upload:",
	//					//	"Db Error")
	//					return
	//				}
	//			}
	//
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something when wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent(err.Error()+" at signed files/auth/download:",
	//			//	"Db Error")
	//			return
	//		}
	//
	//		c.JSON(http.StatusOK, gin.H{
	//			"message": "successfully deleted",
	//		})
	//	})
	//}
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
