package project

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