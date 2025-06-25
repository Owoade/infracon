package http_handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func UploadFile(w http.ResponseWriter, r *http.Request) (e error) {

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		return err
	}

	file, metadata, err := r.FormFile("file")
	if err != nil {
		return err
	}
	fmt.Println("Metadata", metadata)
	defer file.Close()

	path := r.FormValue("path")

	pathElements:=  strings.Split(path, "/")
	
	if len(pathElements) > 1 {
		folders := pathElements[:len(pathElements)-1]
		folderPath := strings.Join(folders, "/")
		os.MkdirAll(folderPath, 0755)
	}

	destinationFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, file)

	return err

}

