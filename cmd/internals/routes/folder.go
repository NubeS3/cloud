package routes

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/gin-gonic/gin"
)

func FolderRoutes(r *gin.Engine) {
	ar := r.Group("/auth/folders", middlewares.UserAuthenticate)
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
					"error": "something when wrong",
				})

				//_ = nats.SendErrorEvent("uid not found in authenticate at /folders/auth/allFolder",
				//	"Unknown Error")
				return
			}

			res, err := arango.FindFolderByOwnerId(uid.(string), limit, offset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				//_ = nats.SendErrorEvent(err.Error()+" at authenticated folders/auth/allFolder:",
				//	"Db Error")
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

			if ok, err := ultis.ValidateFolderName(curInsertedFolder.Name); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent("create folder auth > "+err.Error(), "validate")
				return
			} else if !ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Folder name must be 1-32 characters, contains only alphanumeric or -",
				})

				return
			}

			folderParent, err := arango.FindFolderByFullpath(ultis.StandardizedPath(curInsertedFolder.ParentPath, true))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": err.Error(),
				})

				//_ = nats.SendErrorEvent(err.Error()+" at authenticated folders/auth/insertFolder:",
				//	"Db Error")
				return
			}

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				//_ = nats.SendErrorEvent("uid not found in authenticate at /folders/auth/allFolder",
				//	"Unknown Error")
				return
			}

			if folderParent.OwnerId != uid.(string) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "permission denied",
				})
				return
			}

			_, err = arango.FindFolderByFullpath(folderParent.Fullpath + "/" + curInsertedFolder.Name)
			if err == nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "folder duplicated",
				})

				return
			}

			folder, err := arango.InsertFolder(curInsertedFolder.Name,
				folderParent.Id, uid.(string))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				//_ = nats.SendErrorEvent(err.Error()+" at authenticated folders/auth/folder:",
				//	"Db Error")
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

				//_ = nats.SendErrorEvent("uid not found in authenticate at /folders/auth/allByFid",
				//	"Unknown Error")
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

		ar.GET("/child/all/*full_path", func(c *gin.Context) {
			queryPath := c.Param("full_path")
			path := ultis.StandardizedPath(queryPath, true)
			token := strings.Split(path, "/")
			if len(token) < 2 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "empty path is disallowed",
				})

				return
			}
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

				//_ = nats.SendErrorEvent("uid not found in authenticate at /folders/auth/allByPath",
				//	"Unknown Error")
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
		//ar.DELETE("/*full_path", func(c *gin.Context) {
		//	queryPath := c.Param("full_path")
		//	path := ultis.StandardizedPath(queryPath, true)
		//	token := strings.Split(path, "/")
		//	bucketName := token[1]
		//	if _, err := arango.FindBucketByName(bucketName); err != nil {
		//		c.JSON(http.StatusInternalServerError, gin.H{
		//			"error": err.Error(),
		//		})
		//		return
		//	}
		//	folder, err := arango.FindFolderByFullpath(path)
		//	if err != nil {
		//		c.JSON(http.StatusInternalServerError, gin.H{
		//			"error": err.Error(),
		//		})
		//		return
		//	}
		//	uid, ok := c.Get("uid")
		//	if !ok {
		//		c.JSON(http.StatusInternalServerError, gin.H{
		//			"error": "something when wrong",
		//		})
		//
		//		//_ = nats.SendErrorEvent("uid not found in authenticate at /folders/auth/allByPath",
		//		//	"Unknown Error")
		//		return
		//	}
		//
		//	if folder.OwnerId != uid.(string) {
		//		c.JSON(http.StatusForbidden, gin.H{
		//			"error": "permission denied",
		//		})
		//		return
		//	}
		//
		//	c.JSON(http.StatusOK, folder.Children)
		//})
		ar.DELETE("/*full_path", func(c *gin.Context) {
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

				//_ = nats.SendErrorEvent("uid not found in authenticate at /folders/auth/allByPath",
				//	"Unknown Error")
				return
			}

			if folder.OwnerId != uid.(string) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "permission denied",
				})
				return
			}

			err = arango.RemoveFolderAndItsChildren(ultis.GetParentPath(path), folder.Name)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, folder.Id)
		})
	}

	kr := r.Group("/accessKey/folders", middlewares.AccessKeyAuthenticate)
	{
		kr.GET("/all", func(c *gin.Context) {
			k, ok := c.Get("key")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKey/info/:access_key:",
				//	"Unknown Error")
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

			res, err := arango.FindFolderByOwnerId(key.Uid, limit, offset)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				//_ = nats.SendErrorEvent(err.Error()+" at authenticated folders/auth/allFolder:",
				//	"Db Error")
				return
			}

			c.JSON(http.StatusOK, res)
		})

		kr.POST("/", func(c *gin.Context) {
			k, ok := c.Get("key")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKey/info/:access_key:",
				//	"Unknown Error")
				return
			}

			key := k.(*arango.AccessKey)
			hasPerm, err := CheckPerm(key, arango.WriteFiles)
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

			if ok, err := ultis.ValidateFolderName(curInsertedFolder.Name); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent("create folder auth > "+err.Error(), "validate")
				return
			} else if !ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Folder name must be 1-32 characters, contains only alphanumeric or -",
				})

				return
			}

			folderParent, err := arango.FindFolderByFullpath(ultis.StandardizedPath(curInsertedFolder.ParentPath, true))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": err.Error(),
				})

				//_ = nats.SendErrorEvent(err.Error()+" at authenticated folders/auth/insertFolder:",
				//	"Db Error")
				return
			}

			if folderParent.OwnerId != key.Uid {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "permission denied",
				})
				return
			}

			_, err = arango.FindFolderByFullpath(folderParent.Fullpath + "/" + curInsertedFolder.Name)
			if err == nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "folder duplicated",
				})

				return
			}

			folder, err := arango.InsertFolder(curInsertedFolder.Name,
				folderParent.Id, key.Uid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				//_ = nats.SendErrorEvent(err.Error()+" at authenticated folders/auth/folder:",
				//	"Db Error")
				return
			}

			c.JSON(http.StatusOK, folder)
		})

		kr.GET("/child/all/*full_path", func(c *gin.Context) {
			k, ok := c.Get("key")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKey/info/:access_key:",
				//	"Unknown Error")
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

			queryPath := c.Param("full_path")
			path := ultis.StandardizedPath(queryPath, true)
			token := strings.Split(path, "/")
			if len(token) < 2 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "empty path is disallowed",
				})

				return
			}
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

			if folder.OwnerId != key.Uid {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "permission denied",
				})
				return
			}

			c.JSON(http.StatusOK, folder.Children)
		})

		kr.DELETE("/*full_path", func(c *gin.Context) {
			k, ok := c.Get("key")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				//_ = nats.SendErrorEvent("uid not found in authenticated route at /accessKey/info/:access_key:",
				//	"Unknown Error")
				return
			}

			key := k.(*arango.AccessKey)
			hasPerm, err := CheckPerm(key, arango.DeleteFiles)
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

			if folder.OwnerId != key.Uid {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "permission denied",
				})
				return
			}

			err = arango.RemoveFolderAndItsChildren(ultis.GetParentPath(path), folder.Name)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, folder.Id)
		})
	}

	//acr := r.Group("/accessKey/folders", middlewares.ApiKeyAuthenticate)
	//{
	//	acr.GET("/child/all/*full_path", func(c *gin.Context) {
	//		queryPath := c.Param("full_path")
	//		path := ultis.StandardizedPath(queryPath, true)
	//		token := strings.Split(path, "/")
	//		bucketName := token[1]
	//		if _, err := arango.FindBucketByName(bucketName); err != nil {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": err.Error(),
	//			})
	//			return
	//		}
	//		folder, err := arango.FindFolderByFullpath(path)
	//		if err != nil {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": err.Error(),
	//			})
	//			return
	//		}
	//		key, ok := c.Get("accessKey")
	//		if !ok {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something went wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("accessKey not found in authenticate at accessKey/folders/child/all/*full_path:",
	//			//	"Unknown Error")
	//			return
	//		}
	//
	//		accessKey := key.(*arango.AccessKey)
	//		var isViewHiddenPerm bool
	//		for _, perm := range accessKey.Permissions {
	//			if perm == "GetFileListHidden" {
	//				isViewHiddenPerm = true
	//				break
	//			}
	//		}
	//
	//		var isViewPerm bool
	//		for _, perm := range accessKey.Permissions {
	//			if perm == "GetFileList" {
	//				isViewPerm = true
	//				break
	//			}
	//		}
	//
	//		if folder.OwnerId != accessKey.Uid {
	//			c.JSON(http.StatusForbidden, gin.H{
	//				"error": "permission denied",
	//			})
	//			return
	//		}
	//
	//		if isViewHiddenPerm {
	//			var nonHiddenChild []arango.Child
	//			for _, f := range folder.Children {
	//				if !f.IsHidden {
	//					nonHiddenChild = append(nonHiddenChild, f)
	//				}
	//			}
	//
	//			c.JSON(http.StatusOK, nonHiddenChild)
	//			return
	//		} else if isViewPerm {
	//			c.JSON(http.StatusOK, folder.Children)
	//			return
	//		}
	//	})
	//}
	//
	//kpr := r.Group("/keyPairs/folders", middlewares.CheckSigned)
	//{
	//	kpr.GET("/child/all/*full_path", func(c *gin.Context) {
	//		queryPath := c.Param("full_path")
	//		path := ultis.StandardizedPath(queryPath, true)
	//		token := strings.Split(path, "/")
	//		bucketName := token[1]
	//		if _, err := arango.FindBucketByName(bucketName); err != nil {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": err.Error(),
	//			})
	//			return
	//		}
	//		folder, err := arango.FindFolderByFullpath(path)
	//		if err != nil {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": err.Error(),
	//			})
	//			return
	//		}
	//		key, ok := c.Get("keyPair")
	//		if !ok {
	//			c.JSON(http.StatusInternalServerError, gin.H{
	//				"error": "something went wrong",
	//			})
	//
	//			//_ = nats.SendErrorEvent("keyPair not found in authenticate at keyPair/folders/child/all/*full_path:",
	//			//	"Unknown Error")
	//			return
	//		}
	//
	//		keyPair := key.(*arango.KeyPair)
	//		var isViewHiddenPerm bool
	//		for _, perm := range keyPair.Permissions {
	//			if perm == "GetFileListHidden" {
	//				isViewHiddenPerm = true
	//				break
	//			}
	//		}
	//
	//		var isViewPerm bool
	//		for _, perm := range keyPair.Permissions {
	//			if perm == "GetFileList" {
	//				isViewPerm = true
	//				break
	//			}
	//		}
	//
	//		if folder.OwnerId != keyPair.GeneratorUid {
	//			c.JSON(http.StatusForbidden, gin.H{
	//				"error": "permission denied",
	//			})
	//			return
	//		}
	//
	//		if isViewHiddenPerm {
	//			var nonHiddenChild []arango.Child
	//			for _, f := range folder.Children {
	//				if !f.IsHidden {
	//					nonHiddenChild = append(nonHiddenChild, f)
	//				}
	//			}
	//
	//			c.JSON(http.StatusOK, nonHiddenChild)
	//			return
	//		} else if isViewPerm {
	//			c.JSON(http.StatusOK, folder.Children)
	//			return
	//		}
	//	})
	//}
}
