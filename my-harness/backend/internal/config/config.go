package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	DynamoTableSchedule string
	AWSRegion           string

	CognitoUserPoolID string
	CognitoClientID   string

	RecognitionAPIURL string
	RecognitionAPIKey string

	BedrockModelID string

	ImageDir string

	ExternalTimeoutSec int
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		DBHost:              mustEnv("DB_HOST"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              mustEnv("DB_USER"),
		DBPassword:          mustEnv("DB_PASSWORD"),
		DBName:              mustEnv("DB_NAME"),
		DynamoTableSchedule: mustEnv("DYNAMO_TABLE_SCHEDULE"),
		AWSRegion:           mustEnv("AWS_REGION"),
		CognitoUserPoolID:   mustEnv("COGNITO_USER_POOL_ID"),
		CognitoClientID:     mustEnv("COGNITO_CLIENT_ID"),
		RecognitionAPIURL:   mustEnv("RECOGNITION_API_URL"),
		RecognitionAPIKey:   mustEnv("RECOGNITION_API_KEY"),
		BedrockModelID:      "claude-sonnet-4-6",
		ImageDir:            getEnv("IMAGE_DIR", "/data/imgs"),
		ExternalTimeoutSec:  15,
	}
	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName)
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required env var %s is not set", key))
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
