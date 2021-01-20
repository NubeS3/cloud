package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/scrypt"
	"net/http"
)

func UserRoutes(route *gin.Engine) {
	userRoutesGroup := route.Group("/users")
	{
		userRoutesGroup.POST("/signup", func(c *gin.Context) {
			var user models.User
			if err := c.ShouldBind(&user); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			if err := user.Save(); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, user)
		})

		userRoutesGroup.POST("/signin", func(c *gin.Context) {
			type signinUser struct {
				Username string `json:"username" binding:"required"`
				Password string `json:"password" binding:"required"`
			}
			var curSigninUser signinUser
			if err := c.ShouldBind(&curSigninUser); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			var curUser models.User
			if err := curUser.FindByUsername(curSigninUser.Username); err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Login failed",
				})
				return
			}

			err := scrypt.CompareHashAndPassword([]byte(curUser.HPassword), []byte(curSigninUser.Password))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Login failed",
				})
				return
			}

			accessToken, err := ultis.CreateToken(curUser.Id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"accessToken":  accessToken,
				"refreshToken": curUser.RefreshToken,
			})
		})
	}
}
