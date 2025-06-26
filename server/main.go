package server

import (
	"fmt"
	"net/http"

	"github.com/Owoade/infracon/server/handlers"
)

func Start() {
	fmt.Println("Running server on port 2000")
	http.HandleFunc("/upload", handlers.UploadFile)
	http.HandleFunc("/auth", handlers.Authenticate)
	err := http.ListenAndServe(":2000", nil)

	if err != nil {
		panic(err)
	}
}
