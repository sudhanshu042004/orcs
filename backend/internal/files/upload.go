package files

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
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
	fullPath := parentFolder + "/" + key

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
	if err := c.Request.ParseMultipartForm(maxUploadSize); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form: " + err.Error()})
		return
	}

	form := c.Request.MultipartForm
	if form == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty form"})
		return
	}
	uploads := "uploads"
	info, err := os.Stat(uploads)
	if err == nil {
		if info.IsDir() {
			fmt.Println("Directory exists.")
		} else {
			fmt.Println("Path exists but is a file, not a directory.")
		}
	} else if errors.Is(err, os.ErrNotExist) {
		fmt.Println("Directory does not exist.")
		os.Mkdir(uploads, 0755)
	} else {
		fmt.Printf("Error checking directory: %v\n", err)
	}

	for key, headers := range form.File {
		for _, fileHeader := range headers {
			err := saveFile(key, fileHeader, uploads)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "files uploaded successfully"})
}
