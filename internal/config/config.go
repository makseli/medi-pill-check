package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBType       string
	SQLitePath   string
	PostgresHost string
	PostgresPort string
	PostgresUser string
	PostgresPass string
	PostgresDB   string
	RedisHost    string
	RedisPort    string
	RedisPass    string
	RedisDB      string
	JWTSecret    string
	Port         string
	Environment  string
	CORSHost     string
}

func Load() *Config {
	_ = godotenv.Load() // .env dosyasını yükle

	return &Config{
		DBType:       getEnv("DB_TYPE", "sqlite"),
		SQLitePath:   getEnv("SQLITE_PATH", "data/sqlite/sqlite.db"),
		PostgresHost: getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort: getEnv("POSTGRES_PORT", "5432"),
		PostgresUser: getEnv("POSTGRES_USER", "medipillcheck"),
		PostgresPass: getEnv("POSTGRES_PASSWORD", "medipillcheck123"),
		PostgresDB:   getEnv("POSTGRES_DB", "medipillcheck"),
		RedisHost:    getEnv("REDIS_HOST", "localhost"),
		RedisPort:    getEnv("REDIS_PORT", "6379"),
		RedisPass:    getEnv("REDIS_PASSWORD", ""),
		RedisDB:      getEnv("REDIS_DB", "0"),
		JWTSecret:    getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production"),
		Port:         getEnv("PORT", "8080"),
		Environment:  getEnv("ENVIRONMENT", "development"),
		CORSHost:     getEnv("CORS_HOST", "*"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
