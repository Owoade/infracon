package project

import (
	"errors"
	"fmt"
	"infracon/utils"
	"net/http"
	"os/exec"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func BuildImage(imageName, src string, useCustomDockerfile bool, c *gin.Context, flusher http.Flusher) {
	if useCustomDockerfile {
		if !utils.PathExists(filepath.Join(src, "Dockerfile")) {
			utils.WriteSSEData([]string{"ERROR", "Custom docker file doesn't exist"}, c, flusher)
			return
		}

		cmd := exec.Command(
			"docker",
			"build",
			"-t", imageName,
			"-f", filepath.Join(src, "Dockerfile"), src,
		)

		utils.ExecCommandAndStreamViaSSE(cmd, c, flusher)
	} else {
		cmd := exec.Command(
			"railpack",
			"build",
			src,
			"--name",
			imageName,
			"--verbose",
		)

		utils.ExecCommandAndStreamViaSSE(cmd, c, flusher)
	}

}

func RunContainer(imageName, containerName, envPath string, c *gin.Context, f http.Flusher) (string, error) {
	cmd := exec.Command(
		"docker",
		"run",
		"--name", containerName,
		"--env-file", envPath,
		imageName,
	)

	utils.ExecCommandAndStreamViaSSE(cmd, c, f)

	ct, err := utils.GetDeploymentStatusetDockerContainer(containerName)
	if err != nil {
		utils.WriteSSEData([]string{"ERROR", fmt.Sprintf("Error running docker container: %s", err)}, c, f)
		return "", err
	}

	if !ct.State.Running {
		utils.WriteSSEData([]string{"ERROR", "Container is not running"}, c, f)
		return "", errors.New("Container is not running")
	}

	return ct.State.Status, nil
}
