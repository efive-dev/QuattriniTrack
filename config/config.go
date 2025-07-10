// Package config contains configurations for the environment
package config

import (
	"log"
	"os"

	"github.com/lpernett/godotenv"
)

var JWTSecret []byte

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file %v", err)
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatalf("JWT_SECRET not set in environment file")
	}

	JWTSecret = []byte(secret)
}
