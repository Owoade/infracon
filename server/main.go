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
	http.HandleFunc("/deploy", handler.DeployApplication)
	http.HandleFunc("/auth/login", handler.Login)
	http.HandleFunc("/auth/signup", handler.SignUp)
	http.HandleFunc("/auth/password", handler.ChangePassword)
	http.HandleFunc("/connect", handler.Connect)
	http.HandleFunc("/apps", handler.ListProject)

	if err := http.ListenAndServe(":2000", nil); err != nil {
		panic(err)
	}
}
