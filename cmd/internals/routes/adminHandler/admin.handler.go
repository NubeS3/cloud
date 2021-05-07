package adminHandler

import (
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

		return
	}

	c.JSON(http.StatusOK, resAdmin)
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

		return
	}

	c.JSON(http.StatusOK, admin)
}

func AdminBanUser(c *gin.Context) {
	type toggleReq struct {
		Username string `json:"username" binding:"required"`
		IsBan    *bool  `json:"is_ban" binding:"required"`
	}

	var req toggleReq
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, err := arango.FindUserByUsername(req.Username)
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

		return
	}

	user, err = arango.UpdateBanStatus(user.Id, *req.IsBan)

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

		return
	}

	c.JSON(http.StatusOK, users)
}

func AdminGetMods(c *gin.Context) {
	a, ok := c.Get("admin")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "admin not found.",
		})
	}
	admin := a.(ultis.AdminClaims)

	if admin.AdminType != 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "You are not allowed",
		})
		return
	}

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

	users, err := arango.GetAllMods(int(offset), int(limit))

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

		return
	}

	c.JSON(http.StatusOK, users)
}
