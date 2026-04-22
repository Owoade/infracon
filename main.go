package main

import (
	"infracon/auth"
	"infracon/db"
	"infracon/project"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

func init() {
	log.SetFlags(log.Ldate | log.Lshortfile)
	godotenv.Load()
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
				top_level_directories TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`,
	)
}

func main() {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.POST("/api/auth/sign-in", auth.Signin)
	router.POST("/api/auth/sign-up", auth.SignUp)
	router.POST("/api/auth/forgot-password", auth.ResetPassword)

	projectRouter := router.Group("/api/project")
	projectRouter.Use(Authenticate)

	projectRouter.POST("/", project.CreateProject)
	projectRouter.GET("/:slug", project.GetProject)
	projectRouter.POST("/source", project.AddProjectSource)
	projectRouter.POST("/env", project.SetEnvironmentVariable)
	projectRouter.POST("/github/token", project.AddGithubToken)
	projectRouter.GET("/github/token", project.GetGithubTokens)
	projectRouter.POST("/github/repos", project.GetGithubRepos)
	projectRouter.POST("/github/repos/branches", project.GetGithubRepoBranches)

	router.Run(":3000")
}

func Authenticate(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"message": "Authorization header missing",
			"status":  false,
		})
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"message": "Invalid authorization format",
			"status":  false,
		})
		return
	}

	tokenString := parts[1]

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": "JWT secret not set",
			"status":  false,
		})
		return
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"message": "Invalid or expired token",
			"status":  false,
		})
		return
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		c.Set("user", claims)
	}

	c.Next()
}
