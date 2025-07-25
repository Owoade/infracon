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

	if validAuth, err := VerifyToken(r); validAuth {
		http.Error(w, err, 400)
	}

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

	home, _ := os.UserHomeDir()
	path := r.FormValue("path")
	projectPath := home + "/" + "infracon-apps" + "/" + path

	projectPathArr := strings.Split(projectPath, "/")
	directoryPath := strings.Join(projectPathArr[:len(projectPathArr)-1], "/")

	fmt.Println(directoryPath)

	os.MkdirAll(directoryPath, 0755)

	destinationFile, err := os.Create(projectPath)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, file)

	if err != nil {
		response := map[string]any{
			"file_name": metadata.Filename,
			"uploaded":  false,
		}

		json.NewEncoder(w).Encode(response)
	}

	response := map[string]any{
		"file_name": metadata.Filename,
		"uploaded":  true,
	}

	json.NewEncoder(w).Encode(response)

}
