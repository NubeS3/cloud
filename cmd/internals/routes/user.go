package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	scrypt "github.com/elithrar/simple-scrypt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
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
				if err = SendOTP(user.Username, user.Email, otp); err != nil {
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

			otp, err := models.GenerateOTP(user.Username)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			if err := SendOTP(user.Username, user.Email, otp); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
			}
			

			c.JSON(http.StatusOK, user)
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
				if err = SendOTP(user.Username, user.Email, otp); err != nil{
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
			if err = SendOTP(user.Username, user.Email, otp); err != nil{
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

func SendOTP(username string, email string, otp string) error {
	from := mail.NewEmail("NubeS3 Team", "nubes3.storage@gmail.com")
	subject := "Verification with OTP"
	to := mail.NewEmail(username, email)
	plainTextContent := "Your OTP will be expired in 5 minutes. Do not share it.\r\n OTP: "+otp 
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, "")
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	_, err := client.Send(message)
	if err != nil {
		return err
	}
	return nil
}
