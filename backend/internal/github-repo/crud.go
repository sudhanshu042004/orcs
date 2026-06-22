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

type CloneRequest struct {
	CloneUrl string `json:"clone_url" binding:"required"`
	Name     string `json:"name" binding:"required"`
}

func CloneRepo(ctx *gin.Context) {
	var req CloneRequest
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

	// Create target directory structure: uploads/cloned/<user_id>/<repo_name>
	targetDir := filepath.Join("uploads", "cloned", fmt.Sprintf("%d", userId), req.Name)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory: " + err.Error()})
		return
	}

	// Execute git clone inside isolated mount and PID namespaces
	cmd := exec.Command("git", "clone", req.CloneUrl, targetDir)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID,
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to clone repository inside isolated namespace: " + err.Error(),
			"details": stderrBuf.String(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Repository successfully cloned inside isolated namespace",
		"path":    targetDir,
		"stdout":  stdoutBuf.String(),
	})
}
