package routes

import (
	"errors"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/gin-gonic/gin"
	"github.com/gocql/gocql"
	"github.com/linxGnu/goseaweedfs"
	"github.com/m1ome/randstr"
	"io"
	"net/http"
	"strings"
	"time"
)

func TestRoute(r *gin.Engine) {
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Connected",
		})
	})
	r.GET("/testUser", func(c *gin.Context) {
		user, err := models.FindUserByUsername("test")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": errors.New("read fail"),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user": user,
		})
	})
	r.GET("/testInsertDB", func(c *gin.Context) {
		res := models.TestDb()
		c.JSON(http.StatusOK, res)
	})

	r.GET("/testRedis", func(c *gin.Context) {
		res := models.TestRedis()
		c.JSON(http.StatusOK, res)
	})

	r.POST("/upload", func(c *gin.Context) {
		someFile, err := c.FormFile("upload")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": errors.New("upload fail"),
			})
			return
		}
		path := c.PostForm("path")
		//Get bucket name via bucket_id
		bucketName := c.PostForm("bucket_id") + "Name"
		newPath := bucketName + path + someFile.Filename
		fileContent, _ := someFile.Open()
		fileSize := someFile.Size

		var res *goseaweedfs.FilerUploadResult
		res, err = models.TestUpload(fileContent, fileSize, newPath, "", "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, res)
	})

	r.GET("/download", func(c *gin.Context) {
		path := c.Query("path")
		tokens := strings.Split(path, "/")
		err := models.TestDownload(path, func(r io.Reader) error {

			contentLength := int64(7443)
			contentType := "Content-type : image/jpeg"

			extraHeaders := map[string]string{
				"Content-Disposition": `attachment; filename=` + tokens[len(tokens)-1],
			}
			c.DataFromReader(http.StatusOK, contentLength, contentType, r, extraHeaders)
			return nil

		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "download success",
		})
	})

	r.DELETE("/delete", func(c *gin.Context) {
		path := c.Query("path")
		err := models.TestDelete(path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "delete success",
		})
	})

	r.POST("/uploadFile", func(c *gin.Context) {
		file, _ := c.FormFile("file")
		f, _ := file.Open()
		testUuid, _ := gocql.RandomUUID()
		rands := randstr.GetString(5)
		mt, err := models.SaveFile(f, testUuid, "test"+rands, "/", file.Filename, false, "image/jpeg", file.Size, time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, mt)
	})

	r.POST("/downloadFile", func(c *gin.Context) {
		//models.GetFile(nil, )
		//
		//c.JSON(http.StatusOK, mt)
	})
}