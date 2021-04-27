package adminHandler

import (
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	scrypt "github.com/elithrar/simple-scrypt"
	"github.com/gin-gonic/gin"
	"net/http"
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
