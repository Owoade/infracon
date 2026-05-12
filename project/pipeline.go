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

func BuildImage(slug, imageName, src string, useCustomDockerfile bool, c *gin.Context, flusher http.Flusher) {
	if useCustomDockerfile {
		if !utils.PathExists(filepath.Join(src, "Dockerfile")) {
			utils.WriteSSEData(slug, []string{"ERROR", "Custom docker file doesn't exist"}, c, flusher)
			return
		}

		cmd := exec.Command(
			"docker",
			"build",
			"-t", imageName,
			"-f", filepath.Join(src, "Dockerfile"), src,
		)

		utils.ExecCommandAndStreamViaSSE(slug, cmd, c, flusher)
	} else {
		cmd := exec.Command(
			"railpack",
			"build",
			src,
			"--name",
			imageName,
			"--verbose",
		)

		utils.ExecCommandAndStreamViaSSE(slug, cmd, c, flusher)
	}

}

func RunContainer(slug, imageName, containerName, envPath string, c *gin.Context, f http.Flusher) (string, error) {
	cmd := exec.Command(
		"docker",
		"run",
		"--name", containerName,
		"--env-file", envPath,
		imageName,
	)

	utils.ExecCommandAndStreamViaSSE(slug, cmd, c, f)

	ct, err := utils.GetDeploymentStatusetDockerContainer(containerName)
	if err != nil {
		utils.WriteSSEData(slug, []string{"ERROR", fmt.Sprintf("Error running docker container: %s", err)}, c, f)
		return "", err
	}

	if !ct.State.Running {
		utils.WriteSSEData(slug, []string{"ERROR", "Container is not running"}, c, f)
		return "", errors.New("Container is not running")
	}

	return ct.State.Status, nil
}

func RemoveContainer(slug, containerName string, c *gin.Context, f http.Flusher) error {

	cmd := exec.Command(
		"docker",
		"rm",
		"-f", // force stop and remove if running
		containerName,
	)
	utils.ExecCommandAndStreamViaSSE(slug, cmd, c, f)
	_, err := utils.GetDeploymentStatusetDockerContainer(containerName)
	if err == nil {
		utils.WriteSSEData(slug, []string{"ERROR", "Container still exists after removal"}, c, f)
		return errors.New("failed to remove container")
	}
	utils.WriteSSEData(slug, []string{"INFO", "Container removed successfully"}, c, f)
	return nil

}
