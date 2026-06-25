package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	DynamoTableSchedule   string
	DynamoTableEmbedJobs  string
	AWSRegion             string

	CognitoUserPoolID string
	CognitoClientID   string

	RecognitionAPIURL string
	RecognitionAPIKey string

	BedrockModelID    string
	EmbeddingModelID  string

	S3VectorsBucket string

	ImageDir  string
	ExportDir string

	ExternalTimeoutSec int

	ConfidenceAutoApprove float64
	ConfidenceMinAccept   float64
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:                  getEnv("PORT", "8080"),
		DBHost:                mustEnv("DB_HOST"),
		DBPort:                getEnv("DB_PORT", "3306"),
		DBUser:                mustEnv("DB_USER"),
		DBPassword:            mustEnv("DB_PASSWORD"),
		DBName:                mustEnv("DB_NAME"),
		DynamoTableSchedule:   mustEnv("DYNAMO_TABLE_SCHEDULE"),
		DynamoTableEmbedJobs:  mustEnv("DYNAMO_TABLE_EMBED_JOBS"),
		AWSRegion:             mustEnv("AWS_REGION"),
		CognitoUserPoolID:     mustEnv("COGNITO_USER_POOL_ID"),
		CognitoClientID:       mustEnv("COGNITO_CLIENT_ID"),
		RecognitionAPIURL:     getEnv("RECOGNITION_API_URL", ""),
		RecognitionAPIKey:     getEnv("RECOGNITION_API_KEY", ""),
		BedrockModelID:        getEnv("BEDROCK_MODEL_ID", "claude-sonnet-4-6"),
		EmbeddingModelID:      getEnv("EMBEDDING_MODEL_ID", "amazon.titan-embed-text-v2:0"),
		S3VectorsBucket:       getEnv("S3_VECTORS_BUCKET", ""),
		ImageDir:              getEnv("IMAGE_DIR", "/data/imgs"),
		ExportDir:             getEnv("EXPORT_DIR", "/data/exports"),
		ExternalTimeoutSec:    getEnvInt("EXTERNAL_TIMEOUT_SEC", 30),
		ConfidenceAutoApprove: 0.85,
		ConfidenceMinAccept:   0.50,
	}
	return cfg, nil
}

func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC&timeout=10s&readTimeout=30s&writeTimeout=30s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
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

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
