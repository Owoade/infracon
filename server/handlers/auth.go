package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Owoade/infracon/config"
)

type AuthPayload struct {
	Key string `json:"access_key"`
}

const message = "AUTH"

func Authenticate(w http.ResponseWriter, r *http.Request) {
	var body AuthPayload
	err := json.NewDecoder(r.Body).Decode(&body)

	fmt.Println(body)

	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if body.Key == "" {
		http.Error(w, "access_key is required", 400)
		return
	}

	accessKey, err := config.GetCredentials("access_key")
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if accessKey != body.Key {
		http.Error(w, "Invalid access key", 400)
		return
	}

	token := signToken(accessKey, message)

	w.Header().Set("Content-Type", "application/json")

	response := map[string]string{
		"token": token,
	}

	json.NewEncoder(w).Encode(response)

}

func VerifyToken(r *http.Request) (bool, string) {
	authHeader := r.Header.Get("Authorization")
	accessKey, err := config.GetCredentials("access_key")
	if err != nil {
		return false, "Acces key is not set"
	}

	if !verifyToken(accessKey, message, authHeader) {
		return false, "Invalid token"
	}

	return true, ""
}

func signToken(accessKey string, message string) string {
	h := hmac.New(sha256.New, []byte(accessKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

func verifyToken(accessKey, message, token string) bool {
	expected := signToken(accessKey, message)
	return hmac.Equal([]byte(expected), []byte(token))
}
