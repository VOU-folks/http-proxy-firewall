package utils

import (
	"os"

	"github.com/joho/godotenv"
)

func init() {
	_ = godotenv.Load()
}

func GetEnv(key string) string {
	return os.Getenv(key)
}
