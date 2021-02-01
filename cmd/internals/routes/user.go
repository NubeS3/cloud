package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	scrypt "github.com/elithrar/simple-scrypt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func UserRoutes(route *gin.Engine) {
	userRoutesGroup := route.Group("/users")
	{
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

			user, err := models.FindUserByUsername(curSigninUser.Username)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid username",
				})
				return
			}

			err = scrypt.CompareHashAndPassword([]byte(user.Pass), []byte(curSigninUser.Password))
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid password",
				})
				return
			}

			accessToken, err := ultis.CreateToken(user.Id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"accessToken":  accessToken,
				"refreshToken": user.RefreshToken,
			})
		})

		userRoutesGroup.POST("/signup", func(c *gin.Context) {
			var user models.User
			if err := c.ShouldBind(&user); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			if _, err := models.FindUserByUsername(user.Username); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}
			if _, err := models.FindUserByEmail(user.Email); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			if _, err := models.SaveUser(
				user.Firstname,
				user.Lastname,
				user.Username,
				user.Pass,
				user.Email,
				user.Dob,
				user.Company,
				user.Gender,
			); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			err := models.GenerateOTP(user.Id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, user)
		})

		userRoutesGroup.POST("/confirm", func(c *gin.Context) {
			type otpValidate struct {
				Username string `json:"username"`
				Otp      string `json:"otp"`
			}

			var otpVal otpValidate
			if err := c.ShouldBind(&otpVal); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			//curValidationUser := models.User{}
			//if err := curValidationUser.ConfirmOtp(otpVal.Username, otpVal.Otp); err != nil {
			//	c.JSON(http.StatusBadRequest, gin.H{
			//		"error": err.Error(),
			//	})
			//	return
			//}
			//
			//if err := curValidationUser.GenerateRfToken(); err != nil {
			//	c.JSON(http.StatusInternalServerError, gin.H{
			//		"error": err.Error(),
			//	})
			//	return
			//}

			c.JSON(http.StatusOK, gin.H{
				"message": "otp confirmed",
			})
		})
	}
}
