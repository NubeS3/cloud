package middlewares

import (
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/gin-gonic/gin"
	"net/http"
)

func ApiKeyAuthenticate(c *gin.Context) {
	key := c.Query("accessKey")

	if key == "" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "access key missing",
		})
		c.Abort()
		return
	}

	accessKey, err := arango.FindAccessKeyByKey(key)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "access key mismatch",
		})
		c.Abort()
		return
	}
	c.Set("accessKey", accessKey)
	c.Next()
}
