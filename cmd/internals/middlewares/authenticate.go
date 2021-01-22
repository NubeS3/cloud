package middlewares

import (
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func UserAuthenticate(c *gin.Context) {
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
	var userClaims ultis.UserClaims
	token, err := ultis.ParseToken(authToken, &userClaims)

	if err != nil {
		validationError, _ := err.(*jwt.ValidationError)

		if validationError.Errors == jwt.ValidationErrorExpired {
			rfToken := c.GetHeader("Refresh")
			if rfToken == "" {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "please log in again",
				})
				c.Abort()
				return
			}

			//TODO Wait for Model

			//var rfUser models.User
			//if err := rfUser.FindByIdAndRfToken(userClaims.Id, rfToken); err != nil {
			//	c.JSON(http.StatusUnauthorized, gin.H{
			//		"error": "unauthorized",
			//	})
			//	c.Abort()
			//	return
			//}

			//newAccessToken, newRfToken, err := "access", "refresh", errors.New("")
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "unauthorized",
				})
				c.Abort()
				return
			}

			//END TODO

			//c.Writer.Header().Set("AccessToken", *newAccessToken)
			//c.Writer.Header().Set("RefreshToken", *newRfToken)
		}

		if err == jwt.ErrSignatureInvalid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			c.Abort()
			return
		}

		c.Set("id", userClaims.Id)
		c.Next()
		return
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

	c.Set("id", userClaims.Id)
	c.Next()
}

func ApiKeyAuthenticate(c *gin.Context) {
	accessKey := c.Param("accessKey")

	if accessKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		c.Abort()
		return
	}

	//TODO Query User By AccessKey and Set user info

	c.Next()
}
