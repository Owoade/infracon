package utils

import "time"

type ValidatorConfig struct {
	NotEmpty       bool
	MinLength      int
	MaxLength      int
	ExpectedValues []string
}

type DockerImage struct {
	ID       string   `json:"Id"`
	RepoTags []string `json:"RepoTags"`
	Config   struct {
		Env          []string            `json:"Env"`
		Cmd          []string            `json:"Cmd"`
		Entrypoint   []string            `json:"Entrypoint"`
		WorkingDir   string              `json:"WorkingDir"`
		ExposedPorts map[string]struct{} `json:"ExposedPorts"`
	} `json:"Config"`
}

type DockerContainer struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	Image string `json:"Image"`
	State struct {
		Status  string `json:"Status"`
		Running bool   `json:"Running"`
		Pid     int    `json:"Pid"`
	} `json:"State"`
	NetworkSettings struct {
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

type PullfromGithub struct {
	Owner       string
	Repo        string
	Ref         string
	AccessToken string
	Destination string
}

type Project struct {
	ID            int       `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Slug          string    `json:"slug" db:"slug"`
	Type          *string   `json:"type" db:"type"`
	Env           *string   `json:"env" db:"env"`
	GithubRepo    *string   `json:"github_repo" db:"github_repo"`
	ProjectPath   *string   `json:"project_path" db:"project_path"`
	Status        *string   `json:"status" db:"status"`
	ContainerName *string   `json:"container_name" db:"container_name"`
	CurrentImage  *string   `json:"current_image" db:"current_image"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}
