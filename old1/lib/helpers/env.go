package helpers

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	_ = godotenv.Load()
}

// GetEnv
// returns string value from env os string or "" empty string
func GetEnv(key string) string {
	return os.Getenv(key)
}

// GetEnvOr
// returns string value from env os string or defaultValue as string
func GetEnvOr(key string, defaultValue string) string {
	value := GetEnv(key)
	if value == "" {
		return defaultValue
	}

	return value
}

// GetEnvAsInt
// returns integer value from env as int64 or 0 (int64)
func GetEnvAsInt(key string) int64 {
	env := GetEnv(key)
	if env == "" {
		return 0
	}

	value, err := strconv.ParseInt(env, 10, 64)
	if err != nil {
		return 0
	}

	return value
}

// GetEnvAsIntOr
// returns integer value from env as int64 or defaultValue as int64
func GetEnvAsIntOr(key string, defaultValue int64) int64 {
	value := GetEnvAsInt(key)
	if value == 0 {
		return defaultValue
	}

	return value
}

// GetEnvAsDuration
// gets value from env by key as possibly signed sequence of
// decimal numbers, each with optional fraction and a unit suffix,
// such as "300ms", "-1.5h" or "2h45m".
// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
// if not found takes defaultValue with same format
// converts it to Duration
// returns Duration
func GetEnvAsDuration(key string, defaultValue string) time.Duration {
	duration, err := time.ParseDuration(GetEnvOr(key, defaultValue))
	if err != nil {
		panic(err.Error())
	}

	return duration
}
