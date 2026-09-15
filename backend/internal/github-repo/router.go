package githubrepo

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/repos", GetRepos)
	r.GET("/stacks", GetStacks)
	r.GET("/deployments", GetDeployments)
	r.POST("/deployments", CreateDeployment)
	r.DELETE("/deployments/:id", DeleteDeployment)
	r.GET("/deployments/:id/stream", StreamDeploymentLogs)
}
