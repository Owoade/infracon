package main

import (
	"infracon/auth"
	"infracon/db"

	"github.com/gin-gonic/gin"
)

func init() {
	db, err := db.GetDatabase()
	if err != nil {
		panic(err)
	}
	db.Exec(
		`
			CREATE TABLE IF NOT EXISTS users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				email TEXT NOT NULL UNIQUE,
				password TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);

			CREATE TABLE IF NOT EXISTS github_tokens (
				user_id INTEGER PRIMARY KEY AUTOINCREMENT,
				token TEXT NOT NULL,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);

			CREATE TABLE IF NOT EXISTS projects (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				slug TEXT NOT NULL UNIQUE,
				type TEXT,
				env TEXT,
				github_repo TEXT,
				project_path TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`,
	)
}

func main() {
	router := gin.Default()
	router.POST("/api/auth/sign-in", auth.Signin)
	router.POST("/api/auth/sign-up", auth.SignUp)
	router.POST("/api/auth/forgot-password", auth.ResetPassword)

	router.Run(":3000")
}
