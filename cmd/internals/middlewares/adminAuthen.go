package middlewares

import (
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func AdminAuthenticate(c *gin.Context) {
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
	var adminClaims ultis.AdminClaims
	token, err := ultis.ParseAdminToken(authToken, &adminClaims)

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

	admin, err := arango.FindAdminById(adminClaims.Id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})

		c.Abort()
		return
	}
	if admin.IsDisable {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Account Disabled",
		})

		c.Abort()
		return
	}

	c.Set("admin", admin)
	c.Next()
}
