package db

import (
	"database/sql"
	"errors"
	"fmt"
	"infracon/utils"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func GetDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./infracon.db")
	if err != nil {
		return nil, err
	}
	return db, nil
}

func GetGithubToken() (string, error) {
	db, err := GetDatabase()
	if err != nil {
		return "", err
	}

	var token string
	if err := db.QueryRow("SELECT token FROM github_tokens").Scan(&token); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	return token, nil
}

func CreateProject(p utils.Project) (int, error) {
	db, err := GetDatabase()
	if err != nil {
		return 0, err
	}

	var id int
	if err := db.QueryRow("INSERT INTO projects (name, slug, type) VALUES ($1, $2, $3) RETURNING id", p.Name, p.Slug, p.Type).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func UpdateProject(p utils.Project) error {
	db, err := GetDatabase()
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		UPDATE projects SET
			name = CASE 
				WHEN $1 IS NOT NULL AND $1 <> '' THEN $1 
				ELSE name 
			END,
			slug = CASE 
				WHEN $2 IS NOT NULL AND $2 <> '' THEN $2 
				ELSE slug 
			END,
			type = CASE 
				WHEN $3 IS NOT NULL THEN $3 
				ELSE type 
			END,
			env = CASE 
				WHEN $4 IS NOT NULL THEN $4 
				ELSE env 
			END,
			github_repo = CASE 
				WHEN $5 IS NOT NULL THEN $5 
				ELSE github_repo 
			END,
			project_path = CASE 
				WHEN $6 IS NOT NULL THEN $6 
				ELSE project_path 
			END,
			status = CASE 
				WHEN $7 IS NOT NULL THEN $7 
				ELSE status 
			END,
			container_name = CASE 
				WHEN $8 IS NOT NULL THEN $8 
				ELSE container_name 
			END,
			current_image = CASE 
				WHEN $9 IS NOT NULL THEN $9 
				ELSE current_image 
			END,
			updated_at = NOW()
		WHERE id = $10
	`,
		p.Name,
		p.Slug,
		p.Type,
		p.Env,
		p.GithubRepo,
		p.ProjectPath,
		p.Status,
		p.ContainerName,
		p.CurrentImage,
		p.ID,
	)

	return err
}

func SaveLogs(slug string, logs []string) error {
	db, err := GetDatabase()
	if err != nil {
		return err
	}

	var values []string
	for _, log := range logs {
		values = append(values, fmt.Sprintf("('%s', '%s')", slug, log))
	}

	query := fmt.Sprintf(`
		INSERT INTO logs (project_slug, log)
		VALUES %s
	`, strings.Join(values, ", "))

	_, err = db.Exec(query)
	if err != nil {
		return err
	}

	return err
}

func GetProject(slug string) (*utils.Project, error) {
	db, err := GetDatabase()
	if err != nil {
		return nil, err
	}

	var p utils.Project
	err = db.QueryRow("SELECT name, slug, type, env, project_path, status, container_name, current_image from projects where slug = $1", slug).Scan(&p.Name, &p.Slug, &p.Type, &p.Env, &p.ProjectPath, &p.Status, &p.Status, &p.ContainerName, &p.CurrentImage)
	return &p, err
}

func GetProjects() ([]utils.Project, error) {
	db, err := GetDatabase()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query("SELECT name, slug, type, env, project_path, status, container_name, current_image, created_at, updated_at FROM projects ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}

	var projects []utils.Project
	for rows.Next() {
		var p utils.Project
		err = rows.Scan(&p.Name, &p.Slug, &p.Type, &p.Env, &p.ProjectPath, &p.Status, &p.Status, &p.ContainerName, &p.CurrentImage, &p.CreatedAt, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}

	return projects, nil
}

func AddDockerImage(slug, imageName string) error {
	db, err := GetDatabase()
	if err != nil {
		return err
	}

	return db.QueryRow("INSERT INTO docker_images (project_slug, image_tag) VALUES($1, $2) RETURNING id", slug, imageName).Scan(new(int))
}

func HasDockerImage(slug string, imageTag string) (bool, error) {
	db, err := GetDatabase()
	if err != nil {
		return false, err
	}
	defer db.Close()

	var count int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM docker_images WHERE slug = ? AND image_tag = ?`,
		slug, imageTag,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// func GetDockerImages(slug string) ([]utils.DockerImage, error) {
// 	db, err := GetDatabase()
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer db.Close()

// 	rows, err := db.Query(
// 		`SELECT image_tag FROM docker_images WHERE project_slug = $1 ORDER BY created_at DESC`,
// 		slug,
// 	)
// }
