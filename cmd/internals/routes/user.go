package routes

import (
	"net/http"
	"time"

	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	scrypt "github.com/elithrar/simple-scrypt"
	"github.com/gin-gonic/gin"
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

			if !user.IsActive {
				otp, err := models.UpdateOTP(user.Username)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": err.Error(),
					})
					return
				}
				if err = SendOTP(user.Username, user.Email, otp.Otp, otp.ExpiredTime); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": err.Error(),
					})
					return
				}
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
					"error": "username already used",
				})
				return
			}

			if _, err := models.FindUserByEmail(user.Email); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "email already used",
				})
				return
			}

			var curUser, err = models.SaveUser(
				user.Firstname,
				user.Lastname,
				user.Username,
				user.Pass,
				user.Email,
				user.Dob,
				user.Company,
				user.Gender,
			)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			otp, err := models.GenerateOTP(user.Username)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			if err := SendOTP(user.Username, user.Email, otp.Otp, otp.ExpiredTime); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
			}
			

			c.JSON(http.StatusOK, curUser)
		})

		userRoutesGroup.POST("/resend-otp", func(c *gin.Context) {
			type resendUser struct {
				Username string `json:"username"`
			}
			var curUser *resendUser
			if err := c.ShouldBind(&curUser); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
			}
			
			user, err := models.FindUserByUsername(curUser.Username)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "user not found",
				})
				return
			}
			_, err = models.GetUserOTP(user.Username)
			if err != nil {
				otp, err := models.GenerateOTP(user.Username)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": err.Error(),
					})
					return
				}
				if err = SendOTP(user.Username, user.Email, otp.Otp, otp.ExpiredTime); err != nil{
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": err.Error(),
					})
					return
				}
			}

			otp, err := models.UpdateOTP(user.Username)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}
			if err = SendOTP(user.Username, user.Email, otp.Otp, otp.ExpiredTime); err != nil{
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": err.Error(),
					})
					return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "otp resent",
			})

		})

		userRoutesGroup.POST("/confirm-otp", func(c *gin.Context) {
			type otpValidate struct {
				Username	string `json:"username"`
				Otp     	string `json:"otp"`
			}
			var curSigninUser *otpValidate
			if err := c.ShouldBind(&curSigninUser); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			_, err := models.GetOTPByUsername(curSigninUser.Username)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": err.Error(),
				})
				return
			}
			
			
			c.JSON(http.StatusOK, gin.H{
				"message": "otp confirmed",
			})
		})
	}
}

func SendOTP(username string, email string, otp string, expiredTime time.Time) error {
	err := ultis.SendMail(
		username, 
		email,
		"Verify email",
		"Enter the OTP we sent you via email to continue.\r\n" + otp + "\r\n" +
		"The OTP will be expired at " + expiredTime.Format("2006-01-02 15:04:05") + ". Do not share it to public.",
	)

	return err
}
