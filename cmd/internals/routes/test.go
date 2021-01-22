package routes

import (
	"errors"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/gin-gonic/gin"
	"github.com/linxGnu/goseaweedfs"
	"net/http"
)

func TestRoute(r *gin.Engine) {
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Connected",
		})
	})
	r.GET("/testInsertDB", func(c *gin.Context) {
		res := models.TestDb()
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
}
