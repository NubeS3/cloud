package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func FolderRoutes(r *gin.Engine) {
	ar := r.Group("/folders/auth", middlewares.UserAuthenticate)
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

		ar.POST("/insertFolder", func(c *gin.Context) {
			type insertFolder struct {
				Name     string `json:"name"`
				ParentId string `json:"parent_id"`
			}

			var curInsertedFolder insertFolder
			if err := c.ShouldBind(&curInsertedFolder); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
			}

			folderParent, err := arango.FindFolderById(curInsertedFolder.ParentId)
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
				curInsertedFolder.ParentId, uid.(string))

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at authenticated folders/auth/insertFolder:",
					"Db Error")
				return
			}

			c.JSON(http.StatusOK, folder)
		})

		ar.GET("/allChild", func(c *gin.Context) {
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

				_ = nats.SendErrorEvent("uid not found in authenticate at /folders/auth/allFolder",
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
