package githubrepo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/sudhanshu042004/orcs/internal/repository"
)

func GetRepos(ctx *gin.Context) {
	email, ok := ctx.Get("email")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userEmail, ok := email.(string)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid email type"})
		return
	}

	userData, err := repository.FindUser(userEmail)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Could not find user in database"})
		return
	}

	if userData.RepoUrl == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "User does not have a repository URL"})
		return
	}

	// Create request to fetch repositories from GitHub
	client := &http.Client{}
	req, err := http.NewRequest("GET", userData.RepoUrl, nil)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request to GitHub"})
		return
	}
	req.Header.Add("User-Agent", "orcs-app")
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch repositories from GitHub"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		ctx.JSON(resp.StatusCode, gin.H{"error": "GitHub API returned error", "details": string(body)})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from GitHub"})
		return
	}

	// Unmarshal to verify JSON validity and then return it
	var repos interface{}
	if err := json.Unmarshal(body, &repos); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid response format from GitHub"})
		return
	}

	ctx.JSON(http.StatusOK, repos)
}

type DeployRequest struct {
	CloneUrl string `json:"clone_url" binding:"required"`
	Name     string `json:"name" binding:"required"`
}

func DeployRepo(ctx *gin.Context) {
	var req DeployRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	userIdVal, ok := ctx.Get("id")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userId, ok := userIdVal.(int64)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	// 1. Create a new deployment in the database with status 'draft'
	depId, err := repository.CreateDeployment(userId, req.Name, req.CloneUrl, "draft")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create deployment record: " + err.Error()})
		return
	}

	// Create target directory structure: uploads/cloned/<user_id>/<repo_name>
	targetDir := filepath.Join("uploads", "cloned", fmt.Sprintf("%d", userId), req.Name)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		repository.UpdateDeploymentStatus(depId, "failed", "")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory: " + err.Error()})
		return
	}

	// 2. Execute git clone inside isolated mount and PID namespaces
	cmd := exec.Command("git", "clone", req.CloneUrl, targetDir)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID,
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		// Update deployment status to failed
		repository.UpdateDeploymentStatus(depId, "failed", "")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to clone repository inside isolated namespace: " + err.Error(),
			"details": stderrBuf.String(),
		})
		return
	}

	// 3. Update deployment status to draft and save the url (targetDir path)
	if err := repository.UpdateDeploymentStatus(depId, "draft", targetDir); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update deployment status: " + err.Error()})
		return
	}

	// 4. Trigger asynchronous build execution
	BuildProjectAsync(depId, targetDir)

	ctx.JSON(http.StatusOK, gin.H{
		"message":       "Repository successfully cloned inside isolated namespace, build in progress",
		"deployment_id": depId,
		"status":        "draft",
		"path":          targetDir,
		"stdout":        stdoutBuf.String(),
	})
}

func GetDeployments(ctx *gin.Context) {
	userIdVal, ok := ctx.Get("id")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userId, ok := userIdVal.(int64)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	deployments, err := repository.GetDeployments(userId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve deployments: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, deployments)
}

func DeleteDeployment(ctx *gin.Context) {
	userIdVal, ok := ctx.Get("id")
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userId, ok := userIdVal.(int64)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	depIdStr := ctx.Param("id")
	depId, err := strconv.ParseInt(depIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deployment ID"})
		return
	}

	// 1. Find deployment to get its path
	deployment, err := repository.GetDeployment(depId, userId)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found: " + err.Error()})
		return
	}

	// 2. Safely remove directory if it exists and matches pattern
	// Target path is uploads/cloned/<user_id>/<name>
	expectedPrefix := filepath.Join("uploads", "cloned", fmt.Sprintf("%d", userId))
	if strings.HasPrefix(deployment.Url, expectedPrefix) && deployment.Url != expectedPrefix {
		if err := os.RemoveAll(deployment.Url); err != nil {
			fmt.Printf("Warning: failed to delete directory %s: %s\n", deployment.Url, err.Error())
		}
	}

	// 3. Delete deployment row
	if err := repository.DeleteDeployment(depId, userId); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete deployment record: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Deployment successfully deleted"})
}
