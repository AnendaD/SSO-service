package models

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

type App struct {
	ID     int
	Name   string
	Secret string
}

func GenerateAppSecret() (string, error) {
	const op = "models.app.GenerateAppSecret"

	const secretLength = 32

	bytes := make([]byte, secretLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("%s: %w ", op, err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil

}
