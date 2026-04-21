package auth

import (
	"database/sql"
	"errors"
	"infracon/db"
	"infracon/utils"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func SignUp(c *gin.Context) {
	db, err := db.GetDatabase()
	if err != nil {
		log.Printf("db error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	if err := db.QueryRow("SELECT * FROM users").Scan(new(int)); err != nil {
		log.Println(err)
		if !errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "Admin user exists already",
				"status":  false,
			})
			return
		}
	}

	var body AuthPayload
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid payload",
			"status":  false,
			"error":   err,
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("password error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}
	body.Password = string(hash)
	var userId int
	if err := db.QueryRow(
		`
			INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id;	
		`,
		body.Email,
		body.Password,
	).Scan(&userId); err != nil {
		log.Printf("insert query error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	token, err := utils.GenerateJwtToken(userId)
	if err != nil {
		log.Printf("generating token error: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Error generating token",
			"status":  false,
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Operation successful",
		"status":  true,
		"data": gin.H{
			"token": token,
		},
	})

}

func Signin(c *gin.Context) {
	var userId int
	var password string

	db, err := db.GetDatabase()
	if err != nil {
		log.Printf("db error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	var body AuthPayload
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid payload",
			"status":  false,
			"error":   err,
		})
		return
	}

	if err = db.QueryRow(
		`
			SELECT id, password FROM users WHERE email = $1	
		`,
		body.Email,
	).Scan(&userId, &password); err != nil {
		log.Printf("query error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(password), []byte(body.Password))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid password",
			"status":  false,
		})
		return
	}

	token, err := utils.GenerateJwtToken(userId)
	if err != nil {
		log.Printf("generating token error: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Error generating token",
			"status":  false,
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Operation successful",
		"status":  true,
		"data": gin.H{
			"token": token,
		},
	})

}

func ResetPassword(c *gin.Context) {
	var body ForgotPasswordPayload
	if err := c.ShouldBindBodyWithJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Invalid payload",
		})
		return
	}

	key, err := utils.GetSetupKey()
	if err != nil {
		log.Printf("setup key error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  false,
			"message": "Something went wrong",
		})
		return
	}

	if strings.TrimSpace(key) != body.SetupKey {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Invalid setup key",
		})
		return
	}

	db, err := db.GetDatabase()
	if err != nil {
		log.Printf("db error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("password error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}
	body.Password = string(hash)

	if err := db.QueryRow("UPDATE users SET password = $1 RETURNING 1", body.Password).Scan(new(int)); err != nil {
		log.Printf("query error: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
			"status":  false,
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":  true,
		"message": "Operation successful",
	})

	if err := utils.GenerateSetupKeyFile(); err != nil {
		log.Printf("error generating setup key %s", err)
	}
}
