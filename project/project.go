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
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func CreateProject(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "SSE not supported",
		})
		return
	}

	var body CreateProjectPayload
	if err := c.ShouldBind(&body); err != nil {
		line := strings.Join([]string{strconv.Itoa(int(time.Now().UnixMilli())), "ERROR", err.Error()}, utils.InfraconLogSeparator)
		fmt.Fprintf(c.Writer, "data: %s\n\n", line)
		flusher.Flush()
		return
	}

	uniqueSlug := fmt.Sprintf("%s-%d", utils.Slugify(body.Name), time.Now().UnixMilli())
	imageName := uniqueSlug
	containerName := uniqueSlug
	projectPath := filepath.Join("infracon-apps", uniqueSlug)

	project := utils.Project{
		Name: body.Name,
		Slug: uniqueSlug,
		Type: &body.Type,
	}
	var err error
	project.ID, err = db.CreateProject(project)
	if err != nil {
		utils.WriteSSEData(uniqueSlug, []string{"ERROR", fmt.Sprintf("Error saving project to db: %s", err)}, c, flusher)
		return
	}

	var useCustomDockerfile = body.UseCustomDockerfile == "true"

	if body.Type == "zip-upload" {
		file, err := c.FormFile("file")
		if err != nil {
			utils.WriteSSEData(uniqueSlug, []string{"ERROR", fmt.Sprintf("error uploading file: %s", err)}, c, flusher)
			return
		}

		folders, err := utils.UnzipFileFromMultipartFile(file, projectPath)
		if err != nil {
			utils.WriteSSEData(uniqueSlug, []string{"ERROR", fmt.Sprintf("error unziping file: %s", err)}, c, flusher)
			return
		}

		if len(folders) != 1 {
			os.RemoveAll(projectPath)
			utils.WriteSSEData(uniqueSlug, []string{"ERROR", "The uploaded zip file must contain exactly one root folder, but none or multiple were found."}, c, flusher)
			return
		}

		projectPath = filepath.Join(projectPath, folders[0])
	}

	if body.Type == "github" {
		if body.RepoName == "" {
			utils.WriteSSEData(uniqueSlug, []string{"ERROR", "`repo_name` is required"}, c, flusher)
			return
		}

		if body.RepoOwner == "" {
			utils.WriteSSEData(uniqueSlug, []string{"ERROR", "`repo_owner` is required"}, c, flusher)
			return
		}

		if body.RepoRef == "" {
			utils.WriteSSEData(uniqueSlug, []string{"ERROR", "`repo_ref` is required"}, c, flusher)
			return
		}

		accessToken, err := db.GetGithubToken()
		if err != nil {
			utils.WriteSSEData(uniqueSlug, []string{"ERROR", fmt.Sprintf("error getting github token: %s", err.Error())}, c, flusher)
			return
		}

		payload := utils.PullfromGithub{
			Owner:       body.RepoOwner,
			Repo:        body.Name,
			Ref:         body.RepoRef,
			AccessToken: accessToken,
			Destination: projectPath,
		}

		commithash, err := utils.PullFromGithub(payload)
		if err != nil {
			utils.WriteSSEData(uniqueSlug, []string{"ERROR", fmt.Sprintf("error pulling repo from github: %s", err.Error())}, c, flusher)
			return
		}

		projectPath = filepath.Join(projectPath, commithash)
	}

	BuildImage(uniqueSlug, imageName, projectPath, useCustomDockerfile, c, flusher)
	if _, err := utils.GetDockerImage(uniqueSlug); err != nil {
		utils.WriteSSEData(uniqueSlug, []string{"ERROR", fmt.Sprintf("Error building docker image: %s", err)}, c, flusher)
		return
	}

	envPath := filepath.Join(projectPath, ".env")
	if body.Env != "" {
		utils.WriteEnvFile(projectPath, body.Env)
	}

	status, _ := RunContainer(uniqueSlug, imageName, containerName, envPath, c, flusher)

	project = utils.Project{
		Env:           &body.Env,
		ProjectPath:   &projectPath,
		Status:        &status,
		ContainerName: &uniqueSlug,
		CurrentImage:  &uniqueSlug,
	}

	if err := db.UpdateProject(project); err != nil {
		utils.WriteSSEData(uniqueSlug, []string{"ERROR", fmt.Sprintf("Error saving project to db: %s", err)}, c, flusher)
		return
	}

	logs := utils.GetLogs(uniqueSlug)
	defer utils.DeleteLogs(uniqueSlug)
	if len(logs) > 0 {
		if err := db.SaveLogs(uniqueSlug, logs); err != nil {
			line := strings.Join([]string{strconv.Itoa(int(time.Now().UnixMilli())), "ERROR", err.Error()}, utils.InfraconLogSeparator)
			fmt.Fprintf(c.Writer, "data: %s\n\n", line)
			flusher.Flush()
		}
	}

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

	project, err := db.GetProject(slug)
	if err != nil {
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

func GetProjects(c *gin.Context) {

	projects, err := db.GetProjects()
	if err != nil {
		log.Printf("project fetch query error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data": gin.H{
			"project": projects,
		},
	})

}

func UpdateProjectSource(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "SSE not supported",
		})
		return
	}

	slug := c.PostForm("slug")
	source := c.PostForm("source")
	useCustomDockerfile := c.PostForm("use_custom_dockerfile") == "true"

	if err := utils.StringValidator("slug", slug, utils.ValidatorConfig{
		NotEmpty: true,
	}); err != nil {
		line := strings.Join([]string{strconv.Itoa(int(time.Now().UnixMilli())), "ERROR", err.Error()}, utils.InfraconLogSeparator)
		fmt.Fprintf(c.Writer, "data: %s\n\n", line)
		flusher.Flush()
		return
	}

	if err := utils.StringValidator("source", source, utils.ValidatorConfig{
		NotEmpty:       true,
		ExpectedValues: []string{"github", "zip-upload"},
	}); err != nil {
		line := strings.Join([]string{strconv.Itoa(int(time.Now().UnixMilli())), "ERROR", err.Error()}, utils.InfraconLogSeparator)
		fmt.Fprintf(c.Writer, "data: %s\n\n", line)
		flusher.Flush()
		return
	}

	project, err := db.GetProject(slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			line := strings.Join([]string{strconv.Itoa(int(time.Now().UnixMilli())), "ERROR", "Project nor found"}, utils.InfraconLogSeparator)
			fmt.Fprintf(c.Writer, "data: %s\n\n", line)
			flusher.Flush()
			return
		} else {
			line := strings.Join([]string{strconv.Itoa(int(time.Now().UnixMilli())), "ERROR", "Project nor found"}, utils.InfraconLogSeparator)
			fmt.Fprintf(c.Writer, "data: %s\n\n", line)
			flusher.Flush()
			return
		}
	}

	deploymentId := fmt.Sprintf("%s-%s", slug, strconv.Itoa(int(time.Now().UnixMilli())))
	newProjectPath := filepath.Join("infracon-apps", deploymentId)
	containerName := deploymentId
	imageName := deploymentId

	if source == "zip-upload" {
		file, err := c.FormFile("file")
		if err != nil {
			utils.WriteSSEData(slug, []string{"ERROR", fmt.Sprintf("error uploading file: %s", err)}, c, flusher)
			return
		}

		folders, err := utils.UnzipFileFromMultipartFile(file, newProjectPath)
		if err != nil {
			os.RemoveAll(newProjectPath)
			utils.WriteSSEData(slug, []string{"ERROR", fmt.Sprintf("error unziping file: %s", err)}, c, flusher)
			return
		}

		if len(folders) != 1 {
			os.RemoveAll(newProjectPath)
			utils.WriteSSEData(slug, []string{"ERROR", "The uploaded zip file must contain exactly one root folder, but none or multiple were found."}, c, flusher)
			return
		}

		newProjectPath = filepath.Join(newProjectPath, folders[0])
	}

	if source == "github" {
		repoName := c.PostForm("github_repo_name")
		repoOwner := c.PostForm("github_owner")
		repoBranch := c.PostForm("github_branch")

		if repoName == "" {
			utils.WriteSSEData(project.Slug, []string{"ERROR", "`repo_name` is required"}, c, flusher)
			return
		}

		if repoOwner == "" {
			utils.WriteSSEData(project.Slug, []string{"ERROR", "`repo_owner` is required"}, c, flusher)
			return
		}

		if repoBranch == "" {
			utils.WriteSSEData(project.Slug, []string{"ERROR", "`repo_ref` is required"}, c, flusher)
			return
		}

		accessToken, err := db.GetGithubToken()
		if err != nil {
			utils.WriteSSEData(project.Slug, []string{"ERROR", fmt.Sprintf("error getting github token: %s", err.Error())}, c, flusher)
			return
		}

		payload := utils.PullfromGithub{
			Owner:       repoOwner,
			Repo:        repoName,
			Ref:         repoBranch,
			AccessToken: accessToken,
			Destination: newProjectPath,
		}

		commithash, err := utils.PullFromGithub(payload)
		if err != nil {
			utils.WriteSSEData(project.Slug, []string{"ERROR", fmt.Sprintf("error pulling repo from github: %s", err.Error())}, c, flusher)
			return
		}

		newProjectPath = filepath.Join(newProjectPath, commithash)
	}

	BuildImage(deploymentId, deploymentId, newProjectPath, useCustomDockerfile, c, flusher)
	if _, err := utils.GetDockerImage(deploymentId); err != nil {
		utils.WriteSSEData(project.Slug, []string{"ERROR", fmt.Sprintf("Error building docker image: %s", err)}, c, flusher)
		return
	}

	envPath := filepath.Join(newProjectPath, ".env")
	if project.Env != nil {
		utils.WriteEnvFile(newProjectPath, *project.Env)
	}

	status, _ := RunContainer(project.Slug, imageName, containerName, envPath, c, flusher)
	oldProjectPath := project.ProjectPath
	oldDockerImage := project.CurrentImage
	oldDockerContainer := project.ContainerName

	project.Status = &status
	project.ContainerName = &containerName
	project.CurrentImage = &imageName
	project.Type = &source
	project.ProjectPath = &newProjectPath

	if err := RemoveContainer(slug, *oldDockerContainer, c, flusher); err != nil {
		utils.WriteSSEData(project.Slug, []string{"ERROR", fmt.Sprintf("Error removing old docker container: %s", err)}, c, flusher)
		return
	}

	if err := db.UpdateProject(*project); err != nil {
		utils.WriteSSEData(project.Slug, []string{"ERROR", fmt.Sprintf("Error saving project to db: %s", err)}, c, flusher)
		return
	}

	os.RemoveAll(*oldProjectPath)
	if err := db.AddDockerImage(slug, *oldDockerImage); err != nil {
		utils.WriteSSEData(slug, []string{"ERROR", fmt.Sprintf("Error archiving old docker image: %s", err)}, c, flusher)
	}

	logs := utils.GetLogs(project.Slug)
	defer utils.DeleteLogs(project.Slug)
	if len(logs) > 0 {
		if err := db.SaveLogs(project.Slug, logs); err != nil {
			utils.WriteSSEData(slug, []string{"ERROR", fmt.Sprintf("Error saving logs: %s", err)}, c, flusher)
		}
	}

}

func SetEnvironmentVariable(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "SSE not supported",
		})
		return
	}

	var body SetEnvironmentVariablePayload
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		line := strings.Join([]string{strconv.Itoa(int(time.Now().UnixMilli())), "ERROR", err.Error()}, utils.InfraconLogSeparator)
		fmt.Fprintf(c.Writer, "data: %s\n\n", line)
		flusher.Flush()
		return
	}

	project, err := db.GetProject(body.Slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			line := strings.Join([]string{strconv.Itoa(int(time.Now().UnixMilli())), "ERROR", "Project not found"}, utils.InfraconLogSeparator)
			fmt.Fprintf(c.Writer, "data: %s\n\n", line)
			flusher.Flush()
			return
		} else {
			line := strings.Join([]string{strconv.Itoa(int(time.Now().UnixMilli())), "ERROR", err.Error()}, utils.InfraconLogSeparator)
			fmt.Fprintf(c.Writer, "data: %s\n\n", line)
			flusher.Flush()
			return
		}
	}

	envPath := filepath.Join(*project.ProjectPath, ".env")
	if body.Env != "" {
		utils.WriteEnvFile(*project.ProjectPath, body.Env)
	}

	status, _ := RunContainer(body.Slug, *project.CurrentImage, *project.ContainerName, envPath, c, flusher)
	update := utils.Project{
		ID:     project.ID,
		Status: &status,
		Env:    &body.Env,
	}

	if err := db.UpdateProject(update); err != nil {
		utils.WriteSSEData(project.Slug, []string{"ERROR", fmt.Sprintf("Error saving project to db: %s", err)}, c, flusher)
		return
	}

	utils.WriteSSEData(body.Slug, []string{"ERROR", "Environment variable set"}, c, flusher)

}

func RollDeployment(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, map[string]any{
			"status":  false,
			"message": "SSE not supported",
		})
		return
	}

	var body RollDeploymentPayload
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		line := strings.Join([]string{strconv.Itoa(int(time.Now().UnixMilli())), "ERROR", err.Error()}, utils.InfraconLogSeparator)
		fmt.Fprintf(c.Writer, "data: %s\n\n", line)
		flusher.Flush()
		return
	}

	project, err := db.GetProject(body.Slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			line := strings.Join([]string{strconv.Itoa(int(time.Now().UnixMilli())), "ERROR", "Project not found"}, utils.InfraconLogSeparator)
			fmt.Fprintf(c.Writer, "data: %s\n\n", line)
			flusher.Flush()
			return
		} else {
			line := strings.Join([]string{strconv.Itoa(int(time.Now().UnixMilli())), "ERROR", err.Error()}, utils.InfraconLogSeparator)
			fmt.Fprintf(c.Writer, "data: %s\n\n", line)
			flusher.Flush()
			return
		}
	}

	if *project.CurrentImage == body.Tag {
		line := strings.Join([]string{strconv.Itoa(int(time.Now().UnixMilli())), "ERROR", "Current image already deployed"}, utils.InfraconLogSeparator)
		fmt.Fprintf(c.Writer, "data: %s\n\n", line)
		flusher.Flush()
		return
	}

	imageTagExists, err := db.HasDockerImage(project.Slug, body.Tag)
	if err != nil {
		line := strings.Join([]string{strconv.Itoa(int(time.Now().UnixMilli())), "ERROR", fmt.Sprintf("Error getting docker image: %s", err)}, utils.InfraconLogSeparator)
		fmt.Fprintf(c.Writer, "data: %s\n\n", line)
		flusher.Flush()
		return
	}

	if !imageTagExists {
		line := strings.Join([]string{strconv.Itoa(int(time.Now().UnixMilli())), "ERROR", "Docker image not found"}, utils.InfraconLogSeparator)
		fmt.Fprintf(c.Writer, "data: %s\n\n", line)
		flusher.Flush()
		return
	}

	envPath := filepath.Join(*project.ProjectPath, ".env")
	oldContainerName := project.ContainerName
	oldImageName := *project.CurrentImage
	newContainerName := fmt.Sprintf("%s-%s", body.Tag, strconv.Itoa(int(time.Now().UnixMilli())))
	status, _ := RunContainer(body.Slug, body.Tag, *project.ContainerName, envPath, c, flusher)
	update := utils.Project{
		ID:            project.ID,
		Status:        &status,
		ContainerName: &newContainerName,
		CurrentImage:  &body.Tag,
	}

	if err := db.UpdateProject(update); err != nil {
		utils.WriteSSEData(project.Slug, []string{"ERROR", fmt.Sprintf("Error saving project to db: %s", err)}, c, flusher)
		return
	}

	if err := db.AddDockerImage(body.Slug, oldImageName); err != nil {
		utils.WriteSSEData(project.Slug, []string{"ERROR", fmt.Sprintf("Error archiving docker image: %s", err)}, c, flusher)
		return
	}

	if err := RemoveContainer(body.Slug, *oldContainerName, c, flusher); err != nil {
		utils.WriteSSEData(project.Slug, []string{"ERROR", fmt.Sprintf("Error removing old docker container: %s", err)}, c, flusher)
		return
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
