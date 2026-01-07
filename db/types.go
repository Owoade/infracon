package db

import "database/sql"

type UserModel struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ApplicationModel struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Path               string  `json:"path"`
	ClientPath         string  `json:"client_path"`
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
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

type Repo struct {
	DB *sql.DB
}
