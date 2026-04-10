package main

import (
	"github.com/gin-gonic/gin"
	"github.com/sudhanshu042004/orcs/database"
	"github.com/sudhanshu042004/orcs/internal/auth"
	"github.com/sudhanshu042004/orcs/internal/middleware"
	"github.com/sudhanshu042004/orcs/internal/user"
	"github.com/sudhanshu042004/orcs/pkg/config"
	"golang.org/x/oauth2"
)

type App struct {
	config *oauth2.Config
}

func main() {
	router := gin.Default()
	database.ConnectDb()

	router.Use(config.CorsMiddleware())

	//health
	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"success": "true",
		})
	})
	//routes
	config.GithubConfig()
	router.GET("/login", auth.GithubLogin)
	router.GET("/auth/callback", auth.GithubCallback)

	router.Use(middleware.AuthRequired())
	router.GET("/api/user", user.GetUser)
	router.Run(":3000")
}
