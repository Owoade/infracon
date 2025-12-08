package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

type ConnectApplicationPayload struct {
	Type          string `json:"type"` // "new" or "existing"
	Path          string `json:"path,omitempty"`
	Name          string `json:"name,omitempty"`
	ApplicationID string `json:"application_id,omitempty"`
}

func (handler *ServerHandler) Connect(w http.ResponseWriter, r *http.Request) {

	if validAuth, err := VerifyToken(r); !validAuth {
		http.Error(w, err, 400)
		return
	}

	var body ConnectApplicationPayload

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Error decoding request body", 400)
		return
	}

	if body.Type == "" {
		http.Error(w, "`type` is required", 400)
		return
	}

	if body.Type != "new" || body.Type == "existing" {
		http.Error(w, "`type` must be either of value 'new' or 'existing'", 400)
		return
	}

	if body.Type == "existing" && body.ApplicationID == "" {
		http.Error(w, "`application_id` is required", 400)
		return
	}

	if body.Type == "new" && body.Name == "" {
		http.Error(w, "`name` is required", 400)
		return
	}

	if body.Type == "new" && body.Path == "" {
		http.Error(w, "`path` is required", 400)
		return
	}

	if body.Type == "new" {
		applicationID, err := connectNewProject(handler, body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		response := map[string]any{
			"application_id": applicationID,
		}

		json.NewEncoder(w).Encode(response)

		return

	}

	if err := connectExistingProject(handler, body.ApplicationID, body.Path); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	response := map[string]any{
		"application_id": body.ApplicationID,
	}

	json.NewEncoder(w).Encode(response)

}

func connectNewProject(handler *ServerHandler, payload ConnectApplicationPayload) (applicationId string, err error) {

	println("Connecting new project")

	applicationID := uuid.New().String()
	clientPath := payload.Path
	timestamp := time.Now().UTC()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.New("unable to resolve home directory path")
	}

	projectPath := homeDir + "/" + "infracon-apps" + "/" + payload.Path

	if err = os.MkdirAll(projectPath, 0755); err != nil {
		return "", errors.New("unable to create project folder")
	}

	existingProjectByClientPath, err := handler.Repo.GetApplicationFromDBByClientPath(clientPath)
	if err != nil {
		log.Fatal(err.Error())
		return "", errors.New("unable to fetch project by client path")
	}

	if existingProjectByClientPath == nil {

		sqlStatement := `
			INSERT INTO apps (
				id,
				name,
				path,
				client_path,
				created_at,
				updated_at
			) 
			VALUES (?,?,?,?,?,?)
		`
		if _, err := handler.DB.Exec(
			sqlStatement,
			applicationID,
			payload.Name,
			projectPath,
			clientPath,
			timestamp,
			timestamp,
		); err != nil {
			return "", errors.New("unable to create folder entry in db")
		}

		return applicationID, nil

	}

	return existingProjectByClientPath.ID, nil

}

func connectExistingProject(handler *ServerHandler, applicationID string, clientPath string) error {

	existingApplication, err := handler.Repo.GetApplicationFromDB(applicationID)

	if err != nil {
		return err
	}

	if existingApplication == nil {
		return errors.New("project not found")
	}

	if existingApplication.ClientPath != clientPath {
		timestamp := time.Now().UTC()
		if _, err := handler.DB.Exec("UPDATE apps SET client_path=?, updated_at=? where id=?", clientPath, timestamp, applicationID); err != nil {
			return errors.New("unable to update existing entry in db")
		}
	}

	return nil
}
