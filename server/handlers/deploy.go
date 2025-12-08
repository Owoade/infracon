package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Owoade/infracon/server/utils"
)

type DeployApplicationPayload struct {
	Type          string        `json:"type"` // "native" or "docker"
	ApplicationID string        `json:"application_id"`
	Docker        *DockerConfig `json:"docker_config,omitempty"`
	Native        *NativeConfig `json:"native_config,omitempty"`
}

type DockerConfig struct {
	DockerfilePath string `json:"dockerfile_path"`
}

type NativeConfig struct {
	ApplicationType string `json:"application_type"` // "node", "go" e.t.c
	BuildCommand    string `json:"build_command"`
	RunCommand      string `json:"run_command"`
	Port            int    `json:"port"`
}

type GenerateDockerfileContent struct {
	Image        string
	BuildCommand string
	RunCommand   string
	InternalPort int
}

func (handler *ServerHandler) DeployApplication(w http.ResponseWriter, r *http.Request) {
	var body DeployApplicationPayload

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "Error decoding request body", 400)
		return
	}

	if body.Type == "docker" && body.Docker == nil {
		http.Error(w, "`docker_config` is required", 400)
		return
	}

	if body.Type == "native" && body.Native == nil {
		http.Error(w, "`native_config` is required", 400)
		return
	}

	if body.Type != "docker" && body.Type != "native" {
		http.Error(w, "Invalid type", 400)
		return
	}

	application, err := handler.Repo.GetApplicationFromDB(body.ApplicationID)
	if err != nil {
		http.Error(w, "Error getting application", 500)
		return
	}
	if application == nil {
		http.Error(w, "Invalid application id", 500)
		return
	}

	if body.Type == "native" {
		dockerImage := dockerImageMap[body.Native.ApplicationType]
		if dockerImage == "" {
			http.Error(w, "Unable to resolve docker image", 500)
			return
		}

		dockerContainerInteralPort := 4300
		dockerFileContent := generateDockerfileContent(GenerateDockerfileContent{
			InternalPort: dockerContainerInteralPort,
			Image:        dockerImage,
			RunCommand:   body.Native.RunCommand,
		})

		home, _ := os.UserHomeDir()
		dockerFilePath := home + "/" + "infracon-apps" + "/" + application.Name + "/" + "Dockerfile.ic"
		if utils.FileExists(dockerFilePath) {
			os.Remove(dockerFilePath)
		}

		file, err := os.Create(dockerFilePath)
		if err != nil {
			http.Error(w, "Unable to create dynamic docker file ", 500)
			return
		}
		defer file.Close()

		writter := bufio.NewWriter(file)
		for _, line := range dockerFileContent {
			writter.WriteString(line)
		}
		writter.Flush()

	}
}

func generateDockerfileContent(payload GenerateDockerfileContent) []string {
	content := []string{}
	content = append(content, fmt.Sprintf("FROM %s\n", payload.Image))
	content = append(content, "WORKDIR /app\n")
	content = append(content, "COPY . .\n")
	content = append(content, fmt.Sprintf("RUN %s\n", payload.BuildCommand))
	content = append(content, fmt.Sprintf("EXPOSE %d\n", payload.InternalPort))
	content = append(content, fmt.Sprintf("CMD [%s]\n", strings.Join(strings.Split(payload.RunCommand, " "), ", ")))
	return content
}
