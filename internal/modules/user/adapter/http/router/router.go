package router

import (
	"github.com/gin-gonic/gin"

	"github.com/maguowei/gotobeta/internal/modules/user/adapter/http/handler"
)

// RegisterRoutes 注册用户认证路由。
// rateLimit 为可选限流中间件（nil 表示不限流），作用于易被爆破/撞库的凭据类端点。
func RegisterRoutes(group *gin.RouterGroup, authHandler *handler.AuthHandler, userHandler *handler.UserHandler, authMiddleware gin.HandlerFunc, rateLimit gin.HandlerFunc) {
	// 凭据敏感端点统一挂限流，抵御在线密码爆破、验证码/重置 token 暴力尝试。
	limited := group.Group("")
	if rateLimit != nil {
		limited.Use(rateLimit)
	}
	limited.POST("/auth/register", authHandler.Register)
	limited.POST("/auth/login", authHandler.Login)
	limited.POST("/auth/refresh", authHandler.Refresh)
	limited.POST("/auth/password/forgot", authHandler.ForgotPassword)
	limited.POST("/auth/password/reset", authHandler.ResetPassword)
	limited.POST("/auth/email/verify", authHandler.VerifyEmail)
	limited.POST("/auth/oauth/token", authHandler.OAuthToken)

	// OAuth 浏览器跳转流程不限流，避免误伤正常登录跳转。
	group.GET("/auth/oauth/:provider/start", authHandler.StartOAuth)
	group.GET("/auth/oauth/:provider/callback", authHandler.OAuthCallback)

	protected := group.Group("")
	protected.Use(authMiddleware)
	protected.POST("/auth/logout", authHandler.Logout)
	protected.POST("/users/me/email/verification", authHandler.SendEmailVerification)
	protected.GET("/users/me", userHandler.Me)
	protected.PATCH("/users/me", userHandler.UpdateMe)
	protected.PUT("/users/me/password", userHandler.ChangePassword)
	protected.GET("/users/me/identities", userHandler.ListIdentities)
	protected.DELETE("/users/me/identities/:provider", userHandler.UnlinkIdentity)
}
