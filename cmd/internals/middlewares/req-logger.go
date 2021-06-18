package middlewares

import (
	"errors"
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/gin-gonic/gin"
)

func ReqLogger(reqType, class string) func(ctx *gin.Context) {
	switch reqType {
	case "unauth":
		return func(ctx *gin.Context) {
			senderIp := readUserIP(ctx.Request)
			err := nats.SendReqCountEvent("", "Req", ctx.Request.Method, senderIp, ctx.Request.URL.String(), "")
			if err != nil {
				println(err)
			}
		}
	case "auth":
		return func(ctx *gin.Context) {
			senderIp := readUserIP(ctx.Request)
			uid, ok := ctx.Get("uid")
			if !ok {
				_ = nats.SendErrorEvent("uid not found at "+ctx.Request.Method+" "+ctx.Request.URL.String(), "abnormal")
				ctx.Abort()
				return
			}
			err := nats.SendReqCountEvent(uid.(string), "Auth", ctx.Request.Method, senderIp, ctx.Request.URL.String(), class)
			if err != nil {
				println(err)
			}
		}
	case "key":
		return func(ctx *gin.Context) {
			senderIp := readUserIP(ctx.Request)
			key, ok := ctx.Get("key")
			if !ok {
				_ = nats.SendErrorEvent("key not found at "+ctx.Request.Method+" "+ctx.Request.URL.String(), "abnormal")
				ctx.Abort()
				return
			}
			err := nats.SendReqCountEvent(key.(*arango.AccessKey).Id, "AccessKey", ctx.Request.Method, senderIp, ctx.Request.URL.String(), class)
			if err != nil {
				println(err)
			}
		}
	}

	panic(errors.New("NO LOGGER FOUND FOR: " + reqType + " with class: " + class))
}
