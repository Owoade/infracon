package handlers

import (
	"encoding/json"
	"net/http"
)

type DeployApplicationPayload struct {
	Type string 		`json:"type"` // "native" or "docker"
	// If not passed as a pointer struct will never be empty hence failing nil check
	Docker *DockerConfig `json:"docker_config,omitempty"`
	Native *NativeConfig `json:"native_config,omitempty"`
}

type DockerConfig struct {
	DockerfilePath string `json:"dockerfile_path"`
}

type NativeConfig struct {
	ApplicationType string `json:"application_type"` // "node", "go" e.t.c
	BuildCommand    string `json:"build_command"`
	RunCommand      string `json:"run_command"`
	Port			int	   `json:"port"`
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

}


