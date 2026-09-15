package auth

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/login", GithubLogin)
	r.GET("/auth/callback", GithubCallback)
}
