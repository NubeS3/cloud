package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func TestRoute(r *gin.Engine) {
	r.GET("/test/nats/sendEmailEvent", func(c *gin.Context) {
		err := nats.SendEmailEvent("nhokbm13@gmail.com", "lu123", "123456", time.Now().Add(time.Minute*5))
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusOK, "sent")
	})

	//r.GET("/test", func(c *gin.Context) {
	//	c.JSON(http.StatusOK, gin.H{
	//		"message": "Connected",
	//	})
	//})
	//r.GET("/testUser", func(c *gin.Context) {
	//	user, err := cassandra.FindUserByUsername("test")
	//	if err != nil {
	//		c.JSON(http.StatusInternalServerError, gin.H{
	//			"error": errors.New("read fail"),
	//		})
	//		return
	//	}
	//	c.JSON(http.StatusOK, gin.H{
	//		"user": user,
	//	})
	//})
	//r.GET("/arango/test/user/create", func(c *gin.Context) {
	//	user, err := arango.SaveUser("test", "user",
	//		"test123", "1234",
	//		"abc@abc.com", time.Now(),
	//		"meow", true)
	//	if err != nil {
	//		c.JSON(http.StatusBadRequest, gin.H{
	//			"error": err.Error(),
	//		})
	//		return
	//	}
	//	c.JSON(http.StatusOK, user)
	//})
	//r.GET("/arango/test/otp/create", func(c *gin.Context) {
	//	otp, err := arango.GenerateOTP("test123", "abc@abc.com")
	//	if err != nil {
	//		if arangoDriver.IsNotFound(err) {
	//			c.JSON(http.StatusNotFound, gin.H{
	//				"error": "otp not found",
	//			})
	//			return
	//		} else {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": errors.New("read fail"),
	//			})
	//			return
	//		}
	//	}
	//	c.JSON(http.StatusOK, otp)
	//})
	//r.GET("/arango/test/otp/confirm", func(c *gin.Context) {
	//	err := arango.OTPConfirm("test123", "7A785414")
	//	if err != nil {
	//		if arangoDriver.IsNotFound(err) {
	//			c.JSON(http.StatusNotFound, gin.H{
	//				"error": "otp not found",
	//			})
	//			return
	//		} else {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": errors.New("read fail"),
	//			})
	//			return
	//		}
	//	}
	//	c.JSON(http.StatusOK, gin.H{
	//		"message": "otp confirmed.",
	//	})
	//})
	//r.GET("/arango/test/user/findId", func(c *gin.Context) {
	//	user, err := arango.FindUserById("14112")
	//	if err != nil {
	//		if arangoDriver.IsNotFound(err) {
	//			c.JSON(http.StatusNotFound, gin.H{
	//				"error": "user not found",
	//			})
	//			return
	//		} else {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": errors.New("read fail"),
	//			})
	//			return
	//		}
	//	}
	//	c.JSON(http.StatusOK, user)
	//})
	//r.GET("/arango/test/user/findUname", func(c *gin.Context) {
	//	user, err := arango.FindUserByUsername("test123")
	//	if err != nil {
	//		c.JSON(http.StatusInternalServerError, gin.H{
	//			"error": errors.New("read fail"),
	//		})
	//		return
	//	}
	//	c.JSON(http.StatusOK, user)
	//})
	//r.GET("/arango/test/bucket/create", func(c *gin.Context) {
	//	user, err := arango.InsertBucket("user123", "tringuyen", "vietnam")
	//	if err != nil {
	//		c.JSON(http.StatusBadRequest, gin.H{
	//			"error": err.Error(),
	//		})
	//		return
	//	}
	//	c.JSON(http.StatusOK, user)
	//})
	//r.GET("/arango/test/bucket/findName", func(c *gin.Context) {
	//	user, err := arango.FindBucketByName("tringuyen")
	//	if err != nil {
	//		if arangoDriver.IsNotFound(err) {
	//			c.JSON(http.StatusNotFound, gin.H{
	//				"error": "bucket not found",
	//			})
	//			return
	//		} else {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": errors.New("read fail"),
	//			})
	//			return
	//		}
	//	}
	//	c.JSON(http.StatusOK, user)
	//})
	//r.GET("/arango/test/bucket/findId", func(c *gin.Context) {
	//	bucket, err := arango.FindBucketById("9988")
	//	if err != nil {
	//		if arangoDriver.IsNotFound(err) {
	//			c.JSON(http.StatusNotFound, gin.H{
	//				"error": "bucket not found",
	//			})
	//			return
	//		} else {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": errors.New("read fail"),
	//			})
	//			return
	//		}
	//	}
	//	c.JSON(http.StatusOK, bucket)
	//})
	//r.GET("/arango/test/bucket/findUid", func(c *gin.Context) {
	//	user, err := arango.FindBucketByUid("user123", 10, 1)
	//	if err != nil {
	//		if arangoDriver.IsNotFound(err) {
	//			c.JSON(http.StatusNotFound, gin.H{
	//				"error": "bucket not found",
	//			})
	//			return
	//		} else {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": errors.New("read fail"),
	//			})
	//			return
	//		}
	//	}
	//	c.JSON(http.StatusOK, user)
	//})
	//
	//r.GET("/arango/test/token/create", func(c *gin.Context) {
	//	err := arango.GenerateRfToken("54155")
	//	if err != nil {
	//		if arangoDriver.IsNotFound(err) {
	//			c.JSON(http.StatusNotFound, gin.H{
	//				"error": "bucket not found",
	//			})
	//			return
	//		} else {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": errors.New("read fail"),
	//			})
	//			return
	//		}
	//	}
	//	c.JSON(http.StatusOK, gin.H{
	//		"msg": "rftoken gen",
	//	})
	//})
	//r.GET("/arango/test/token/findUid", func(c *gin.Context) {
	//	rfToken, err := arango.FindRfTokenByUid("54155")
	//	if err != nil {
	//		if arangoDriver.IsNotFound(err) {
	//			c.JSON(http.StatusNotFound, gin.H{
	//				"error": "token not found",
	//			})
	//			return
	//		} else {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": errors.New("read fail"),
	//			})
	//			return
	//		}
	//	}
	//	c.JSON(http.StatusOK, rfToken)
	//})
	//r.GET("/arango/test/token/update", func(c *gin.Context) {
	//	access, refresh, err := arango.UpdateToken("54155")
	//	if err != nil {
	//		if arangoDriver.IsNotFound(err) {
	//			c.JSON(http.StatusNotFound, gin.H{
	//				"error": "token not found",
	//			})
	//			return
	//		} else {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": errors.New("read fail"),
	//			})
	//			return
	//		}
	//	}
	//	c.JSON(http.StatusOK, gin.H{
	//		"access":  access,
	//		"refresh": refresh,
	//	})
	//})
	//r.POST("/arango/test/file/upload", func(c *gin.Context) {
	//	file, _ := c.FormFile("file")
	//	f, _ := file.Open()
	//	cType, _ := ultis.GetFileContentType(f)
	//	mt, err := arango.SaveFile(f, "1234", "test1", "", "cat_img.jpg", false, cType, file.Size, time.Nanosecond)
	//	if err != nil {
	//		c.JSON(http.StatusInternalServerError, err.Error())
	//		return
	//	}
	//
	//	c.JSON(http.StatusOK, mt)
	//})
	//r.GET("/arango/test/file/download", func(c *gin.Context) {
	//	_ = arango.GetFile("1234", "", "cat_img.jpg", func(reader io.Reader, metadata *arango.FileMetadata) error {
	//		extraHeaders := map[string]string{
	//			"Content-Disposition": `attachment; filename=` + metadata.Name,
	//		}
	//
	//		c.DataFromReader(http.StatusOK, metadata.Size, metadata.ContentType, reader, extraHeaders)
	//		return nil
	//	})
	//})
	//
	//r.GET("/testInsertDB", func(c *gin.Context) {
	//	res := cassandra.TestDb()
	//	c.JSON(http.StatusOK, res)
	//})
	//
	//r.GET("/testRedis", func(c *gin.Context) {
	//	res := cassandra.TestRedis()
	//	c.JSON(http.StatusOK, res)
	//})
	//
	//r.POST("/upload", func(c *gin.Context) {
	//	someFile, err := c.FormFile("upload")
	//	if err != nil {
	//		c.JSON(http.StatusInternalServerError, gin.H{
	//			"error": errors.New("upload fail"),
	//		})
	//		return
	//	}
	//	path := c.PostForm("path")
	//	//Get bucket name via bucket_id
	//	bucketName := c.PostForm("bucket_id") + "Name"
	//	newPath := bucketName + path + someFile.Filename
	//	fileContent, _ := someFile.Open()
	//	fileSize := someFile.Size
	//
	//	var res *goseaweedfs.FilerUploadResult
	//	res, err = cassandra.TestUpload(fileContent, fileSize, newPath, "", "")
	//	if err != nil {
	//		c.JSON(http.StatusInternalServerError, gin.H{
	//			"error": err.Error(),
	//		})
	//		return
	//	}
	//
	//	c.JSON(http.StatusOK, res)
	//})
	//
	//r.GET("/download", func(c *gin.Context) {
	//	path := c.Query("path")
	//	tokens := strings.Split(path, "/")
	//	err := cassandra.TestDownload(path, func(r io.Reader) error {
	//
	//		contentLength := int64(7443)
	//		contentType := "Content-type : image/jpeg"
	//
	//		extraHeaders := map[string]string{
	//			"Content-Disposition": `attachment; filename=` + tokens[len(tokens)-1],
	//		}
	//		c.DataFromReader(http.StatusOK, contentLength, contentType, r, extraHeaders)
	//		return nil
	//
	//	})
	//	if err != nil {
	//		c.JSON(http.StatusInternalServerError, gin.H{
	//			"error": err.Error(),
	//		})
	//		return
	//	}
	//
	//	c.JSON(http.StatusOK, gin.H{
	//		"message": "download success",
	//	})
	//})
	//
	//r.DELETE("/delete", func(c *gin.Context) {
	//	path := c.Query("path")
	//	err := cassandra.TestDelete(path)
	//	if err != nil {
	//		c.JSON(http.StatusInternalServerError, gin.H{
	//			"error": err.Error(),
	//		})
	//		return
	//	}
	//	c.JSON(http.StatusOK, gin.H{
	//		"message": "delete success",
	//	})
	//})
	//
	//r.POST("/uploadFile", func(c *gin.Context) {
	//	file, _ := c.FormFile("file")
	//	f, _ := file.Open()
	//	testUuid, _ := gocql.RandomUUID()
	//	rands := randstr.GetString(5)
	//	mt, err := cassandra.SaveFile(f, testUuid, "test"+rands, "/", file.Filename, false, "image/jpeg", file.Size, time.Hour)
	//	if err != nil {
	//		c.JSON(http.StatusInternalServerError, err.Error())
	//		return
	//	}
	//	c.JSON(http.StatusOK, mt)
	//})
	//
	//r.POST("/downloadFile", func(c *gin.Context) {
	//	//models.GetFile(nil, )
	//	//
	//	//c.JSON(http.StatusOK, mt)
	//})
}
