package middlewares

import (
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/NubeS3/cloud/cmd/internals/ultis"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

func CheckSigned(c *gin.Context) {
	alg := c.Query("ALG")
	pub := c.Query("PUB")
	sig := c.Query("SIG")
	sExp := c.Query("EXP")

	exp, err := strconv.ParseInt(sExp, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid expiration date",
		})

		c.Abort()
		return
	}

	expT := time.Unix(exp, 0)
	if expT.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "signed url expired",
		})

		c.Abort()
		return
	}

	hashFunc, err := ultis.GetHashFunc(alg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})

		c.Abort()
		return
	}

	kp, err := arango.FindKeyPairByPublic(pub)
	if err != nil {
		if err, ok := err.(*models.ModelError); ok {
			if err.ErrType == models.DocumentNotFound {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "pub mismatch",
				})

				c.Abort()
				return
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "something went wrong",
		})

		_ = nats.SendErrorEvent("signed authenticate error: "+err.Error(), "Unknown Error")
		c.Abort()
		return
	}

	if kp.ExpiredDate.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "signed url expired",
		})

		c.Abort()
		return
	}

	expStr := strconv.FormatInt(exp, 10)
	method := c.Request.Method
	host := "https://" + c.Request.Host
	path := c.Request.URL.Path

	pHash, err := hashFunc(kp.Private + method + host + path + expStr)
	if pHash == sig {
		c.Set("keyPair", kp)
		c.Next()
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid signature",
		})

		c.Abort()
		return
	}
}
