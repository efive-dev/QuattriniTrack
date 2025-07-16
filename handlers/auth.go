package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"quattrinitrack/config"
	"quattrinitrack/database"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

// UserStore is an interface for interacting with users in the database.
type UserStore interface {
	CreateUser(ctx context.Context, arg database.CreateUserParams) (database.CreateUserRow, error)
	GetUserByEmail(ctx context.Context, email string) (database.User, error)
}

// AuthRequest is the structure for the user credentials.
type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register handles user registration.
func Register(queries UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var reqAuth AuthRequest
		err := json.NewDecoder(req.Body).Decode(&reqAuth)
		if err != nil {
			http.Error(w, "Status Bad Request", http.StatusBadRequest)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(reqAuth.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("error in encrypting password %v", err)
			return
		}

		user, err := queries.CreateUser(req.Context(), database.CreateUserParams{
			Email:        reqAuth.Email,
			PasswordHash: string(hash),
		})
		if err != nil {
			log.Printf("error in creating user %v", err)
			http.Error(w, "Email already used", http.StatusConflict)
			return
		}

		json.NewEncoder(w).Encode(user)
	}
}

// Login handles user login.
func Login(queries UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var reqAuth AuthRequest
		err := json.NewDecoder(req.Body).Decode(&reqAuth)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		user, err := queries.GetUserByEmail(req.Context(), reqAuth.Email)
		if err != nil {
			http.Error(w, "Invalid email/password", http.StatusUnauthorized)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(reqAuth.Password))
		if err != nil {
			http.Error(w, "Invalid email/password", http.StatusUnauthorized)
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": user.ID,
			"exp":     time.Now().Add(time.Hour * 24).Unix(),
		})

		tokenString, _ := token.SignedString(config.JWTSecret)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
	}
}

// Me returns the current user ID.
func Me(queries UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		userID := req.Context().Value("userID").(int64)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int64{"user_id": userID})
	}
}
