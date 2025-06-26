package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func UploadFile(w http.ResponseWriter, r *http.Request) {

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	file, metadata, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	fmt.Println("Metadata", metadata)
	defer file.Close()

	path := r.FormValue("path")

	pathElements := strings.Split(path, "/")

	if len(pathElements) > 1 {
		folders := pathElements[:len(pathElements)-1]
		folderPath := strings.Join(folders, "/")
		os.MkdirAll(folderPath, 0755)
	}

	destinationFile, err := os.Create(path)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, file)

	if err != nil {
		response := map[string]any{
			"file_name": metadata.Filename,
			"uploaded": false,
		}
	
		json.NewEncoder(w).Encode(response)
	}

	response := map[string]any{
		"file_name": metadata.Filename,
		"uploaded": true,
	}

	json.NewEncoder(w).Encode(response)

}
