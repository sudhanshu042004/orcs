package files

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	githubrepo "github.com/sudhanshu042004/orcs/internal/github-repo"
	"github.com/sudhanshu042004/orcs/internal/repository"
)

type NodeType string

const (
	FileNode   NodeType = "file"
	FolderNode NodeType = "folder"
)

func findNode(nodes []Node, name string) Node {
	for _, node := range nodes {
		if node.Folder.Name == name {
			return node
		} else {
			return Node{}
		}
	}
	return Node{}
}

type File struct {
	Name    string
	Content []byte
}

type Folder struct {
	Name     string
	Children []Node
}

type Node struct {
	Type   NodeType // "file" or "folder"
	File   *File
	Folder *Folder
}

func saveFile(key string, fileHeader *multipart.FileHeader, parentFolder string) error {
	// key = "folder1/subfolder/file.txt"
	// Strip the original root folder name (e.g. folder1) to save directly under parentFolder
	relPath := key
	if idx := strings.IndexAny(key, "/\\"); idx != -1 {
		relPath = key[idx+1:]
	}
	fullPath := filepath.Join(parentFolder, relPath)

	// Create all folders
	dir := filepath.Dir(fullPath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	// Open uploaded file
	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy content
	_, err = io.Copy(dst, src)
	return err
}

const maxUploadSize = 100 << 20

func FileUploadHandler(c *gin.Context) {
	userIdVal, ok := c.Get("id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userId, ok := userIdVal.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	if err := c.Request.ParseMultipartForm(maxUploadSize); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form: " + err.Error()})
		return
	}

	projectName := c.PostForm("projectName")
	if projectName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "projectName is required"})
		return
	}

	form := c.Request.MultipartForm
	if form == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty form"})
		return
	}

	// Determine original folder name from first file path key (e.g. folderName/index.html)
	folderName := "Local Folder"
	for key := range form.File {
		parts := strings.Split(key, "/")
		if len(parts) > 0 && parts[0] != "" {
			folderName = parts[0]
			break
		}
	}

	// Create target directory structure: uploads/cloned/<user_id>/<projectName>
	targetDir := filepath.Join("uploads", "cloned", fmt.Sprintf("%d", userId), projectName)
	info, err := os.Stat(targetDir)
	if err == nil {
		if !info.IsDir() {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Path exists but is a file, not a directory."})
			return
		}
	} else if errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(targetDir, 0755)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create directory: " + err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stat directory: " + err.Error()})
		return
	}

	// Save all uploaded files to targetDir
	for key, headers := range form.File {
		for _, fileHeader := range headers {
			err := saveFile(key, fileHeader, targetDir)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}

	// Create a new deployment in the database with status 'draft'
	depId, err := repository.CreateDeployment(userId, projectName, folderName, "draft")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create deployment record: " + err.Error()})
		return
	}

	// Update the URL path in the database for the deployment
	err = repository.UpdateDeploymentStatus(depId, "draft", targetDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update deployment url: " + err.Error()})
		return
	}

	// Trigger asynchronous build execution
	githubrepo.BuildProjectAsync(depId, targetDir)

	c.JSON(http.StatusOK, gin.H{
		"message":       "files uploaded successfully into namespace as draft, build in progress",
		"deployment_id": depId,
		"path":          targetDir,
	})
}
