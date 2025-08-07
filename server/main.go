package server

import (
	"fmt"
	"net/http"

	"github.com/Owoade/infracon/db"
	"github.com/Owoade/infracon/server/handlers"
)

func Start() {

	fmt.Println("Running server on port 2000")

	db, err := db.InitializeDB()

	if err != nil {
		panic(err)
	}

	handler := handlers.NewServerHandler(db)

	http.HandleFunc("/upload", handler.UploadFile)
	http.HandleFunc("/auth", handler.Authenticate)

	err = http.ListenAndServe(":2000", nil)

	if err != nil {
		panic(err)
	}
}
