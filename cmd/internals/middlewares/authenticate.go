package middlewares

import (
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
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

			refreshToken, err := arango.FindRfTokenByUid(userClaims.Id)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "unauthorized",
				})
				c.Abort()
				return
			}

			if refreshToken.RfToken != rfToken {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "unauthorized",
				})
				c.Abort()
				return
			}

			newAccessToken, newRfToken, err := arango.UpdateToken(refreshToken.Uid)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "unauthorized",
				})
				c.Abort()
				return
			}

			c.Writer.Header().Set("AccessToken", *newAccessToken)
			c.Writer.Header().Set("RefreshToken", *newRfToken)
			c.Set("uid", userClaims.Id)
			c.Next()
			return
		}

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

	user, err := arango.FindUserById(userClaims.Id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "something went wrong",
		})

		c.Abort()

		_ = nats.SendErrorEvent(err.Error(), "auth error")
		return
	}

	if user.IsBanned {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "account disabled",
		})

		c.Abort()
		return
	}

	if !user.IsActive {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "account is not active",
		})

		return
	}

	c.Set("uid", userClaims.Id)
	c.Next()
}
