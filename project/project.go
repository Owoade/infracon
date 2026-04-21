package project

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"infracon/db"
	"infracon/utils"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
			"message": err,
			"status":  false,
		})
		return
	}

	if err := utils.StringValidator("source", source, utils.ValidatorConfig{
		NotEmpty:       true,
		ExpectedValues: []string{"github", "zip-upload"},
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

	var slug string
	if err := db.QueryRow("SELECT slug FROM projects WHERE id = $1", projectId).Scan(&slug); err != nil {
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
				"details": err,
			})
			return
		}

		if err := utils.UnzipFileFromMultipartFile(fileHeader, destination); err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Error uploading zip",
				"status":  false,
			})
			return
		}

	}

	if source == "github" {

	}

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
			"details": err,
		})
		return
	}

	if err := utils.StringValidator("token", body.Token, utils.ValidatorConfig{
		NotEmpty:  true,
		MinLength: 16,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err,
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
	if err := db.QueryRow("SELECT id FROM github_tokens").Scan(&existingTokenId); err != nil {
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
		if err := db.QueryRow("UPDATE github_tokens SET token = $1 WHERE id = $2 RETURNING id", body.Token, existingTokenId).Scan(new(int)); err != nil {
			log.Printf("update query error: %s", err)
			c.JSON(http.StatusOK, gin.H{
				"message": "Something went wrong",
				"status":  false,
			})
			return
		}
	} else {
		if err := db.QueryRow("INSERT INTO github_tokens (token) VALUES ($1) RETURNING id", body.Token, existingTokenId).Scan(new(int)); err != nil {
			log.Printf("insert query error: %s", err)
			c.JSON(http.StatusOK, gin.H{
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
		if !errors.Is(err, sql.ErrNoRows) {
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
		c.JSON(http.StatusOK, gin.H{
			"message": "Something went wrong",
			"status":  false,
			"details": err,
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

// func CreateProject(c *gin.Context) {
// 	source := c.PostForm("source")
// 	projectName := c.PostForm("name")
// 	if projectName == "" || utf8.RuneCountInString(projectName) > 5 {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"status":  false,
// 			"message": "Project cannot be empty or less than 5 characters",
// 		})
// 		return
// 	}

// 	if source != "github" && source != "zip-upload" {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"status":  false,
// 			"message": "Invalid source type, expecting 'zip-upload' or 'github'",
// 		})
// 		return
// 	}

// 	if source == "zip-upload" {
// 		file, err := c.FormFile("file")
// 		if err != nil {
// 			log.Println(err)
// 			c.JSON(http.StatusBadRequest, gin.H{
// 				"status":  false,
// 				"message": "Zip upload error",
// 			})
// 			return
// 		}

// 		if err := utils.IsZipFile(file); err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{
// 				"status":  false,
// 				"message": "Zip upload error",
// 				"details": err,
// 			})
// 			return
// 		}

// 		home, _ := os.UserHomeDir()
// 		destination := fmt.Sprintf("%s/infracon-apps/%s", home, )

// 	}
// }
