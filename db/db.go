package db

import (
	"database/sql"
	"errors"
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
		`SELECT email, password FROM users where email = $1;`,
		email,
	).Scan(&user.Email, &user.Password); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (repo *Repo) CreateUser(email, password string) error {
	_, err := repo.DB.Exec(
		`
		INSERT INTO users (email, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		`,
		email,
		password,
		time.Now().UTC(),
		time.Now().UTC(),
	)
	return err
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
