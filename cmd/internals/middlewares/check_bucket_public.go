package middlewares

import (
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

func CheckBucketPublic(c *gin.Context) {
	fullpath := c.Param("fullpath")
	fullpath = ultis.StandardizedPath(fullpath, true)
	bucketName := ultis.GetBucketName(fullpath)

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

				//_ = nats.SendErrorEvent(err.Error()+" at authenticated auth/files/download",
				//	"Db Error")
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "something when wrong",
		})

		//_ = nats.SendErrorEvent(err.Error()+" at authenticated auth/files/download",
		//	"Db Error")
		return
	}

	if bucket.IsPublic {
		c.Set("is_public", true)
		c.Set("key", &arango.AccessKey{
			Name:                   "PUBLIC",
			BucketId:               "*",
			ExpiredDate:            time.Time{},
			FileNamePrefixRestrict: "",
			Permissions: []string{
				"ListKeys",
				"WriteKey",
				"DeleteKey",
				"ListBuckets",
				"WriteBucket",
				"DeleteBucket",
				"ReadBucketEncryption",
				"WriteBucketEncryption",
				"ReadBucketRetentions",
				"WriteBucketRetentions",
				"ListFiles",
				"ReadFiles",
				"ShareFiles",
				"WriteFiles",
				"DeleteFiles",
				"LockFiles",
			},
			Uid:     bucket.Uid,
			KeyType: "MASTER",
		})
		c.Next()
		return
	}
	c.Set("is_public", false)
	c.Next()
}

func SkipableAccessKeyAuthenticate(c *gin.Context) {
	isPublic := c.GetBool("is_public")
	if isPublic {
		c.Next()
		return
	}

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
	if ultis.TimeCheck(key.ExpiredDate) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Key expired",
		})

		c.Abort()
		return
	}

	c.Set("key", key)
	c.Next()
}

func SkipableAccessKeyAuthenticateQuery(c *gin.Context) {
	isPublic := c.GetBool("is_public")
	if isPublic {
		c.Next()
		return
	}

	authToken := c.Param("authorization")
	//if len(auths) < 2 {
	//	c.JSON(http.StatusUnauthorized, gin.H{
	//		"error": "unauthorized",
	//	})
	//	c.Abort()
	//	return
	//}

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
	if ultis.TimeCheck(key.ExpiredDate) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Key expired",
		})

		c.Abort()
		return
	}

	c.Set("key", key)
	c.Next()
}
