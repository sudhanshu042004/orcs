package main

import (
	"github.com/gin-gonic/gin"
	"github.com/sudhanshu042004/orcs/database"
	"github.com/sudhanshu042004/orcs/internal/auth"
	githubrepo "github.com/sudhanshu042004/orcs/internal/github-repo"
	"github.com/sudhanshu042004/orcs/internal/middleware"
	"github.com/sudhanshu042004/orcs/internal/site"
	"github.com/sudhanshu042004/orcs/internal/user"
	"github.com/sudhanshu042004/orcs/internal/worker"
	"github.com/sudhanshu042004/orcs/pkg/config"
	"golang.org/x/oauth2"
)

type App struct {
	config *oauth2.Config
}

// buildWorkers is how many deployments can be built at the same time.
const buildWorkers = 2

func main() {
	router := gin.Default()
	database.ConnectDb()

	// Deployed sites are served on their own hosts (<id>.localhost:3000), so this runs
	// ahead of the API routes and takes those requests before any of them match
	router.Use(site.Middleware())
	router.Use(config.CorsMiddleware())

	//health
	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"success": "true",
		})
	})

	config.GithubConfig()

	// Build-job queue: workers pick deployments up from here, and anything still queued
	// from a previous run is put back on the queue
	worker.Start(buildWorkers)

	// Auth routes
	auth.RegisterRoutes(router.Group("/"))

	// Protected routes
	protected := router.Group("/")
	protected.Use(middleware.AuthRequired())

	user.RegisterRoutes(protected.Group("/api"))
	githubrepo.RegisterRoutes(protected)

	router.Run(":3000")
}
