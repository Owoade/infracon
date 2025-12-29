package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Owoade/infracon/server/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/go-connections/nat"
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

type BuildMessage struct {
	Stream string `json:"stream"`
	Error  string `json:"error"`
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

	home, _ := os.UserHomeDir()

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
			BuildCommand: body.Native.BuildCommand,
		})

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

		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			http.Error(w, "unable to initiate docker client", 500)
			return
		}
		defer cli.Close()

		contextPath := fmt.Sprintf("%s/infracon-apps/%s", home, application.ClientPath)

		tar, err := archive.TarWithOptions(contextPath, &archive.TarOptions{})
		if err != nil {
			http.Error(w, "unable to create tar", 404)
			return
		}
		defer tar.Close()

		imageName := fmt.Sprintf("infracon-image-%s:latest", application.ID)

		buildOptions := types.ImageBuildOptions{
			Tags:       []string{imageName},
			Dockerfile: "Dockerfile.ic", // Name of Dockerfile in context
			Remove:     true,            // Remove intermediate containers
			NoCache:    false,           // Use cache if available
			BuildArgs: map[string]*string{
				"BUILD_DATE": stringPtr(time.Now().Format(time.RFC3339)),
			},
		}

		buildResp, err := cli.ImageBuild(context.Background(), tar, buildOptions)
		fmt.Println(contextPath)
		if err != nil {
			log.Println(err)
			http.Error(w, "unable build docker image", 500)
			return
		}
		defer buildResp.Body.Close()

		// Display build output
		var buildSuccess bool
		decoder := json.NewDecoder(buildResp.Body)
		for {
			var msg BuildMessage
			if err := decoder.Decode(&msg); err != nil {
				if err == io.EOF {
					break
				}
				panic(err)
			}
			if msg.Error != "" {
				fmt.Printf("BUILD ERROR: %s\n", msg.Error)
				panic(fmt.Errorf("build failed: %s", msg.Error))
			}
			if msg.Stream != "" {
				fmt.Print(msg.Stream)
				// Check for successful build completion
				if contains(msg.Stream, "Successfully built") || contains(msg.Stream, "Successfully tagged") {
					buildSuccess = true
				}
			}
		}

		if !buildSuccess {
			panic("Build completed but no success message found")
		}

		containerConfig := &container.Config{
			Image: imageName,
			Env: []string{
				"APP_ENV=production",
			},
			ExposedPorts: nat.PortSet{
				"4300/tcp": struct{}{},
			},
		}

		hostConfig := &container.HostConfig{
			PortBindings: nat.PortMap{
				"4300/tcp": []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: "4300",
					},
				},
			},
			AutoRemove: true,
		}

		// Create the container
		resp, err := cli.ContainerCreate(context.Background(), containerConfig, hostConfig, nil, nil, fmt.Sprintf("container-%s", application.ClientPath))
		if err != nil {
			panic(err)
		}

		fmt.Println(resp)

		if err := cli.ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
			panic(err)
		}

		fmt.Printf("Container started successfully\n")
		fmt.Printf("Access app at http://localhost:8080\n")

	}

	if body.Type == "docker" {
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			http.Error(w, "unable to initiate docker client", 500)
			return
		}
		defer cli.Close()

		contextPath := fmt.Sprintf("%s/infracon-apps/%s", home, application.ClientPath)

		tar, err := archive.TarWithOptions(contextPath, &archive.TarOptions{})
		if err != nil {
			http.Error(w, "unable to create tar", 404)
			return
		}
		defer tar.Close()

		imageName := fmt.Sprintf("infracon-image-%s:latest", application.ID)

		buildOptions := types.ImageBuildOptions{
			Tags:       []string{imageName},
			Dockerfile: body.Docker.DockerfilePath, // Name of Dockerfile in context
			Remove:     true,                       // Remove intermediate containers
			NoCache:    false,                      // Use cache if available
			BuildArgs: map[string]*string{
				"BUILD_DATE": stringPtr(time.Now().Format(time.RFC3339)),
			},
		}

		buildResp, err := cli.ImageBuild(context.Background(), tar, buildOptions)
		fmt.Println(contextPath)
		if err != nil {
			log.Println(err)
			http.Error(w, "unable build docker image", 500)
			return
		}
		defer buildResp.Body.Close()

		// Display build output
		var buildSuccess bool
		decoder := json.NewDecoder(buildResp.Body)
		for {
			var msg BuildMessage
			if err := decoder.Decode(&msg); err != nil {
				if err == io.EOF {
					break
				}
				panic(err)
			}
			if msg.Error != "" {
				fmt.Printf("BUILD ERROR: %s\n", msg.Error)
				panic(fmt.Errorf("build failed: %s", msg.Error))
			}
			if msg.Stream != "" {
				fmt.Print(msg.Stream)
				// Check for successful build completion
				if contains(msg.Stream, "Successfully built") || contains(msg.Stream, "Successfully tagged") {
					buildSuccess = true
				}
			}
		}

		if !buildSuccess {
			panic("Build completed but no success message found")
		}

		containerConfig := &container.Config{
			Image: imageName,
			Env: []string{
				"APP_ENV=production",
			},
			ExposedPorts: nat.PortSet{
				"8080/tcp": struct{}{},
			},
		}

		hostConfig := &container.HostConfig{
			PortBindings: nat.PortMap{
				"8080/tcp": []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: "8080",
					},
				},
			},
			AutoRemove: true,
		}

		// Create the container
		resp, err := cli.ContainerCreate(context.Background(), containerConfig, hostConfig, nil, nil, fmt.Sprintf("container-%s", application.ClientPath))
		if err != nil {
			panic(err)
		}

		fmt.Println(resp)

		if err := cli.ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
			panic(err)
		}

		fmt.Printf("Container started successfully\n")
		fmt.Printf("Access app at http://localhost:8080\n")
	}

}

func stringPtr(s string) *string {
	return &s
}

func generateDockerfileContent(payload GenerateDockerfileContent) []string {
	content := []string{}
	content = append(content, fmt.Sprintf("FROM %s\n", payload.Image))
	content = append(content, "WORKDIR /app\n")
	content = append(content, "COPY . .\n")
	content = append(content, fmt.Sprintf("RUN %s\n", payload.BuildCommand))
	content = append(content, fmt.Sprintf("EXPOSE %d\n", payload.InternalPort))
	content = append(
		content,
		fmt.Sprintf(
			"CMD [%s]\n",
			strings.Join(
				utils.Map(
					strings.Split(payload.RunCommand, " "),
					func(cmd string) string {
						return fmt.Sprintf(`"%s"`, cmd)
					},
				),
				", ",
			),
		))
	return content
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
