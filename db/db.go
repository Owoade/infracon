package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func InitializeDB() (*sql.DB, error) {

	db, err := sql.Open("sqlite3", "./infracon.db")
	if err != nil {
		return nil, err
	}

	sqlStmt := `
		CREATE TABLE IF NOT EXISTS apps (
			id TEXT NOT NULL PRIMARY KEY,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			client_path TEXT NOT NULL,
			deployment_strategy TEXT,
			type TEXT,
			dockerfile_path TEXT,
			build_command TEXT,
			run_command TEXT,
			application_port int,
			internal_port int,
			container_id TEXT,
			image_id TEXT,
			application_type TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS users (
			email TEXT NOT NULL,
			password TEXT NOT NULL,
			is_root INTEGER CHECK(is_root IN (0,1)) DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)
    `

	_, err = db.Exec(sqlStmt)
	if err != nil {
		return nil, err
	}

	return db, err
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{
		DB: db,
	}
}

func (repo *Repo) GetApplicationFromDB(applicationId string) (*ApplicationModel, error) {

	var application ApplicationModel

	err := repo.DB.QueryRow(
		`SELECT * FROM apps WHERE id = ?`,
		applicationId,
	).Scan(
		&application.ID,
		&application.Name,
		&application.Path,
		&application.ClientPath,
		&application.DeploymentStrategy,
		&application.Type,
		&application.DockerfilePath,
		&application.BuildCommand,
		&application.RunCommand,
		&application.ApplicationPort,
		&application.InternalPort,
		&application.ContainerID,
		&application.ImageID,
		&application.ApplicationType,
		&application.CreatedAt,
		&application.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &application, nil

}

func (repo *Repo) GetApplicationFromDBByClientPath(clientPath string) (*ApplicationModel, error) {

	var application ApplicationModel

	err := repo.DB.QueryRow(
		`SELECT * FROM apps WHERE client_path = ?`,
		clientPath,
	).Scan(
		&application.ID,
		&application.Name,
		&application.Path,
		&application.ClientPath,
		&application.DeploymentStrategy,
		&application.Type,
		&application.DockerfilePath,
		&application.BuildCommand,
		&application.RunCommand,
		&application.ApplicationPort,
		&application.InternalPort,
		&application.ContainerID,
		&application.ImageID,
		&application.ApplicationType,
		&application.CreatedAt,
		&application.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &application, nil

}

func (repo *Repo) GetApplicationsFromDB() (*[]ApplicationModel, error) {

	var applications []ApplicationModel

	rows, err := repo.DB.Query(`SELECT * FROM apps`)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {

		var application ApplicationModel

		if err = rows.Scan(
			&application.ID,
			&application.Name,
			&application.Path,
			&application.ClientPath,
			&application.DeploymentStrategy,
			&application.Type,
			&application.DockerfilePath,
			&application.BuildCommand,
			&application.RunCommand,
			&application.ApplicationPort,
			&application.InternalPort,
			&application.CreatedAt,
			&application.UpdatedAt,
		); err != nil {
			return nil, err
		}

		applications = append(applications, application)

	}

	return &applications, nil

}

func (repo *Repo) GetUser(email string) (*UserModel, error) {
	var user UserModel
	if err := repo.DB.QueryRow(
		`SELECT email, password, is_root FROM users where email = $1;`,
		email,
	).Scan(&user.Email, &user.Password, &user.IsRoot); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (repo *Repo) CreateUser(email, password string) (bool, error) {
	var existingUsersCount int
	if err := repo.DB.QueryRow(
		`SELECT COUNT(*) FROM users`,
	).Scan(&existingUsersCount); err != nil {
		return false, err
	}

	isRootUser := 0
	if existingUsersCount == 0 {
		isRootUser = 1
	}

	_, err := repo.DB.Exec(
		`
		INSERT INTO users (email, password, is_root, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		`,
		email,
		password,
		isRootUser,
		time.Now().UTC(),
		time.Now().UTC(),
	)
	if err != nil {
		return false, err
	}

	return existingUsersCount == 0, nil
}

func (repo *Repo) UpdateUserPassword(email, password string) error {
	_, err := repo.DB.Exec(
		`
		UPDATE users SET password = $1, updated_at = $2 WHERE email = $3
		`,
		password,
		time.Now().UTC(),
		email,
	)
	return err
}
