package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
)

func FolderRoutes(r *gin.Engine) {
	ar := r.Group("/auth/folders", middlewares.UserAuthenticate)
	{
		ar.GET("/allFolder", func(c *gin.Context) {
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
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent("uid not found in authenticate at /folders/auth/allFolder",
					"Unknown Error")
				return
			}

			res, err := arango.FindFolderByOwnerId(uid.(string), limit, offset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at authenticated folders/auth/allFolder:",
					"Db Error")
				return
			}

			c.JSON(http.StatusOK, res)
		})

		ar.POST("/", func(c *gin.Context) {
			type insertFolder struct {
				Name       string `json:"name"`
				ParentPath string `json:"parent_path"`
			}

			var curInsertedFolder insertFolder
			if err := c.ShouldBind(&curInsertedFolder); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
			}

			folderParent, err := arango.FindFolderByFullpath(curInsertedFolder.ParentPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at authenticated folders/auth/insertFolder:",
					"Db Error")
				return
			}

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent("uid not found in authenticate at /folders/auth/allFolder",
					"Unknown Error")
				return
			}

			if folderParent.OwnerId != uid.(string) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "permission denied",
				})
				return
			}

			folder, err := arango.InsertFolder(curInsertedFolder.Name,
				folderParent.Id, uid.(string))

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at authenticated folders/auth/folder:",
					"Db Error")
				return
			}

			c.JSON(http.StatusOK, folder)
		})

		ar.GET("/child/allByFid", func(c *gin.Context) {
			fid := c.DefaultQuery("folderId", "")
			folder, err := arango.FindFolderById(fid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}
			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent("uid not found in authenticate at /folders/auth/allByFid",
					"Unknown Error")
				return
			}

			if folder.OwnerId != uid.(string) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "permission denied",
				})
				return
			}
			c.JSON(http.StatusOK, folder.Children)
		})

		ar.GET("/child/all/:full_path", func(c *gin.Context) {
			queryPath := c.Param("full_path")
			path := ultis.StandardizedPath(queryPath, true)
			token := strings.Split(path, "/")
			bucketName := token[1]
			if _, err := arango.FindBucketByName(bucketName); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}
			folder, err := arango.FindFolderByFullpath(path)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}
			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent("uid not found in authenticate at /folders/auth/allByPath",
					"Unknown Error")
				return
			}

			if folder.OwnerId != uid.(string) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "permission denied",
				})
				return
			}
			c.JSON(http.StatusOK, folder.Children)
		})
	}
}
