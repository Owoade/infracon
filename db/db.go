package db

import (
	"database/sql"
	"errors"

	_ "github.com/mattn/go-sqlite3"
)

type ApplicationModel struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Path               string `json:"path"`
	ClientPath         string `json:"client_path"`
	DeploymentStrategy *string `json:"deployment_strategy,omitempty"`
	Type               *string `json:"type,omitempty"`
	DockerfilePath     *string `json:"dockerfile_path,omitempty"`
	BuildCommand       *string `json:"build_command,omitempty"`
	RunCommand         *string `json:"run_command,omitempty"`
	ApplicationPort    *int    `json:"application_port,omitempty"`
	InternalPort       *int    `json:"internal_port,omitempty"`
	ContainerID        *string `json:"container_id,omitempty"`
	ImageID            *string `json:"image_id,omitempty"`
	ApplicationType    *string `json:"application_type,omitempty"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type Repo struct {
	DB *sql.DB
}

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
