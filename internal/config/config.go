package config

import (
	"os"
	"strconv"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `json:"port"`
	Host string `json:"host"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	Charset  string `json:"charset"`
}

// Load 加载配置
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port: getEnvOrDefault("SERVER_PORT", "8080"),
			Host: getEnvOrDefault("SERVER_HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			Host:     getEnvOrDefault("DB_HOST", "127.0.0.1"),
			Port:     getEnvOrDefaultInt("DB_PORT", 3306),
			Username: getEnvOrDefault("DB_USERNAME", "root"),
			Password: getEnvOrDefault("DB_PASSWORD", "root123456"),
			Database: getEnvOrDefault("DB_DATABASE", "orchestrator"),
			Charset:  getEnvOrDefault("DB_CHARSET", "utf8mb4"),
		},
	}
	return config, nil
}

// getEnvOrDefault 获取环境变量或默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvOrDefaultInt 获取环境变量或默认整数值
func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}