package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func GetDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./infracon.db")
	if err != nil {
		return nil, err
	}
	return db, nil
}