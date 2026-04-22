package project

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"infracon/db"
	"infracon/utils"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func CreateProject(c *gin.Context) {
	var body CreateProjectPayload
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Invalid payload",
			"details": err,
		})
		return
	}

	db, err := db.GetDatabase()
	if err != nil {
		log.Printf("db error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	uniqueSlug := fmt.Sprintf("%s-%d", utils.Slugify(body.Name), time.Now().UnixMilli())
	if err := db.QueryRow("INSERT INTO projects (name, slug, type) VALUES ($1, $2, $3) RETURNING id", body.Name, uniqueSlug, body.Type).Scan(new(int)); err != nil {
		log.Printf("query error: %s", err)
		c.JSON(http.StatusOK, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"message": "Project created",
		"data": gin.H{
			"slug": uniqueSlug,
		},
	})

}

func GetProject(c *gin.Context) {
	slug := c.Param("slug")
	if err := utils.StringValidator("slug", slug, utils.ValidatorConfig{
		NotEmpty: true,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err,
			"status":  false,
		})
		return
	}

	db, err := db.GetDatabase()
	if err != nil {
		log.Printf("db error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}
	var project Project
	if err := db.QueryRow("SELECT name, type, project_path, env, github_repo FROM projects WHERE slug = $1", slug).
		Scan(&project.Name, &project.Type, &project.ProjectPath, &project.Env, &project.GithubRepo); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Project not found!",
				"status":  false,
			})
			return
		} else {
			log.Printf("project lookup query error: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Something went wrong",
				"status":  false,
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data": gin.H{
			"project": project,
		},
	})

}

func AddProjectSource(c *gin.Context) {
	projectId := c.PostForm("project_id")
	source := c.PostForm("source")

	if err := utils.StringValidator("project_id", projectId, utils.ValidatorConfig{
		NotEmpty:  true,
		MaxLength: 16,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
			"status":  false,
		})
		return
	}

	if err := utils.StringValidator("source", source, utils.ValidatorConfig{
		NotEmpty:       true,
		ExpectedValues: []string{"github", "zip-upload"},
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
			"status":  false,
		})
		return
	}

	db, err := db.GetDatabase()
	if err != nil {
		log.Printf("db error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	var slug string
	var existingProjectPath *string
	if err := db.QueryRow("SELECT slug, project_path FROM projects WHERE id = $1", projectId).Scan(&slug, &existingProjectPath); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Project not found!",
				"status":  false,
			})
			return
		} else {
			log.Printf("project lookup query error: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Something went wrong",
				"status":  false,
			})
			return
		}
	}

	home, _ := os.UserHomeDir()
	destination := filepath.Join(home, "infracon-apps", slug)

	if source == "zip-upload" {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Error uploading zip",
				"status":  false,
				"details": err.Error(),
			})
			return
		}

		if err := utils.IsZipFile(fileHeader); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Invalid file",
				"status":  false,
				"details": err.Error(),
			})
			return
		}

		clientFolders, err := utils.UnzipFileFromMultipartFile(fileHeader, destination)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Error uploading zip",
				"status":  false,
			})
			return
		}

		if len(clientFolders) == 1 {
			clientFolder := clientFolders[0]
			destination = filepath.Join(destination, clientFolder)

			if err := db.QueryRow("UPDATE projects SET project_path = $1 WHERE id = $2 RETURNING id", destination, projectId).Scan(new(int)); err != nil {
				log.Printf("update query err: %s", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Error uploading zip",
					"status":  false,
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status":  true,
				"message": "Source added",
				"data": gin.H{
					"top_level_folders": clientFolders,
				},
			})
			return

		} else {
			if err := db.QueryRow("UPDATE projects SET project_path = $1, top_level_directories = $2 WHERE id = $3 RETURNING id", destination, strings.Join(clientFolders, ","), projectId).Scan(new(int)); err != nil {
				log.Printf("update query err: %s", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Error uploading zip",
					"status":  false,
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"status":  true,
				"message": "Source added",
				"data": gin.H{
					"next_action": gin.H{
						"message":           "YYour ZIP file contains multiple top-level folders. Please choose the correct one.",
						"top_level_folders": clientFolders,
					},
				},
			})
			return
		}

	}

	if source == "github" {
		repo := c.PostForm("repo")
		owner := c.PostForm("owner")
		branch := c.PostForm("branch")

		if err := utils.StringValidator("repo", repo, utils.ValidatorConfig{
			NotEmpty: true,
		}); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status":  false,
				"message": err.Error(),
			})
			return
		}

		if err := utils.StringValidator("owner", owner, utils.ValidatorConfig{
			NotEmpty: true,
		}); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status":  false,
				"message": err.Error(),
			})
			return
		}

		if err := utils.StringValidator("branch", branch, utils.ValidatorConfig{
			NotEmpty: true,
		}); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status":  false,
				"message": err.Error(),
			})
			return
		}

		var token string
		if err := db.QueryRow("SELECT token FROM github_tokens").Scan(&token); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{
					"message": "Github token not found!",
					"status":  false,
				})
				return
			} else {
				log.Printf("query token error: %s", err)
				c.JSON(http.StatusOK, gin.H{
					"message": "Something went wrong",
					"status":  false,
				})
				return
			}
		}

		payload := PullfromGithub{
			Repo:        repo,
			Owner:       owner,
			Ref:         branch,
			AccessToken: token,
			Destination: destination,
		}

		commitHash, err := pullFromGithub(payload)
		if err != nil {
			log.Printf("github repo pull error: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Something went wrong",
				"status":  false,
				"details": gin.H{
					"message":        err.Error(),
					"possible_cause": "Invalid github token or repository name, owner or branch",
				},
			})
			return
		}
		clientFolder := fmt.Sprintf("%s-%s-%s", owner, repo, commitHash)
		destination = filepath.Join(destination, clientFolder)
		githubRepo := map[string]string{
			"name":   repo,
			"owner":  owner,
			"branch": branch,
		}
		bytes, _ := json.Marshal(githubRepo)
		marshaledRepo := string(bytes)

		if err := db.QueryRow("UPDATE projects SET project_path = $1, github_repo = $2 WHERE id = $3 RETURNING id", destination, marshaledRepo, projectId).Scan(new(int)); err != nil {
			log.Printf("update query err: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Error uploading zip",
				"status":  false,
			})
			return
		}

		println("existingProjectPath", *existingProjectPath)
		println("destination", destination)
		if existingProjectPath != nil && destination != *existingProjectPath {
			go os.RemoveAll(*existingProjectPath)
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  true,
			"message": "Source Added",
		})

	}

}

func SetEnvironmentVariable(c *gin.Context) {
	var body SetEnvironmentVariablePayload
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Invalid payload",
			"details": err.Error(),
		})
		return
	}

	db, err := db.GetDatabase()
	if err != nil {
		log.Printf("db error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	var project_path string
	if err := db.QueryRow("SELECT project_path FROM projects WHERE id = $1", body.ProjectId).Scan(&project_path); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Project not found!",
				"status":  false,
			})
			return
		} else {
			log.Printf("project lookup query error: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Something went wrong",
				"status":  false,
			})
			return
		}
	}

	if err := db.QueryRow("UPDATE projects SET env = $1 WHERE id = $2 RETURNING id", body.Env, body.ProjectId).Scan(new(int)); err != nil {
		log.Printf("env write error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"message": "Environment variables set!",
	})

}

func GetGithubTokens(c *gin.Context) {
	db, err := db.GetDatabase()
	if err != nil {
		log.Printf("db error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	var token string
	if err := db.QueryRow("SELECT token FROM github_tokens").Scan(&token); err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("query error: %s", err)
		c.JSON(http.StatusOK, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"message": "Github classic tokens",
		"data": gin.H{
			"token": token,
		},
	})

}

func AddGithubToken(c *gin.Context) {
	var body AddGithubTokenPayload
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Invalid payload",
			"details": err.Error(),
		})
		return
	}

	if err := utils.StringValidator("token", body.Token, utils.ValidatorConfig{
		NotEmpty:  true,
		MinLength: 16,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
			"status":  false,
		})
		return
	}

	if err := validateGithubToken(body.Token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid github token",
			"status":  false,
			"details": err,
		})
		return
	}

	db, err := db.GetDatabase()
	if err != nil {
		log.Printf("db error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	var existingTokenId int
	if err := db.QueryRow("SELECT user_id FROM github_tokens").Scan(&existingTokenId); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("query error: %s", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "Something went wrong",
				"status":  false,
			})
			return
		}
	}

	if existingTokenId > 0 {
		if err := db.QueryRow("UPDATE github_tokens SET token = $1 WHERE user_id = $2 RETURNING user_id", body.Token, existingTokenId).Scan(new(int)); err != nil {
			log.Printf("update query error: %s", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "Something went wrong",
				"status":  false,
			})
			return
		}
	} else {
		if err := db.QueryRow("INSERT INTO github_tokens (user_id, token) VALUES (1,$1) RETURNING user_id", body.Token, existingTokenId).Scan(new(int)); err != nil {
			log.Printf("insert query error: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Something went wrong",
				"status":  false,
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"message": "Github token added successfully",
	})

}

func GetGithubRepos(c *gin.Context) {
	var body GetGithubReposPayload
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Invalid payload",
			"details": err.Error(),
		})
		return
	}

	db, err := db.GetDatabase()
	if err != nil {
		log.Printf("db error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	var token string
	if err := db.QueryRow("SELECT token FROM github_tokens").Scan(&token); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Github token not found!",
				"status":  false,
			})
			return
		} else {
			log.Printf("query token error: %s", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "Something went wrong",
				"status":  false,
			})
			return
		}
	}

	if body.Page == 0 {
		body.Page = 1
	}

	if body.PerPage == 0 {
		body.PerPage = 50
	}

	url := fmt.Sprintf("https://api.github.com/user/repos?page=%d&per_page=%d&sort=%s&direction=%s", body.Page, body.PerPage, "pushed", "desc")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("http error 1: %s", err)
		c.JSON(http.StatusOK, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("http error 2: %s", err)
		c.JSON(http.StatusOK, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("http error 3: %s", err)
		var response map[string]string
		json.NewDecoder(resp.Body).Decode(&response)
		c.JSON(http.StatusOK, gin.H{
			"message": "Something went wrong",
			"status":  false,
			"details": fmt.Sprintf("Github error: %s", response["message"]),
		})
		return
	}

	var repos []FetchGithubRepoAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		log.Printf("json encoding error: %s", err)
		c.JSON(http.StatusOK, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data": gin.H{
			"repos": repos,
		},
		"meta": gin.H{
			"page":     body.Page,
			"per_page": body.PerPage,
		},
	})

}

func GetGithubRepoBranches(c *gin.Context) {
	var body GetGithubBranchesPayload
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Invalid payload",
			"details": err,
		})
		return
	}

	db, err := db.GetDatabase()
	if err != nil {
		log.Printf("db error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	var token string
	if err := db.QueryRow("SELECT token FROM github_tokens").Scan(&token); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Github token not found!",
				"status":  false,
			})
			return
		} else {
			log.Printf("query token error: %s", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "Something went wrong",
				"status":  false,
			})
			return
		}
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches", body.Owner, body.RepoName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("http error 1: %s", err)
		c.JSON(http.StatusOK, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("http error 2: %s", err)
		c.JSON(http.StatusOK, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("http error 3: %s", err)
		var response map[string]string
		json.NewDecoder(resp.Body).Decode(&response)
		c.JSON(http.StatusOK, gin.H{
			"message": "Something went wrong",
			"status":  false,
			"details": fmt.Sprintf("Github error: %s", response["message"]),
		})
		return
	}

	var branches []FetchGithubRepoBranchesAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&branches); err != nil {
		log.Printf("json encoding error: %s", err)
		c.JSON(http.StatusOK, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data": gin.H{
			"branches": branches,
		},
	})

}

func validateGithubToken(token string) error {
	if token == "" {
		return errors.New("invalid token: empty token")
	}

	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("invalid github token")
	}

	return nil
}

func pullFromGithub(p PullfromGithub) (commitHash string, err error) {
	if p.AccessToken == "" {
		return "", errors.New("GITHUB_ACCESS_TOKEN is not set")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/%s", p.Owner, p.Repo, p.Ref)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+p.AccessToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var response map[string]string
		json.NewDecoder(resp.Body).Decode(&response)
		return "", fmt.Errorf("Github error: %s", response["message"])
	}

	var buf bytes.Buffer
	size, err := io.Copy(&buf, resp.Body)
	if err != nil {
		return "", err
	}

	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), size)
	if err != nil {
		return "", err
	}

	for _, f := range r.File {
		fpath := filepath.Join(p.Destination, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(p.Destination)+string(os.PathSeparator)) {
			return "", fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return "", err
		}

		inFile, err := f.Open()
		if err != nil {
			return "", err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			inFile.Close()
			return "", err
		}

		_, err = io.Copy(outFile, inFile)
		inFile.Close()
		outFile.Close()
		if err != nil {
			return "", err
		}
	}

	return r.Comment, nil
}
