package utils

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

var (
	nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9]+`)
	multipleHyphensRegex = regexp.MustCompile(`-+`)
)

func Slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	s = nonAlphanumericRegex.ReplaceAllString(s, "-")
	s = multipleHyphensRegex.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func GenerateJwtToken(userID int) (string, error) {
	if err := godotenv.Load(); err != nil {
		return "", err
	}

	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		return "", errors.New("secret key not set")
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(secretKey))
	return tokenString, err
}

func GetSetupKey() (string, error) {
	data, err := os.ReadFile("setup-key.txt")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func GenerateSetupKeyFile() error {
	fileName := "setup-key.txt"
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return err
	}
	key := hex.EncodeToString(bytes)
	return os.WriteFile(fileName, []byte(key), 0600)
}

func IsZipFile(file *multipart.FileHeader) error {
	if filepath.Ext(file.Filename) != ".zip" {
		return errors.New("file extension is not .zip")
	}

	f, err := file.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	buffer := make([]byte, 512)
	n, err := f.Read(buffer)
	if err != nil {
		return err
	}
	buffer = buffer[:n]

	contentType := http.DetectContentType(buffer)
	if contentType != "application/zip" && contentType != "application/octet-stream" {
		return errors.New("invalid file type, not a zip")
	}

	if len(buffer) < 2 || buffer[0] != 'P' || buffer[1] != 'K' {
		return errors.New("invalid zip signature")
	}
	return nil
}

func UnzipFileFromMultipartFile(fh *multipart.FileHeader, dest string) error {
	f, err := fh.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	buf, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	r := bytes.NewReader(buf)
	zr, err := zip.NewReader(r, int64(len(buf)))
	if err != nil {
		return err
	}

	for _, f := range zr.File {
		outPath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(outPath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", outPath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(outPath, f.Mode()); err != nil {
				return err
			}
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, rc)
		return err
	}

	return nil
}

func StringValidator(valueName, value string, config ValidatorConfig) error {
	if config.NotEmpty && value == "" {
		return fmt.Errorf(`The value for "%s" cannot be emty`, valueName)
	}

	if utf8.RuneCountInString(value) < config.MinLength && config.MinLength > 0 {
		return fmt.Errorf(`Expected length of "%s" to be %d recieved %d`, valueName, config.MinLength, utf8.RuneCountInString(value))
	}

	if utf8.RuneCountInString(value) > config.MaxLength && config.MaxLength > 0 {
		return fmt.Errorf(`Expected length of "%s" to be %d recieved %d`, valueName, config.MaxLength, utf8.RuneCountInString(value))
	}

	if len(config.ExpectedValues) > 0 && !Contains(config.ExpectedValues, value) {
		return fmt.Errorf(`Expected values for "%s" are %s`, valueName, strings.Join(config.ExpectedValues, ", "))
	}

	return nil
}

func Contains[T comparable](slice []T, element T) bool {
	for _, v := range slice {
		if v == element {
			return true
		}
	}

	return false
}
