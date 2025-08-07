package handlers

import "database/sql"

type ServerHandler struct {
	db *sql.DB
}

func NewServerHandler( db *sql.DB )(*ServerHandler){
	return &ServerHandler{
		db: db,
	}
}
