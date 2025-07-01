package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Security SecurityConfig
	File     FileConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	Type string
	Path string
}

type SecurityConfig struct {
	JWTSecret     string
	SessionSecret string
}

type FileConfig struct {
	TempDir    string
	OutputDir  string
	UploadsDir string
	MaxSize    int64
	YTDLPPath  string
}

var AppConfig *Config

func LoadConfig() error {
	AppConfig = &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Host: getEnv("HOST", "localhost"),
		},
		Database: DatabaseConfig{
			Type: getEnv("DB_TYPE", "sqlite"),
			Path: getEnv("DB_PATH", "./database/clipflow.db"),
		},
		Security: SecurityConfig{
			JWTSecret:     getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production"),
			SessionSecret: getEnv("SESSION_SECRET", "your-session-secret-change-this-in-production"),
		},
		File: FileConfig{
			TempDir:    getEnv("TEMP_DIR", "./temp"),
			OutputDir:  getEnv("OUTPUT_DIR", "./output"),
			UploadsDir: getEnv("UPLOADS_DIR", "./uploads"),
			MaxSize:    getEnvAsInt64("MAX_FILE_SIZE", 100*1024*1024), // 100MB default
			YTDLPPath:  getEnv("YTDLP_PATH", "yt-dlp"),                // Default to "yt-dlp" if not specified
		},
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
