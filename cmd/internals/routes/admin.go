package routes

import (
	"github.com/NubeS3/cloud/cmd/internals/middlewares"
	"github.com/NubeS3/cloud/cmd/internals/routes/adminHandler"
	"github.com/gin-gonic/gin"
)

func AdminRoutes(route *gin.Engine) {
	adminRoutesGroup := route.Group("/admin")
	{
		adminRoutesGroup.POST("/signin", adminHandler.AdminSigninHandler)

		aar := adminRoutesGroup.Group("/auth", middlewares.AdminAuthenticate)
		{
			aar.GET("/test", adminHandler.AdminTestHandler)
			aar.POST("/mod", adminHandler.AdminCreateMod)
			aar.PATCH("/disable-mod", adminHandler.AdminModDisable)
			aar.PATCH("/ban-user", adminHandler.AdminBanUser)
			aar.GET("/err-log", adminHandler.AdminGetErrLog)
			aar.GET("/err-log/type", adminHandler.AdminGetErrLogByType)
			aar.GET("/err-log/date", adminHandler.AdminGetErrLogByDate)
			aar.GET("/bucket-log", adminHandler.AdminGetBucketLog)
			aar.GET("/bucket-log/type", adminHandler.AdminGetBucketLogByType)
			aar.GET("/bucket-log/date", adminHandler.AdminGetBucketLogByDate)
			aar.GET("/user-log", adminHandler.AdminGetUserLog)
			aar.GET("/user-log/type", adminHandler.AdminGetUserLogByType)
			aar.GET("/user-log/date", adminHandler.AdminGetUserLogByDate)
			aar.GET("/accessKey-log", adminHandler.AdminGetAccessKeyLog)
			aar.GET("/accessKey-log/type", adminHandler.AdminGetAccessKeyLogByType)
			aar.GET("/accessKey-log/date", adminHandler.AdminGetAccessKeyLogByDate)
			aar.GET("/keyPair-log", adminHandler.AdminGetKeyPairLog)
			aar.GET("/keyPair-log/type", adminHandler.AdminGetKeyPairLogByType)
			aar.GET("/keyPair-log/date", adminHandler.AdminGetKeyPairLogByDate)
			aar.GET("/req-log/auth", adminHandler.AdminGetAuthReqLog)
			aar.GET("/req-log/accessKey", adminHandler.AdminGetAccessKeyReqLog)
			aar.GET("/req-log/signed", adminHandler.AdminGetSignedReqLog)
			aar.GET("/req-log/count/accessKey", adminHandler.AdminCountAccessKeyReqLog)
			aar.GET("/req-log/count/signed", adminHandler.AdminCountSignedReqLog)
			aar.GET("/users-list", adminHandler.AdminGetUsers)
			aar.GET("/admins-list", adminHandler.AdminGetMods)
			aar.GET("/accessKey/:bucket_id", adminHandler.AdminGetAccessKeyByBid)
			aar.GET("/keyPair/:bucket_id", adminHandler.AdminGetKeyPairByBid)
			aar.GET("/buckets/:uid", adminHandler.AdminGetBucketByUid)
			aar.GET("/buckets", adminHandler.AdminGetAllBucket)

			aar.GET("/bandwidth-report/user/:uid", adminHandler.AdminGetUidTotalBandwidth)
			aar.GET("/bandwidth-report/bucket/:bid", adminHandler.AdminGetBidTotalBandwidth)
			aar.GET("/bandwidth-report/access-key/:key", adminHandler.AdminGetAkTotalBandwidth)
			aar.GET("/bandwidth-report/signed/:key", adminHandler.AdminGetSignedTotalBandwidth)
		}
	}
}
