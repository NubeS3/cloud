package middlewares

import (
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/gin-gonic/gin"
	"net/http"
)

func UnauthReqCount(c *gin.Context) {
	senderIp := readUserIP(c.Request)
	_ = nats.SendReqCountEvent("", "Req", c.Request.Method, senderIp, c.Request.URL.String())
}

func AuthReqCount(c *gin.Context) {
	senderIp := readUserIP(c.Request)
	uid, ok := c.Get("uid")
	if !ok {
		_ = nats.SendErrorEvent("uid not found at "+c.Request.Method+" "+c.Request.URL.String(), "abnormal")
		c.Abort()
		return
	}
	_ = nats.SendReqCountEvent(uid.(string), "Auth", c.Request.Method, senderIp, c.Request.URL.String())
}

func AccessKeyReqCount(c *gin.Context) {
	senderIp := readUserIP(c.Request)
	key, ok := c.Get("accessKey")
	if !ok {
		_ = nats.SendErrorEvent("key not found at "+c.Request.Method+" "+c.Request.URL.String(), "abnormal")
		c.Abort()
		return
	}
	_ = nats.SendReqCountEvent(key.(*arango.AccessKey).Id, "AccessKey", c.Request.Method, senderIp, c.Request.URL.String())
}

func SignedReqCount(c *gin.Context) {
	senderIp := readUserIP(c.Request)
	kp, ok := c.Get("keyPair")
	if !ok {
		_ = nats.SendErrorEvent("key pair not found at "+c.Request.Method+" "+c.Request.URL.String(), "abnormal")
		c.Abort()
		return
	}
	_ = nats.SendReqCountEvent(kp.(*arango.KeyPair).Public, "Signed", c.Request.Method, senderIp, c.Request.URL.String())
}

func readUserIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	return IPAddress
}
