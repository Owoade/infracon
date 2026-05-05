package db

import (
	"database/sql"
	"errors"
	"infracon/utils"

	_ "github.com/mattn/go-sqlite3"
)

func GetDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./infracon.db")
	if err != nil {
		return nil, err
	}
	return db, nil
}

func GetGithubToken() (string, error) {
	db, err := GetDatabase()
	if err != nil {
		return "", err
	}

	var token string
	if err := db.QueryRow("SELECT token FROM github_tokens").Scan(&token); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	return token, nil
}

func CreateProject(p utils.Project) (int, error) {
	db, err := GetDatabase()
	if err != nil {
		return 0, err
	}

	var id int
	if err := db.QueryRow("INSERT INTO projects (name, slug, type, env, project_path, status, container_name, current_image) VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $8) RETURNING id", p.Name, p.Slug, p.Type, p.Env, p.ProjectPath, p.Status, p.ContainerName, p.ContainerName).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}
