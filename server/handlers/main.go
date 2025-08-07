package handlers

import (
	"database/sql"

	infracondb "github.com/Owoade/infracon/db"
	_ "github.com/mattn/go-sqlite3"
)

type ServerHandler struct {
	DB   *sql.DB
	Repo *infracondb.Repo
}

func NewServerHandler() *ServerHandler {

	db, err := infracondb.InitializeDB()

	repo := infracondb.NewRepo(db)

	if err != nil {
		panic(err)
	}

	return &ServerHandler{
		DB:   db,
		Repo: repo,
	}
}
