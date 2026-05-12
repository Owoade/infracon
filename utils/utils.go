package utils

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

var (
	nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9]+`)
	multipleHyphensRegex = regexp.MustCompile(`-+`)
	Logs                 map[string][]string
	logsMu               sync.RWMutex
)

const InfraconLogSeparator = "[INFRACON-LOG-SEPARATOR]"

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

func UnzipFileFromMultipartFile(fh *multipart.FileHeader, dest string) (clientFolders []string, err error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(buf)
	zr, err := zip.NewReader(r, int64(len(buf)))
	if err != nil {
		return nil, err
	}

	topLevelDirectories := []string{}

	for _, f := range zr.File {
		outPath := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(outPath, f.Mode()); err != nil {
				return nil, err
			}

			if len(topLevelDirectories) == 0 {
				topLevelDirectories = append(topLevelDirectories, strings.Split(f.Name, string(filepath.Separator))[0])
			} else {
				lastDir := topLevelDirectories[len(topLevelDirectories)-1]
				if !isSubPath(lastDir, f.Name) {
					topLevelDirectories = append(topLevelDirectories, f.Name)
				}
			}
		} else {
			newFile, _ := os.Create(outPath)

			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			io.Copy(newFile, rc)
		}
	}

	return topLevelDirectories, nil
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

func isSubPath(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}

	return !strings.HasPrefix(rel, "..")
}

func PathExists(p string) bool {
	_, err := os.Stat(p)
	if err == nil {
		return true
	}

	return !errors.Is(err, os.ErrNotExist)
}

func WriteSSEData(slug string, d []string, c *gin.Context, flusher http.Flusher) {
	line := strings.Join(append([]string{strconv.Itoa(int(time.Now().UnixMilli()))}, d...), InfraconLogSeparator)
	fmt.Fprintf(c.Writer, "data: %s\n\n", line)
	flusher.Flush()
	logsMu.Lock()
	Logs[slug] = append(Logs[slug], line)
	logsMu.Unlock()
}

func GetLogs(slug string) []string {
	logsMu.RLock()
	defer logsMu.Unlock()
	return Logs[slug]
}

func DeleteLogs(slug string) {
	logsMu.RLock()
	defer logsMu.Unlock()
	delete(Logs, slug)
}

func ExecCommandAndStreamViaSSE(slug string, c *exec.Cmd, gc *gin.Context, f http.Flusher) {
	if _, ok := gc.Writer.(http.Flusher); !ok {
		return
	}

	println("sse function")
	stdout, _ := c.StdoutPipe()
	stderr, _ := c.StderrPipe()

	logChan := make(chan string)
	done := make(chan bool)

	_ = c.Start()

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			logChan <- scanner.Text()
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Println("LOGS >>", scanner.Text())
			logChan <- scanner.Text()
		}
	}()

	go func() {
		c.Wait()
		done <- true
	}()

	for {
		select {
		case line := <-logChan:
			WriteSSEData(slug, []string{"BUILD", line}, gc, f)

		case <-done:
			WriteSSEData(slug, []string{"BUILD", "BUILD finished"}, gc, f)
			return

		case <-gc.Request.Context().Done():
			return
		}
	}

}

func GetDockerImage(imageName string) (di *DockerImage, err error) {
	cmd := exec.Command("docker", "image", "inspect", imageName)

	output, err := cmd.Output()

	if err != nil {
		return nil, fmt.Errorf("failed to inspect image: %w", err)
	}

	var result []DockerImage
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no image data found")
	}

	return &result[0], nil
}

func GetDeploymentStatusetDockerContainer(name string) (*DockerContainer, error) {

	cmd := exec.Command("docker", "inspect", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}
	var containers []DockerContainer
	if err := json.Unmarshal(output, &containers); err != nil {
		return nil, fmt.Errorf("failed to parse docker inspect output: %w", err)
	}
	if len(containers) == 0 {
		return nil, fmt.Errorf("no container found")
	}
	return &containers[0], nil
}

func WriteEnvFile(destination, env string) (string, error) {
	if err := os.MkdirAll(destination, 0755); err != nil {
		return "", err
	}
	merged := InjectPort(env, 3000)
	envPath := filepath.Join(destination, ".env")
	if err := os.WriteFile(envPath, []byte(merged), 0644); err != nil {
		return "", err
	}
	return envPath, nil
}

func InjectPort(env string, port int) string {
	lines := strings.Split(env, "\n")

	found := false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "PORT=") {
			lines[i] = fmt.Sprintf("PORT=%d", port)
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, fmt.Sprintf("PORT=%d", port))
	}

	return strings.Join(lines, "\n")
}

func PullFromGithub(p PullfromGithub) (commitHash string, err error) {
	if p.AccessToken == "" {
		return "", errors.New("GITHUB_ACCESS_TOKEN is not set")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/%s", p.Owner, p.Repo, p.Ref)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+p.AccessToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var response map[string]string
		json.NewDecoder(resp.Body).Decode(&response)
		return "", fmt.Errorf("Github error: %s", response["message"])
	}

	var buf bytes.Buffer
	size, err := io.Copy(&buf, resp.Body)
	if err != nil {
		return "", err
	}

	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), size)
	if err != nil {
		return "", err
	}

	for _, f := range r.File {
		fpath := filepath.Join(p.Destination, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(p.Destination)+string(os.PathSeparator)) {
			return "", fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return "", err
		}

		inFile, err := f.Open()
		if err != nil {
			return "", err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			inFile.Close()
			return "", err
		}

		_, err = io.Copy(outFile, inFile)
		inFile.Close()
		outFile.Close()
		if err != nil {
			return "", err
		}
	}

	return r.Comment, nil
}
