package files

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"

	"github.com/gin-gonic/gin"
)

type SelectedFile struct {
	Name  string
	Path  string
	Files []*multipart.FileHeader
}

func FileUploadHandler(c *gin.Context) {
	fmt.Println("debgu log here")
	result := map[string]*SelectedFile{}
	form := c.Request.MultipartForm
	for key, values := range form.Value {

		var index, field string
		fmt.Sscanf(key, "filesgrp[%[^]]][%[^]]]", &index, &field)

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
		var index, field string
		fmt.Sscanf(key, "filesgrp%[^]]][%[^]]]", &index, &field)

		if field == "files" {
			if result[index] == nil {
				result[index] = &SelectedFile{}
			}
			result[index].Files = append(result[index].Files, files...)
		}
	}

	for _, group := range result {
		for _, fileHeader := range group.Files {
			src, _ := fileHeader.Open()

			dst, _ := os.Create("./uploads/" + fileHeader.Filename)

			io.Copy(dst, src)

			src.Close()
			dst.Close()
		}
	}

}
