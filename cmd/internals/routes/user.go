package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"net/http"
	"time"

	"github.com/NubeS3/cloud/cmd/internals/ultis"
	scrypt "github.com/elithrar/simple-scrypt"
	"github.com/gin-gonic/gin"
)

func UserRoutes(route *gin.Engine) {
	userRoutesGroup := route.Group("/users")
	{
		userRoutesGroup.POST("/signin", middlewares.UnauthReqCount, func(c *gin.Context) {
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

			user, err := arango.FindUserByUsername(curSigninUser.Username)
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

			if user.IsBanned {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "account disabled",
				})
				return
			}

			accessToken, err := ultis.CreateToken(user.Id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				_ = nats.SendErrorEvent(err.Error()+" at user route/sign in/access token",
					"Token Error")
				return
			}

			rfToken, err := arango.FindRfTokenByUid(user.Id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				_ = nats.SendErrorEvent(err.Error()+" at user route/sign in/find refresh token",
					"Db Error")
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"accessToken":  accessToken,
				"refreshToken": rfToken.RfToken,
			})
		})

		userRoutesGroup.POST("/signup", middlewares.UnauthReqCount, func(c *gin.Context) {
			var user arango.User
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

			var curUser, err = arango.SaveUser(
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

			otp, err := arango.GenerateOTP(user.Username, curUser.Email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				_ = nats.SendErrorEvent(err.Error()+" at user route/sign up/generate otp",
					"Db Error")
				return
			}

			if err := SendOTP(user.Username, user.Email, otp.Otp, otp.ExpiredTime); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				_ = nats.SendErrorEvent(err.Error()+" at user route/sign up/send otp",
					"OTP Failed")
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "verify account via otp sent to your email",
			})
		})

		userRoutesGroup.PUT("/resend-otp", middlewares.UnauthReqCount, func(c *gin.Context) {
			type resendUser struct {
				Username string `json:"username"`
			}
			var curUser resendUser
			if err := c.ShouldBind(&curUser); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
			}

			user, err := arango.FindUserByUsername(curUser.Username)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "user not found",
				})
				return
			}

			otp, err := arango.GenerateOTP(user.Username, user.Email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				_ = nats.SendErrorEvent(err.Error()+" at user route/resend otp/generate otp",
					"Db Error")
				return
			}

			if err := SendOTP(user.Username, user.Email, otp.Otp, otp.ExpiredTime); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				_ = nats.SendErrorEvent(err.Error()+" at user route/resend otp/send otp",
					"OTP Error")
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "otp resent",
			})

		})

		userRoutesGroup.POST("/confirm-otp", middlewares.UnauthReqCount, func(c *gin.Context) {
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

			_, err := arango.FindOTPByUsername(curSigninUser.Username)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": err.Error(),
				})
				return
			}

			err = arango.OTPConfirm(curSigninUser.Username, curSigninUser.Otp)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				_ = nats.SendErrorEvent(err.Error()+" at user route/confirm otp/models otp confirm",
					"Db Error")
				return
			}

			user, err := arango.FindUserByUsername(curSigninUser.Username)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "user not found",
				})
				return
			}

			if err := arango.GenerateRfToken(user.Id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				_ = nats.SendErrorEvent(err.Error()+" at user route/confirm otp/generate refresh token",
					"Db Error")
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "otp confirmed",
			})
		})

		userRoutesGroup.POST("/update", middlewares.UserAuthenticate, middlewares.AuthReqCount, func(c *gin.Context) {
			type updateUser struct {
				Firstname string    `json:"firstname" binding:"required"`
				Lastname  string    `json:"lastname" binding:"required"`
				Dob       time.Time `json:"dob" binding:"required"`
				Company   string    `json:"company" binding:"required"`
				Gender    bool      `json:"gender" binding:"required"`
			}

			var curUpdateUser updateUser
			if err := c.ShouldBind(&curUpdateUser); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent("uid not found in authenticate at /users/update",
					"Unknown Error")
				return
			}

			user, err := arango.UpdateUserData(uid.(string), curUpdateUser.Firstname, curUpdateUser.Lastname,
				curUpdateUser.Dob, curUpdateUser.Company, curUpdateUser.Gender)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				_ = nats.SendErrorEvent(err.Error()+" at authenticated users/update",
					"Db Error")
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"firstname": user.Firstname,
				"lastname":  user.Lastname,
				"dob":       user.Dob,
				"company":   user.Company,
				"gender":    user.Gender,
			})
		})
	}
}

func SendOTP(username string, email string, otp string, expiredTime time.Time) error {
	//err := ultis.SendMail(
	//	username,
	//	email,
	//	"Verify email",
	//	"Enter the OTP we sent you via email to continue.\r\n\r\n"+otp+"\r\n\r\n"+
	//		"The OTP will be expired at "+expiredTime.Local().Format("02-01-2006 15:04")+". Do not share it to public.",
	//)
	//
	//if err != nil {
	//	return err
	//}

	return nats.SendEmailEvent(email, username, otp, expiredTime)
}
