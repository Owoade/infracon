package server

import (
	"fmt"
	"net/http"

	"github.com/Owoade/infracon/server/handlers"
)

func Start() {

	fmt.Println("Running server on port 2000")

	handler := handlers.NewServerHandler()

	http.HandleFunc("/upload", handler.UploadFile)
	http.HandleFunc("/auth", handler.Authenticate)
	http.HandleFunc("/connect", handler.Connect)
	http.HandleFunc("/apps", handler.ListProject)

	if err := http.ListenAndServe(":2000", nil); err != nil {
		panic(err)
	}
}
