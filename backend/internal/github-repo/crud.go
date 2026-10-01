package githubrepo

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sudhanshu042004/orcs/internal/container"
	"github.com/sudhanshu042004/orcs/internal/queue"
	"github.com/sudhanshu042004/orcs/internal/repository"
	"github.com/sudhanshu042004/orcs/internal/s3"
	"github.com/sudhanshu042004/orcs/internal/site"
	"github.com/sudhanshu042004/orcs/internal/stack"
	"github.com/sudhanshu042004/orcs/internal/worker"
	"github.com/sudhanshu042004/orcs/types"
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

// CreateDeploymentRequest is the payload produced by the deploy configuration form.
type CreateDeploymentRequest struct {
	Name       string `json:"name" binding:"required"`
	Stack      string `json:"stack" binding:"required"`
	RepoUrl    string `json:"repo_url" binding:"required"`
	InstallCmd string `json:"install_cmd"`
	BuildCmd   string `json:"build_cmd"`
	RunCmd     string `json:"run_cmd"`
}

// GetStacks lists the project types the deploy form can offer, including the ones that are
// not buildable yet so the UI can show them as coming soon.
func GetStacks(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, stack.All())
}

// CreateDeployment records a deployment in the 'queued' state and hands a build job to the
// queue. The build itself happens later, on a worker.
func CreateDeployment(ctx *gin.Context) {
	var req CreateDeploymentRequest
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

	name := strings.TrimSpace(req.Name)
	if name == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Deployment name is required"})
		return
	}

	target, ok := stack.Get(req.Stack)
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Unknown stack: " + req.Stack})
		return
	}
	if !target.Enabled {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": target.Label + " projects are not supported yet"})
		return
	}

	repoUrl := strings.TrimSpace(req.RepoUrl)
	if parsed, err := url.Parse(repoUrl); err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Repository URL must be a http(s) git URL"})
		return
	}

	// A locked stack (React today) always builds with its own commands; everything else
	// takes the commands the user configured on the deploy form
	installCmd, buildCmd, runCmd := strings.TrimSpace(req.InstallCmd), strings.TrimSpace(req.BuildCmd), strings.TrimSpace(req.RunCmd)
	if target.Locked {
		installCmd, buildCmd, runCmd = target.InstallCmd, target.BuildCmd, target.RunCmd
	}
	if installCmd == "" || buildCmd == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Install and build commands are required"})
		return
	}
	// A build that only publishes files is finished once they are uploaded; one that leaves
	// an app running has to be told how to start it
	if target.Kind == stack.KindDynamic && runCmd == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": target.Label + " projects need a run command"})
		return
	}

	depId, err := repository.CreateDeployment(userId, types.Deployment{
		Name:       name,
		RepoUrl:    repoUrl,
		Stack:      target.Key,
		InstallCmd: installCmd,
		BuildCmd:   buildCmd,
		RunCmd:     runCmd,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create deployment record: " + err.Error()})
		return
	}

	job := queue.Job{
		DeploymentId: depId,
		Stack:        target.Key,
		RepoUrl:      repoUrl,
		InstallCmd:   installCmd,
		BuildCmd:     buildCmd,
		RunCmd:       runCmd,
	}
	if err := queue.Default().Enqueue(job); err != nil {
		_ = repository.UpdateDeploymentStatus(depId, "failed", "")
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":       "Deployment queued",
		"deployment_id": depId,
		"status":        "queued",
	})
}

// streamPoll is how often a running build's log file and status are re-checked.
const streamPoll = 500 * time.Millisecond

// logTail follows one deployment's build log. The worker writes every line of a build to
// that file as it happens, so tailing it both replays a finished build and follows a
// running one. Nothing here talks to docker, so there is no gap between a job being picked
// up and its container existing in which lines can be missed.
type logTail struct {
	path    string
	file    *os.File
	offset  int64
	partial []byte
}

// next returns whatever lines were written since the last call. A line that is still being
// written is held back until its newline arrives.
func (t *logTail) next() []string {
	if t.file == nil {
		file, err := os.Open(t.path)
		if err != nil {
			// The worker has not started writing the log yet
			return nil
		}
		t.file = file
	}

	// A rebuild truncates the log, so start it over rather than waiting past its end
	if info, err := t.file.Stat(); err == nil && info.Size() < t.offset {
		if _, err := t.file.Seek(0, io.SeekStart); err == nil {
			t.offset = 0
			t.partial = nil
		}
	}

	var lines []string
	buf := make([]byte, 32*1024)
	for {
		n, err := t.file.Read(buf)
		if n > 0 {
			t.offset += int64(n)
			t.partial = append(t.partial, buf[:n]...)
			for {
				i := bytes.IndexByte(t.partial, '\n')
				if i < 0 {
					break
				}
				lines = append(lines, strings.TrimRight(string(t.partial[:i]), "\r"))
				t.partial = t.partial[i+1:]
			}
		}
		// A short read means the log is drained for now
		if err != nil || n == 0 {
			return lines
		}
	}
}

func (t *logTail) close() {
	if t.file != nil {
		_ = t.file.Close()
	}
}

func StreamDeploymentLogs(ctx *gin.Context) {
	depIdStr := ctx.Param("id")
	depId, err := strconv.ParseInt(depIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deployment ID"})
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

	dep, err := repository.GetDeployment(depId, userId)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}

	ctx.Header("Content-Type", "application/x-ndjson")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	// Proxies that buffer would hold the build log back until the build finished
	ctx.Header("X-Accel-Buffering", "no")

	flusher, ok := ctx.Writer.(http.Flusher)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	// A write that fails means the client is gone, and the stream should stop
	writeEvent := func(evt interface{}) bool {
		if err := json.NewEncoder(ctx.Writer).Encode(evt); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	writeStatus := func(d types.Deployment) bool {
		return writeEvent(gin.H{
			"type":     "status",
			"status":   d.Status,
			"url":      d.Url,
			"name":     d.Name,
			"repo_url": d.RepoUrl,
		})
	}

	tail := &logTail{path: filepath.Join("uploads", "logs", fmt.Sprintf("%d.log", depId))}
	defer tail.close()

	drain := func() bool {
		for _, line := range tail.next() {
			if !writeEvent(gin.H{"type": "log", "text": line}) {
				return false
			}
		}
		return true
	}

	if !writeStatus(dep) {
		return
	}

	for {
		if !drain() {
			return
		}

		// Once a build has finished, its whole log has been replayed and there is
		// nothing further to follow
		if dep.Status == worker.StatusDeployed || dep.Status == worker.StatusFailed {
			return
		}

		select {
		case <-ctx.Request.Context().Done():
			return
		case <-time.After(streamPoll):
		}

		refreshed, err := repository.GetDeployment(depId, userId)
		if err != nil {
			return
		}
		if refreshed.Status != dep.Status {
			// Send the lines that go with the old status before announcing the new one
			if !drain() {
				return
			}
			dep = refreshed
			if !writeStatus(dep) {
				return
			}
		}
	}
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

	// 1. Find deployment and make sure it belongs to the caller
	deployment, err := repository.GetDeployment(depId, userId)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found: " + err.Error()})
		return
	}

	// 2. Remove the container. For a dynamic deployment this is the running app, so the
	// site handler is told to forget it rather than keep proxying to a dead port
	site.Forget(depId)
	containerId, err := repository.GetDeploymentContainer(depId)
	if err != nil && err != sql.ErrNoRows {
		fmt.Printf("Warning: failed to look up container for deployment %d: %s\n", depId, err.Error())
	} else if containerId != "" {
		if err := container.RemoveContainer(containerId); err != nil {
			fmt.Printf("Warning: failed to remove container %s: %s\n", containerId, err.Error())
		}
	}

	// 3. Builds happen entirely inside containers now, but deployments created before that
	// may still own a checkout under uploads/cloned/<user_id>/<name>
	legacyPrefix := filepath.Join("uploads", "cloned", fmt.Sprintf("%d", userId))
	legacyDir := filepath.Join(legacyPrefix, filepath.Base(deployment.Name))
	if strings.HasPrefix(deployment.Url, legacyPrefix) {
		legacyDir = filepath.Clean(deployment.Url)
	}
	if legacyDir != legacyPrefix {
		if err := os.RemoveAll(legacyDir); err != nil {
			fmt.Printf("Warning: failed to delete directory %s: %s\n", legacyDir, err.Error())
		}
	}

	// 4. Remove the build log
	logPath := filepath.Join("uploads", "logs", fmt.Sprintf("%d.log", depId))
	if err := os.Remove(logPath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: failed to delete log file %s: %s\n", logPath, err.Error())
	}

	// 5. Remove the published static files from S3 (best effort)
	s3Prefix := fmt.Sprintf("deployments/%d", depId)
	if err := s3.DeletePrefix(ctx.Request.Context(), s3Prefix); err != nil {
		fmt.Printf("Warning: failed to delete S3 objects for %s: %s\n", s3Prefix, err.Error())
	}

	// 6. Delete deployment row (deployment_containers rows cascade)
	if err := repository.DeleteDeployment(depId, userId); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete deployment record: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Deployment successfully deleted"})
}
