package adminHandler

import (
	"github.com/NubeS3/cloud/cmd/internals/models/seaweedfs"
	"net/http"
	"strconv"
	"time"

	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	scrypt "github.com/elithrar/simple-scrypt"
	"github.com/gin-gonic/gin"
)

func AdminSigninHandler(c *gin.Context) {
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

	admin, err := arango.FindAdminByUsername(curSigninUser.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid username",
		})
		return
	}

	err = scrypt.CompareHashAndPassword([]byte(admin.Pass), []byte(curSigninUser.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	if admin.IsDisable {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "account disabled",
		})
		return
	}

	accessToken, err := ultis.CreateAdminToken(admin.Id, int(admin.AType))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		_ = nats.SendErrorEvent(err.Error()+" at user route/sign in/access token",
			"Token Error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accessToken": accessToken,
	})
}

func AdminTestHandler(c *gin.Context) {
	c.JSON(http.StatusOK, "Admin Ok")
}

func AdminCreateMod(c *gin.Context) {
	type admin struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	var newAdmin admin
	if err := c.ShouldBind(&newAdmin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if ok, err := ultis.ValidateUsername(newAdmin.Username); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "something went wrong",
		})

		err = nats.SendErrorEvent(err.Error(), "Validate Error")
		return
	} else if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Username must be 8-24 characters, does not start or end with _ or ., does not contain __, _., ._, ..",
		})

		return
	}

	if ok, err := ultis.ValidatePassword(newAdmin.Password); err != nil {
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

	resAdmin, err := arango.CreateAdmin(newAdmin.Username, newAdmin.Password, arango.ModAdmin)
	if err != nil {
		if err, ok := err.(*models.ModelError); ok {
			if err.ErrType == models.Duplicated {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, resAdmin)
}

func AdminCreateUser(c *gin.Context) {
	type user struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	var newUser user
	if err := c.ShouldBind(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if ok, err := ultis.ValidateEmail(newUser.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "something went wrong",
		})

		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	} else if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email must be <something>@<something.com>",
		})

		return
	}

	if ok, err := ultis.ValidatePassword(newUser.Password); err != nil {
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

	resUser, err := arango.SaveUser(newUser.Password, newUser.Email)
	if err != nil {
		if err, ok := err.(*models.ModelError); ok {
			if err.ErrType == models.Duplicated {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	if err := arango.GenerateRfToken(resUser.Id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	otp, err := arango.GenerateOTP(resUser.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		_ = nats.SendErrorEvent(err.Error()+" at user route/sign up/generate otp",
			"Db Error")
		return
	}

	if err := nats.SendEmailEvent(resUser.Email, otp.Otp, otp.ExpiredTime); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		_ = nats.SendErrorEvent(err.Error()+" at user route/sign up/send otp",
			"OTP Failed")
		return
	}

	c.JSON(http.StatusOK, resUser)
}

func AdminModDisable(c *gin.Context) {
	type toggleReq struct {
		Username string `json:"username" binding:"required"`
		Disable  *bool  `json:"disable" binding:"required"`
	}

	var req toggleReq
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	cAdmin, ok := c.Get("admin")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "current admin not found",
		})
		err := nats.SendErrorEvent("admin not found at admin/auth/disable-mod",
			"Unknown Error")
		print(err)
		return
	}

	currentAdmin := cAdmin.(*arango.Admin)
	if currentAdmin.Username == req.Username {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "self-disable is forbidden, please contact the root admin",
		})

		return
	}

	admin, err := arango.ToggleAdmin(req.Username, *req.Disable)
	if err != nil {
		if err, ok := err.(*models.ModelError); ok {
			if err.ErrType == models.Duplicated {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, admin)
}

func AdminBanUser(c *gin.Context) {
	type toggleReq struct {
		Email string `json:"email" binding:"required"`
		IsBan *bool  `json:"is_ban" binding:"required"`
	}

	var req toggleReq
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, err := arango.FindUserByEmail(req.Email)
	if err != nil {
		if err, ok := err.(*models.ModelError); ok {
			if err.ErrType == models.Duplicated {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	user, err = arango.UpdateBanStatus(user.Id, *req.IsBan)

	c.JSON(http.StatusOK, user)
}

func AdminActiveUser(c *gin.Context) {
	type toggleReq struct {
		Email    string `json:"email" binding:"required"`
		IsActive *bool  `json:"is_active" binding:"required"`
	}

	var req toggleReq
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, err := arango.FindUserByEmail(req.Email)
	if err != nil {
		if err, ok := err.(*models.ModelError); ok {
			if err.ErrType == models.Duplicated {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	err = arango.UpdateActive(user.Email, *req.IsActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})

		return
	}
	user.IsActive = true

	c.JSON(http.StatusOK, user)
}

func AdminGetErrLog(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	logs, err := nats.GetErrLog(int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}
	c.JSON(http.StatusOK, logs)
}

func AdminGetErrLogByType(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	t := c.DefaultQuery("type", "")

	logs, err := nats.GetErrLogByType(t, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, logs)
}

func AdminGetErrLogByDate(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
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

	logs, err := nats.GetErrLogByDate(fromT, toT, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}
	c.JSON(http.StatusOK, logs)
}

func AdminGetBucketLog(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	logs, err := nats.GetBucketLog(int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}
	c.JSON(http.StatusOK, logs)
}

func AdminGetBucketLogByType(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	t := c.DefaultQuery("type", "")

	logs, err := nats.GetBucketLogByType(t, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, logs)
}

func AdminGetBucketLogByDate(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
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

	logs, err := nats.GetBucketLogByDate(fromT, toT, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}
	c.JSON(http.StatusOK, logs)
}

func AdminGetUserLog(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	logs, err := nats.GetUserLog(int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}
	c.JSON(http.StatusOK, logs)
}

func AdminGetUserLogByType(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	t := c.DefaultQuery("type", "")

	logs, err := nats.GetUserLogByType(t, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, logs)
}

func AdminGetUserLogByDate(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
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

	logs, err := nats.GetUserLogByDate(fromT, toT, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}
	c.JSON(http.StatusOK, logs)
}

func AdminGetAccessKeyLog(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	logs, err := nats.GetAccessKeyLog(int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}
	c.JSON(http.StatusOK, logs)
}

func AdminGetAccessKeyLogByType(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	t := c.DefaultQuery("type", "")

	logs, err := nats.GetAccessKeyLogByType(t, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, logs)
}

func AdminGetAccessKeyLogByDate(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
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

	logs, err := nats.GetAccessKeyLogByDate(fromT, toT, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}
	c.JSON(http.StatusOK, logs)
}

func AdminGetKeyPairLog(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	logs, err := nats.GetKeyPairLog(int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}
	c.JSON(http.StatusOK, logs)
}

func AdminGetKeyPairLogByType(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	t := c.DefaultQuery("type", "")

	logs, err := nats.GetKeyPairLogByType(t, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, logs)
}

func AdminGetKeyPairLogByDate(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
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

	logs, err := nats.GetKeyPairLogByDate(fromT, toT, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}
	c.JSON(http.StatusOK, logs)
}

func AdminGetAuthReqLog(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
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

	uid := c.DefaultQuery("uid", "")

	res, err := nats.ReadAuthReqCountByDateRange(uid, fromT, toT, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, res)
}

func AdminGetAccessKeyReqLog(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
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
			"error": "invalid to format",
		})

		return
	}

	fromT := time.Unix(from, 0)
	toT := time.Unix(to, 0)

	key := c.DefaultQuery("key", "")

	res, err := nats.ReadAccessKeyReqCountByDateRange(key, fromT, toT, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, res)
}

func AdminCountAuthReqLog(c *gin.Context) {
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

	uid := c.DefaultQuery("uid", "")

	res, err := nats.CountByClass(uid, fromT, toT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, res)
}

func AdminCountAccessKeyReqLog(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
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

	key := c.DefaultQuery("key", "")

	res, err := nats.CountAccessKeyReqCountByDateRange(key, fromT, toT, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, res)
}

func AdminGetSignedReqLog(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
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

	public := c.DefaultQuery("public", "")

	res, err := nats.ReadSignedReqCountByDateRange(public, fromT, toT, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, res)
}

func AdminCountSignedReqLog(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
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

	public := c.DefaultQuery("public", "")

	res, err := nats.CountSignedReqCountByDateRange(public, fromT, toT, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, res)
}

func AdminGetAccessKeyByBid(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	bucketId := c.Param("bucket_id")
	res, err := arango.FindAccessKeyByBid(bucketId, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}
	c.JSON(http.StatusOK, res)
}

func AdminGetKeyPairByBid(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	bucketId := c.Param("bucket_id")
	res, err := arango.FindKeyByBid(bucketId, int(limit), int(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}
	c.JSON(http.StatusOK, res)
}

func AdminGetUsers(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	users, err := arango.GetAllUser(int(offset), int(limit))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, users)
}

func AdminGetNonBannedUsers(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	users, err := arango.GetAllNonBannedUser(int(offset), int(limit))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, users)
}

func AdminGetBannedUsers(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	users, err := arango.GetAllBannedUser(int(offset), int(limit))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, users)
}

func AdminGetMods(c *gin.Context) {
	//a, ok := c.Get("admin")
	//if !ok {
	//	c.JSON(http.StatusBadRequest, gin.H{
	//		"error": "admin not found.",
	//	})
	//	return
	//}
	//admin := a.(ultis.AdminClaims)

	//if admin.AdminType != 0 {
	//	c.JSON(http.StatusBadRequest, gin.H{
	//		"error": "You are not allowed",
	//	})
	//	return
	//}

	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	admins, err := arango.GetAllMods(int(offset), int(limit))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, admins)
}

func AdminGetUidTotalBandwidth(c *gin.Context) {
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

	uid := c.Param("uid")

	user, err := arango.FindUserById(uid)
	if err != nil {
		if err, ok := err.(*models.ModelError); ok {
			if err.ErrType == models.NotFound || err.ErrType == models.DocumentNotFound {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "user not found",
				})

				return
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	res, err := nats.SumBandwidthByDateRangeWithUid(user.Id, fromT, toT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, res)
}

func AdminGetBidTotalBandwidth(c *gin.Context) {
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

	bid := c.Param("bid")

	bucket, err := arango.FindBucketById(bid)
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
			"error": err.Error(),
		})

		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	res, err := nats.SumBandwidthByDateRangeWithBucketId(bucket.Id, fromT, toT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, res)
}

func AdminGetAkTotalBandwidth(c *gin.Context) {
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

	k := c.Param("id")

	key, err := arango.FindAccessKeyById(k)
	if err != nil {
		if err, ok := err.(*models.ModelError); ok {
			if err.ErrType == models.NotFound || err.ErrType == models.DocumentNotFound {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "key not found",
				})

				return
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	res, err := nats.SumBandwidthByDateRangeWithFrom(key.Id, fromT, toT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, res)
}

func AdminGetSignedTotalBandwidth(c *gin.Context) {
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

	k := c.Param("key")

	key, err := arango.FindKeyPairByPublic(k)
	if err != nil {
		if err, ok := err.(*models.ModelError); ok {
			if err.ErrType == models.NotFound || err.ErrType == models.DocumentNotFound {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "key not found",
				})

				return
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	res, err := nats.SumBandwidthByDateRangeWithFrom(key.Public, fromT, toT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, res)
}

func AdminGetBucketByUid(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	uid := c.Param("uid")

	buckets, err := arango.FindBucketByUid(uid, limit, offset)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, buckets)
}

func AdminGetAllBucket(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit format",
		})

		return
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset format",
		})

		return
	}

	buckets, err := arango.FindAllBucket(limit, offset)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, buckets)
}

func AdminGetSystemBandwidth(c *gin.Context) {
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

	res, err := nats.SumBandwidthByDateRangeWithUid("", fromT, toT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, res)
}

func AdminCountAllSystemReqLog(c *gin.Context) {
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

	res, err := nats.CountByClass("", fromT, toT)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		err = nats.SendErrorEvent(err.Error(), "Db Error")
		return
	}

	c.JSON(http.StatusOK, res)
}

func SeaweedInfo(c *gin.Context) {
	systemStatus, err := seaweedfs.SeaweedInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})

		return
	}

	clusterStatus, err := seaweedfs.SeaweedClusterStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"system":  systemStatus,
		"cluster": clusterStatus,
	})
}

func ArangoInfo(c *gin.Context) {
	systemStatus, err := arango.ArangoInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})

		return
	}

	clusterStatus, err := arango.ArangoClusterInfo()
	if err != nil {
		//c.JSON(http.StatusInternalServerError, gin.H{
		//	"error": err.Error(),
		//})
		//
		//return

		c.JSON(http.StatusOK, gin.H{
			"system":  systemStatus,
			"cluster": nil,
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"system":  systemStatus,
		"cluster": clusterStatus,
	})
}
