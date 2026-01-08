package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Owoade/infracon/db"
	"github.com/Owoade/infracon/server/utils"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	Email      string `json:"email"`
	IsRootUser bool   `json:"is_root"`
	jwt.RegisteredClaims
}

type AuthPayload struct {
	Otp string `json:"otp"`
	db.UserModel
}

const message = "AUTH"

func (handler *ServerHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body db.UserModel
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.RespondToCLient(
			w,
			utils.ResponsePayload{
				StatusCode: http.StatusBadRequest,
				Message:    "Invalid payload",
				Status:     false,
				Data:       nil,
			},
		)
		return
	}

	user, err := handler.Repo.GetUser(body.Email)
	if err != nil {
		log.Println(err)
		utils.RespondToCLient(
			w,
			utils.ResponsePayload{
				StatusCode: http.StatusBadRequest,
				Message:    "Error getting user info",
				Status:     false,
				Data:       nil,
			},
		)
		return
	}

	if user == nil {
		utils.RespondToCLient(
			w,
			utils.ResponsePayload{
				StatusCode: http.StatusNotFound,
				Message:    "User not found",
				Status:     false,
				Data:       nil,
			},
		)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))
	if err != nil {
		utils.RespondToCLient(
			w,
			utils.ResponsePayload{
				StatusCode: http.StatusBadRequest,
				Message:    "Invalid passeord",
				Status:     false,
				Data:       nil,
			},
		)
		return
	}

	claims := &Claims{
		Email:      body.Email,
		IsRootUser: user.IsRoot,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * 30 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "Infracon",
		},
	}

	key := []byte(os.Getenv("JWT_SECRET"))

	signing := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := signing.SignedString(key)

	utils.RespondToCLient(
		w,
		utils.ResponsePayload{
			StatusCode: http.StatusOK,
			Message:    "OK",
			Status:     true,
			Data:       token,
		},
	)

}

func (handler *ServerHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	var body AuthPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.RespondToCLient(
			w,
			utils.ResponsePayload{
				StatusCode: http.StatusBadRequest,
				Message:    "Invalid payload",
				Status:     false,
				Data:       nil,
			},
		)
		return
	}

	valid, err := validateOTP(body.Otp)
	if err != nil || !valid {
		log.Println(err)
		utils.RespondToCLient(
			w,
			utils.ResponsePayload{
				StatusCode: http.StatusBadRequest,
				Message:    "Invalid OTP",
				Status:     false,
				Data:       nil,
			},
		)
		return
	}

	user, err := handler.Repo.GetUser(body.Email)
	if err != nil {
		log.Println(err)
		utils.RespondToCLient(
			w,
			utils.ResponsePayload{
				StatusCode: http.StatusBadRequest,
				Message:    "Error getting user info",
				Status:     false,
				Data:       nil,
			},
		)
		return
	}

	if user != nil {
		utils.RespondToCLient(
			w,
			utils.ResponsePayload{
				StatusCode: http.StatusBadRequest,
				Message:    "User already exists",
				Status:     false,
				Data:       nil,
			},
		)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err)
		utils.RespondToCLient(
			w,
			utils.ResponsePayload{
				StatusCode: http.StatusBadRequest,
				Message:    "Error processing request",
				Status:     false,
				Data:       nil,
			},
		)
		return
	}

	password := string(hash)
	isRootUser, err := handler.Repo.CreateUser(
		body.Email,
		password,
	)

	if err != nil {
		log.Println(err)
		utils.RespondToCLient(
			w,
			utils.ResponsePayload{
				StatusCode: http.StatusBadRequest,
				Message:    "Error saving user details",
				Status:     false,
				Data:       nil,
			},
		)
		return
	}

	claims := &Claims{
		Email:      body.Email,
		IsRootUser: isRootUser,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * 30 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "Infracon",
		},
	}

	key := []byte(os.Getenv("JWT_SECRET"))

	signing := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := signing.SignedString(key)

	utils.RespondToCLient(
		w,
		utils.ResponsePayload{
			StatusCode: http.StatusOK,
			Message:    "Operation successful",
			Status:     true,
			Data: map[string]any{
				"token":   token,
				"is_root": isRootUser,
			},
		},
	)

	deleteOTPFile()

}

func (handler *ServerHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var body AuthPayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.RespondToCLient(
			w,
			utils.ResponsePayload{
				StatusCode: http.StatusBadRequest,
				Message:    "Invalid payload",
				Status:     false,
				Data:       nil,
			},
		)
		return
	}

	valid, err := validateOTP(body.Otp)
	if err != nil || !valid {
		log.Println(err)
		utils.RespondToCLient(
			w,
			utils.ResponsePayload{
				StatusCode: http.StatusBadRequest,
				Message:    "Invalid OTP",
				Status:     false,
				Data:       nil,
			},
		)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err)
		utils.RespondToCLient(
			w,
			utils.ResponsePayload{
				StatusCode: http.StatusBadRequest,
				Message:    "Error processing request",
				Status:     false,
				Data:       nil,
			},
		)
		return
	}

	password := string(hash)
	if err = handler.Repo.UpdateUserPassword(
		body.Email,
		password,
	); err != nil {
		log.Println(err)
		utils.RespondToCLient(
			w,
			utils.ResponsePayload{
				StatusCode: http.StatusBadRequest,
				Message:    "Error saving user details",
				Status:     false,
				Data:       nil,
			},
		)
		return
	}

	utils.RespondToCLient(
		w,
		utils.ResponsePayload{
			StatusCode: http.StatusOK,
			Message:    "Password changed",
			Status:     true,
		},
	)

	deleteOTPFile()
}

func validateOTP(inputOTP string) (bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}

	otpPath := filepath.Join(homeDir, "infracon-apps", "otp.txt")

	data, err := os.ReadFile(otpPath)
	if err != nil {
		return false, err
	}

	storedOTP := strings.TrimSpace(string(data))
	inputOTP = strings.TrimSpace(inputOTP)

	return storedOTP == inputOTP, nil
}

func deleteOTPFile() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	otpPath := filepath.Join(homeDir, "infracon-apps", "otp.txt")

	// If file doesn't exist, treat as success
	if _, err := os.Stat(otpPath); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(otpPath)
}
