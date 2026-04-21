package files

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxUploadSize = 100 << 20

type SelectedFile struct {
	Name  string
	Path  string
	Files []*multipart.FileHeader
}

func isSafePath(base, target string) bool {
	absBase, _ := filepath.Abs(base)
	absTarget, _ := filepath.Abs(target)
	return strings.HasPrefix(absTarget, absBase)
}

// parseKey parses "filesgrp[0][name]" -> ("0", "name")
func parseKey(key string) (index, field string) {
	// key format: filesgrp[INDEX][FIELD]
	first := strings.Index(key, "[")
	if first == -1 {
		return
	}
	firstClose := strings.Index(key, "]")
	if firstClose == -1 {
		return
	}
	index = key[first+1 : firstClose]

	rest := key[firstClose+1:] // "[name]"
	if len(rest) < 2 {
		return
	}
	field = rest[1 : len(rest)-1] // strip [ and ]
	return
}

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

	result := map[string]*SelectedFile{}

	for key, values := range form.Value {
		index, field := parseKey(key)
		fmt.Printf("Value parse: key=%q index=%q field=%q\n", key, index, field)

		if index == "" || len(values) == 0 {
			continue
		}
		if result[index] == nil {
			result[index] = &SelectedFile{}
		}
		switch field {
		case "name":
			result[index].Name = values[0]
		case "path":
			result[index].Path = values[0]
		}
	}

	for key, files := range form.File {
		index, field := parseKey(key)
		fmt.Printf("File parse: key=%q index=%q field=%q\n", key, index, field)

		if field == "files" {
			if result[index] == nil {
				result[index] = &SelectedFile{}
			}
			result[index].Files = append(result[index].Files, files...)
		}
	}

	fmt.Println("=== parsed result ===")
	for idx, group := range result {
		fmt.Printf("Group[%s]: Name=%q Path=%q Files=%d\n", idx, group.Name, group.Path, len(group.Files))
		for _, f := range group.Files {
			fmt.Printf("  -> file: %q size: %d\n", f.Filename, f.Size)
		}
	}

	if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create uploads dir"})
		return
	}

	for _, group := range result {
		if len(group.Files) == 0 {
			continue
		}

		destDir := filepath.Join("./uploads", filepath.Dir(filepath.Clean(group.Path)))
		fmt.Printf("Saving to destDir=%q\n", destDir)

		if !isSafePath("./uploads", destDir) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid path: " + destDir})
			return
		}

		if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create directory: " + err.Error()})
			return
		}

		for _, fileHeader := range group.Files {
			destPath := filepath.Join(destDir, fileHeader.Filename)
			fmt.Printf("Writing file: %q\n", destPath)

			src, err := fileHeader.Open()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to open uploaded file: " + err.Error()})
				return
			}

			dst, err := os.Create(destPath)
			if err != nil {
				src.Close()
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create destination file: " + err.Error()})
				return
			}

			if _, err := io.Copy(dst, src); err != nil {
				src.Close()
				dst.Close()
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to save file: " + err.Error()})
				return
			}

			src.Close()
			dst.Close()
			fmt.Printf("Saved: %q\n", destPath)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "files uploaded successfully"})
}
