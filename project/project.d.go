package project

import "time"

type CreateProjectPayload struct {
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required"`
}

type AddGithubTokenPayload struct {
	Token string `json:"token" binding:"required"`
}

type GetGithubReposPayload struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

type FetchGithubRepoAPIResponse struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Private bool   `json:"private"`
	Owner   struct {
		Login string `json:"login"`
	} `json:"owner"`
	DefaultBranch string `json:"default_branch"`
}

type GetGithubBranchesPayload struct {
	Owner    string `json:"owner" binding:"required"`
	RepoName string `json:"repo_name" binding:"required"`
}

type FetchGithubRepoBranchesAPIResponse struct {
	Name string `json:"name"`
}

type PullfromGithub struct {
	Owner       string
	Repo        string
	Ref         string
	AccessToken string
	Destination string
}

type SetEnvironmentVariablePayload struct {
	ProjectId string `json:"project_id"`
	Env       string `json:"env"`
}

type Project struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	Type        *string   `json:"type" db:"type"`
	Env         *string   `json:"env" db:"env"`
	GithubRepo  *string   `json:"github_repo" db:"github_repo"`
	ProjectPath *string   `json:"project_path" db:"project_path"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
