package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
	fmt.Println("Connected to server on port 2000")
	http.HandleFunc("/upload", UploadFile)
	err := http.ListenAndServe(":2000", nil)

	if err != nil {
		panic(err)
	}
}

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
	defer file.Close()
	fmt.Println(metadata.Filename)

	path := r.FormValue("path")

	pathElements:=  strings.Split(path, "/")
	
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
		http.Error(w, err.Error(), 400)
		return
	}
}
