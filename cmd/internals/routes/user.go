package routes

import (
	"fmt"
	"github.com/NubeS3/cloud/cmd/internals/models/cassandra"
	"net/http"
	"time"

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

			user, err := cassandra.FindUserByUsername(curSigninUser.Username)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid username",
				})
				return
			}

			err = scrypt.CompareHashAndPassword([]byte(user.Pass), []byte(curSigninUser.Password))
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": err.Error(),
				})
				return
			}

			if !user.IsActive {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "user have not verified account via otp",
				})
				return
			}

			accessToken, err := ultis.CreateToken(user.Id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				fmt.Println("user route/sign in/access token: " + err.Error())
				return
			}

			rfToken, err := cassandra.FindRfTokenByUid(user.Id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				fmt.Println("user route/sign in/find refresh token: " + err.Error())
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"id":           user.Id,
				"accessToken":  accessToken,
				"refreshToken": rfToken.RfToken,
			})
		})

		userRoutesGroup.POST("/signup", func(c *gin.Context) {
			var user cassandra.User
			if err := c.ShouldBind(&user); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			// if _, err := models.FindUserByUsername(user.Username); err != nil {
			// 	c.JSON(http.StatusBadRequest, gin.H{
			// 		"error": "username already used",
			// 	})
			// 	return
			// }

			// if _, err := models.FindUserByEmail(user.Email); err != nil {
			// 	c.JSON(http.StatusBadRequest, gin.H{
			// 		"error": "email already used",
			// 	})
			// 	return
			// }

			var curUser, err = cassandra.SaveUser(
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

			otp, err := cassandra.GenerateOTP(user.Username, curUser.Id, curUser.Email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				fmt.Println("user route/sign up/generate otp: ", err.Error())
				return
			}

			if err := SendOTP(user.Username, user.Email, otp.Otp, otp.ExpiredTime); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				fmt.Println("user route/sign up/send otp: " + err.Error())
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "verify account via otp sent to your email",
			})
		})

		userRoutesGroup.PUT("/resend-otp", func(c *gin.Context) {
			type resendUser struct {
				Username string `json:"username"`
			}
			var curUser resendUser
			if err := c.ShouldBind(&curUser); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
			}

			user, err := cassandra.FindUserByUsername(curUser.Username)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "user not found",
				})
				return
			}
			_, err = cassandra.GetUserOTP(user.Username)
			if err != nil {
				otp, err := cassandra.GenerateOTP(user.Username, user.Id, user.Email)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "internal server error",
					})
					fmt.Println("user route/resend otp/gennerate otp: " + err.Error())
					return
				}
				if err = SendOTP(user.Username, user.Email, otp.Otp, otp.ExpiredTime); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "internal server error",
					})
					fmt.Println("user route/resend otp/send otp:" + err.Error())
					return
				}
			}

			otp, err := cassandra.ReGenerateOTP(user.Username)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				fmt.Println("user route/resend otp/regenerate otp: " + err.Error())
				return
			}
			if err = SendOTP(user.Username, user.Email, otp.Otp, otp.ExpiredTime); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				fmt.Println("user route/resend otp/resend otp: " + err.Error())
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "otp resent",
			})

		})

		userRoutesGroup.POST("/confirm-otp", func(c *gin.Context) {
			type otpValidate struct {
				Username string `json:"username"`
				Otp      string `json:"otp"`
			}
			var curSigninUser otpValidate
			if err := c.ShouldBind(&curSigninUser); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			_, err := cassandra.GetOTPByUsername(curSigninUser.Username)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": err.Error(),
				})
				return
			}

			err = cassandra.OTPConfirm(curSigninUser.Username, curSigninUser.Otp)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "interal server error",
				})
				fmt.Println("user route/confirm otp/models otp confirm: " + err.Error())
				return
			}

			user, err := cassandra.FindUserByUsername(curSigninUser.Username)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "user not found",
				})
				return
			}

			if err := cassandra.GenerateRfToken(user.Id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				fmt.Println("user route/confirm otp/generate refresh token: " + err.Error())
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
		"Enter the OTP we sent you via email to continue.\r\n\r\n"+otp+"\r\n\r\n"+
			"The OTP will be expired at "+expiredTime.Local().Format("02-01-2006 15:04")+". Do not share it to public.",
	)

	return err
}
