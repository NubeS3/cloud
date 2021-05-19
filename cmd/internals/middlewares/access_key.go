package middlewares

import (
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
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

	if accessKey.ExpiredDate.Before(time.Now()) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "access key expired",
		})
		c.Abort()
		return
	}

	c.Set("accessKey", accessKey)
	c.Next()
}

func AccessKeyAuthenticate(c *gin.Context) {
	authToken := c.GetHeader("Authorization")
	auths := strings.Split(authToken, "Bearer ")
	if len(auths) < 2 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		c.Abort()
		return
	}

	authToken = auths[1]
	var keyClaims ultis.KeyClaims
	token, err := ultis.ParseKeyToken(authToken, &keyClaims)

	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			c.Abort()
			return
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "access token invalid",
			})
			c.Abort()
			return
		}
	}

	if !token.Valid {
		if err == jwt.ErrSignatureInvalid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			c.Abort()
			return
		}
	}

	key, err := arango.FindAccessKeyById(keyClaims.KeyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})

		c.Abort()
		return
	}
	if key.ExpiredDate != time.Unix(0, 0) && key.ExpiredDate.Before(time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Key expired",
		})

		c.Abort()
		return
	}

	c.Set("key", key)
	c.Next()
}
