package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"net/http"
	"strconv"
	"time"

	"github.com/NubeS3/cloud/cmd/internals/ultis"
	scrypt "github.com/elithrar/simple-scrypt"
	"github.com/gin-gonic/gin"
)

func UserRoutes(route *gin.Engine) {
	userRoutesGroup := route.Group("/users")
	{
		userRoutesGroup.GET("/validate-email/*email", middlewares.ReqLogger("unauth", ""), func(c *gin.Context) {
			email := c.Param("emai")
			_, err := arango.FindUserByEmail(email)
			if err != nil {
				if err.(*models.ModelError).ErrType == models.NotFound || err.(*models.ModelError).ErrType == models.DocumentNotFound {
					c.JSON(http.StatusNotFound, gin.H{})

					return
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")

				return
			}

			c.JSON(http.StatusOK, gin.H{})
		})

		userRoutesGroup.GET("/check-active-status", middlewares.UserAuthenticate, middlewares.ReqLogger("auth", "A"), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{})
		})

		userRoutesGroup.POST("/signin", middlewares.ReqLogger("unauth", ""), func(c *gin.Context) {
			type signinUser struct {
				Email    string `json:"email" binding:"required"`
				Password string `json:"password" binding:"required"`
			}
			var curSigninUser signinUser
			if err := c.ShouldBind(&curSigninUser); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			user, err := arango.FindUserByEmail(curSigninUser.Email)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "invalid email",
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

			//if !user.IsActive {
			//	c.JSON(http.StatusUnauthorized, gin.H{
			//		"error": "user have not verified account via otp",
			//	})
			//	return
			//}

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
				err = nats.SendErrorEvent(err.Error()+" at user route/sign in/access token",
					"Token Error")
				return
			}

			rfToken, err := arango.FindRfTokenByUid(user.Id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"accessToken":  accessToken,
				"refreshToken": rfToken.RfToken,
			})
		})

		userRoutesGroup.POST("/signup", middlewares.ReqLogger("unauth", ""), func(c *gin.Context) {
			type signupUser struct {
				Email    string `json:"email" binding:"required"`
				Password string `json:"password" binding:"required"`
			}

			var user signupUser
			if err := c.ShouldBind(&user); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			//if ok, err := ultis.ValidateUsername(user.Username); err != nil {
			//	c.JSON(http.StatusInternalServerError, gin.H{
			//		"error": "something went wrong",
			//	})
			//
			//	_ = nats.SendErrorEvent("user signup > "+err.Error(), "validate")
			//	return
			//} else if !ok {
			//	c.JSON(http.StatusBadRequest, gin.H{
			//		"error": "Username must be 8-24 characters, does not start or end with _ or ., does not contain __, _., ._, ..",
			//	})
			//
			//	return
			//}

			if ok, err := ultis.ValidateEmail(user.Email); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Validate Error")
				return
			} else if !ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Email must be <something>@<something.com>",
				})

				return
			}

			if ok, err := ultis.ValidatePassword(user.Password); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Validate Error")
				return
			} else if !ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Password must be 8-32 characters, contains at least one uppercase, one lowercase, one number and one special character",
				})

				return
			}

			createdUser, err := arango.SaveUser(user.Password, user.Email)
			if err != nil {
				if err.(*models.ModelError).ErrType == models.Duplicated {
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": err.Error(),
					})

					return
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")

				return
			}

			if err := arango.GenerateRfToken(createdUser.Id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			otp, err := arango.GenerateOTP(createdUser.Email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				_ = nats.SendErrorEvent(err.Error()+" at user route/sign up/generate otp",
					"Db Error")
				return
			}

			if err := SendOTP(createdUser.Email, otp.Otp, otp.ExpiredTime); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				_ = nats.SendErrorEvent(err.Error()+" at user route/sign up/send otp",
					"OTP Failed")
				return
			}
			c.JSON(http.StatusOK, createdUser)
		})

		userRoutesGroup.PUT("/send-otp", middlewares.ReqLogger("unauth", ""), func(c *gin.Context) {
			type sentUser struct {
				Email string `json:"email"`
			}
			var curUser sentUser
			if err := c.ShouldBind(&curUser); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				err = nats.SendErrorEvent(err.Error(), "Db Error")
				if err != nil {
					println(err)
				}
				return
			}

			user, err := arango.FindUserByEmail(curUser.Email)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "user not found",
				})
				return
			}

			otp, err := arango.GenerateOTP(user.Email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				err = nats.SendErrorEvent(err.Error(), "Db Error")
				if err != nil {
					println(err)
				}
				return
			}

			if err := SendOTP(user.Email, otp.Otp, otp.ExpiredTime); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				err = nats.SendErrorEvent(err.Error()+" at user route/resend otp/send otp",
					"OTP Error")
				if err != nil {
					println(err)
				}
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "otp sent",
			})
		})

		userRoutesGroup.POST("/confirm-otp", middlewares.ReqLogger("unauth", ""), func(c *gin.Context) {
			type otpValidate struct {
				Email string `json:"email"`
				Otp   string `json:"otp"`
			}
			var curSigninUser otpValidate
			if err := c.ShouldBind(&curSigninUser); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}

			//_, err := arango.FindOTPByEmail(curSigninUser.Email)
			//if err != nil {
			//	c.JSON(http.StatusUnauthorized, gin.H{
			//		"error": err.Error(),
			//	})
			//	return
			//}

			err := arango.OTPConfirm(curSigninUser.Email, curSigninUser.Otp)
			if err != nil {
				if e, ok := err.(*models.ModelError); ok {
					if e.ErrType == models.DbError {
						c.JSON(http.StatusInternalServerError, gin.H{
							"error": "internal server error",
						})
						err = nats.SendErrorEvent(err.Error(), "Db Error")

						return
					} else {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": err.Error(),
						})

						return
					}
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "otp confirmed",
			})
		})

		userRoutesGroup.POST("/update", middlewares.UserAuthenticate, middlewares.ReqLogger("auth", "C"), func(c *gin.Context) {
			//type updateUser struct {
			//	Firstname string    `json:"firstname" binding:"required"`
			//	Lastname  string    `json:"lastname" binding:"required"`
			//	Dob       time.Time `json:"dob" binding:"required"`
			//	Company   string    `json:"company" binding:"required"`
			//	Gender    bool      `json:"gender" binding:"required"`
			//}
			//
			//var curUpdateUser updateUser
			//if err := c.ShouldBind(&curUpdateUser); err != nil {
			//	c.JSON(http.StatusBadRequest, gin.H{
			//		"error": err.Error(),
			//	})
			//	return
			//}
			//
			//uid, ok := c.Get("uid")
			//if !ok {
			//	c.JSON(http.StatusInternalServerError, gin.H{
			//		"error": "something when wrong",
			//	})
			//
			//	_ = nats.SendErrorEvent("uid not found in authenticate at /users/update",
			//		"Unknown Error")
			//	return
			//}
			//
			//user, err := arango.UpdateUserData(uid.(string), curUpdateUser.Firstname, curUpdateUser.Lastname,
			//	curUpdateUser.Dob, curUpdateUser.Company, curUpdateUser.Gender)
			//if err != nil {
			//	c.JSON(http.StatusInternalServerError, gin.H{
			//		"error": "something when wrong",
			//	})
			//
			//	_ = nats.SendErrorEvent(err.Error()+" at authenticated users/update",
			//		"Db Error")
			//	return
			//}
			//c.JSON(http.StatusOK, gin.H{
			//	"firstname": user.Firstname,
			//	"lastname":  user.Lastname,
			//	"dob":       user.Dob,
			//	"company":   user.Company,
			//	"gender":    user.Gender,
			//})
		})

		userRoutesGroup.POST("/update-password", middlewares.UserAuthenticate, middlewares.ReqLogger("auth", "C"), func(c *gin.Context) {
			type updateUser struct {
				OldPassword string `json:"old_password"`
				NewPassword string `json:"new_password"`
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

			user, err := arango.FindUserById(uid.(string))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent("uid not found in authenticate at /users/update",
					"Unknown Error")
				if err != nil {
					println(err)
				}

				return
			}

			err = scrypt.CompareHashAndPassword([]byte(user.Pass), []byte(curUpdateUser.OldPassword))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "incorrect old password",
				})
				return
			}

			u, err := arango.UpdateUserPassword(uid.(string), curUpdateUser.NewPassword)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something when wrong",
				})

				err = nats.SendErrorEvent("uid not found in authenticate at /users/update",
					"Unknown Error")
				if err != nil {
					println(err)
				}

				return
			}

			c.JSON(http.StatusOK, u)
		})

		userRoutesGroup.GET("/bandwidth-report", middlewares.UserAuthenticate, middlewares.ReqLogger("none", "B"), func(c *gin.Context) {
			from, err := strconv.ParseInt(c.DefaultQuery("from", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid from format",
				})

				return
			}

			to, err := strconv.ParseInt(c.DefaultQuery("to", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid from format",
				})

				return
			}

			fromT := time.Unix(from, 0)
			toT := time.Unix(to, 0)

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent("uid not found at user get bandwidth report", "Unknown Error")
				return
			}

			total, err := nats.SumBandwidthByDateRangeWithUid(uid.(string), fromT, toT)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent("error at user get bandwidth report: "+err.Error(), "Unknown Error")
				return
			}

			c.JSON(http.StatusOK, total)
		})

		userRoutesGroup.GET("/request-report", middlewares.UserAuthenticate, middlewares.ReqLogger("none", "C"), func(c *gin.Context) {
			from, err := strconv.ParseInt(c.DefaultQuery("from", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid from format",
				})

				return
			}

			to, err := strconv.ParseInt(c.DefaultQuery("to", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid from format",
				})

				return
			}

			fromT := time.Unix(from, 0)
			toT := time.Unix(to, 0)

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent("uid not found at user get request report", "Unknown Error")
				return
			}

			total, err := nats.CountByClass(uid.(string), fromT, toT)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent("error at user get request report: "+err.Error(), "Unknown Error")
				return
			}

			c.JSON(http.StatusOK, total)
		})

		userRoutesGroup.GET("/bandwidth-report/bucket/:bucketId", middlewares.UserAuthenticate, middlewares.ReqLogger("none", "B"), func(c *gin.Context) {
			bucketID := c.Param("bucketId")
			bucket, err := arango.FindBucketById(bucketID)
			if err != nil {
				if err, ok := err.(*models.ModelError); ok {
					if err.ErrType == models.NotFound || err.ErrType == models.DocumentNotFound {
						c.JSON(http.StatusNotFound, gin.H{
							"error": "bucket not found",
						})

						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				_ = nats.SendErrorEvent("error at bandwidth report bucket: "+err.Error(), "db error")
				return
			}

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			if bucket.Uid != uid {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "not your bucket",
				})

				return
			}

			from, err := strconv.ParseInt(c.DefaultQuery("from", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid from format",
				})

				return
			}

			to, err := strconv.ParseInt(c.DefaultQuery("to", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid from format",
				})

				return
			}

			fromT := time.Unix(from, 0)
			toT := time.Unix(to, 0)

			total, err := nats.SumBandwidthByDateRangeWithBucketId(bucket.Id, fromT, toT)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent("error at user get bandwidth report bucket: "+err.Error(), "Unknown Error")
				return
			}

			c.JSON(http.StatusOK, total)
		})

		userRoutesGroup.GET("/bandwidth-report/access-key/:key", middlewares.UserAuthenticate, middlewares.ReqLogger("none", "B"), func(c *gin.Context) {
			k := c.Param("key")
			key, err := arango.FindAccessKeyById(k)
			if err != nil {
				if err, ok := err.(*models.ModelError); ok {
					if err.ErrType == models.NotFound || err.ErrType == models.DocumentNotFound {
						c.JSON(http.StatusNotFound, gin.H{
							"error": "key_id not found",
						})

						return
					}
				}

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent(err.Error(), "Db Error")
				return
			}

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent("uid not found at user get bandwidth report access-key", "Unknown Error")
				return
			}

			if key.Uid != uid {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "not your key",
				})

				return
			}

			from, err := strconv.ParseInt(c.DefaultQuery("from", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid from format",
				})

				return
			}

			to, err := strconv.ParseInt(c.DefaultQuery("to", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid from format",
				})

				return
			}

			fromT := time.Unix(from, 0)
			toT := time.Unix(to, 0)

			total, err := nats.SumBandwidthByDateRangeWithFrom(key.Id, fromT, toT)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent("uid not found at user get bandwidth report access-key", "Unknown Error")
				return
			}

			c.JSON(http.StatusOK, total)
		})

		userRoutesGroup.GET("/report/size", middlewares.UserAuthenticate, middlewares.ReqLogger("none", "B"), func(c *gin.Context) {
			from, err := strconv.ParseInt(c.DefaultQuery("from", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid from format",
				})

				return
			}

			to, err := strconv.ParseInt(c.DefaultQuery("to", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid from format",
				})

				return
			}

			fromT := time.Unix(from, 0)
			toT := time.Unix(to, 0)

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent("uid not found at user get report size", "Unknown Error")
				return
			}

			total, err := nats.GetAvgStoredSizeByUidInDateRange(uid.(string), fromT, toT)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent("uid not found at user get report size", "Unknown Error")
				return
			}

			c.JSON(http.StatusOK, total)
		})

		userRoutesGroup.GET("/report/object-count", middlewares.UserAuthenticate, middlewares.ReqLogger("none", "B"), func(c *gin.Context) {
			from, err := strconv.ParseInt(c.DefaultQuery("from", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid from format",
				})

				return
			}

			to, err := strconv.ParseInt(c.DefaultQuery("to", "0"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid from format",
				})

				return
			}

			fromT := time.Unix(from, 0)
			toT := time.Unix(to, 0)

			uid, ok := c.Get("uid")
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent("uid not found at user get report object-count", "Unknown Error")
				return
			}

			total, err := nats.GetAvgObjectStoredByUidInDateRange(uid.(string), fromT, toT)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "something went wrong",
				})

				err = nats.SendErrorEvent("uid not found at user get report object-count", "Unknown Error")
				return
			}

			c.JSON(http.StatusOK, total)
		})
		//userRoutesGroup.GET("/bandwidth-report/signed/:key", middlewares.UserAuthenticate, middlewares.AuthReqCount, func(c *gin.Context) {
		//	k := c.Param("key")
		//	key, err := arango.FindKeyPairByPublic(k)
		//	if err != nil {
		//		if err, ok := err.(*models.ModelError); ok {
		//			if err.ErrType == models.NotFound || err.ErrType == models.DocumentNotFound {
		//				c.JSON(http.StatusNotFound, gin.H{
		//					"error": "key not found",
		//				})
		//
		//				return
		//			}
		//		}
		//
		//		c.JSON(http.StatusInternalServerError, gin.H{
		//			"error": "something went wrong",
		//		})
		//
		//		_ = nats.SendErrorEvent("error at bandwidth report: "+err.Error(), "db error")
		//		return
		//	}
		//
		//	uid, ok := c.Get("uid")
		//	if !ok {
		//		c.JSON(http.StatusInternalServerError, gin.H{
		//			"error": "something went wrong",
		//		})
		//
		//		_ = nats.SendErrorEvent("uid not found at user get bandwidth report", "unknown")
		//		return
		//	}
		//
		//	if key.GeneratorUid != uid {
		//		c.JSON(http.StatusForbidden, gin.H{
		//			"error": "not your key",
		//		})
		//
		//		return
		//	}
		//
		//	from, err := strconv.ParseInt(c.DefaultQuery("from", "0"), 10, 64)
		//	if err != nil {
		//		c.JSON(http.StatusBadRequest, gin.H{
		//			"error": "invalid from format",
		//		})
		//
		//		return
		//	}
		//
		//	to, err := strconv.ParseInt(c.DefaultQuery("to", "0"), 10, 64)
		//	if err != nil {
		//		c.JSON(http.StatusBadRequest, gin.H{
		//			"error": "invalid from format",
		//		})
		//
		//		return
		//	}
		//
		//	fromT := time.Unix(from, 0)
		//	toT := time.Unix(to, 0)
		//
		//	total, err := nats.SumBandwidthByDateRangeWithFrom(key.Public, fromT, toT)
		//	if err != nil {
		//		c.JSON(http.StatusInternalServerError, gin.H{
		//			"error": "something went wrong",
		//		})
		//
		//		_ = nats.SendErrorEvent("error at user get bandwidth report: "+err.Error(), "unknown")
		//		return
		//	}
		//
		//	c.JSON(http.StatusOK, total)
		//})
	}
}

func SendOTP(email string, otp string, expiredTime time.Time) error {
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

	return nats.SendEmailEvent(email, otp, expiredTime)
}
