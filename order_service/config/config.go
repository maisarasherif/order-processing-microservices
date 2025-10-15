package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Database     DatabaseConfig
	Server       ServerConfig
	Payment      PaymentServiceConfig
	Notification NotificationServiceConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type ServerConfig struct {
	Port     string
	LogLevel string
}

type PaymentServiceConfig struct {
	URL string
}

type NotificationServiceConfig struct {
	URL string
}

func Load() (*Config, error) {

	config := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5431),
			User:     getEnv("DB_USER", "orderuser"),
			Password: getEnv("DB_PASSWORD", "orderpass"),
			DBName:   getEnv("DB_NAME", "order_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Server: ServerConfig{
			Port:     getEnv("SERVICE_PORT", "8081"),
			LogLevel: getEnv("LOG_LEVEL", "info"),
		},
		Payment: PaymentServiceConfig{
			URL: getEnv("PAYMENT_SERVICE_URL", "http://payment_service:8082"),
		},
		Notification: NotificationServiceConfig{
			URL: getEnv("NOTIFICATION_SERVICE_URL", "http://notification_service:8083"),
		},
	}

	if config.Database.Host == "" {

		return nil, fmt.Errorf("DB_HOST is required")
	}
	if config.Database.User == "" {
		return nil, fmt.Errorf("DB_USER is required")
	}
	if config.Database.Password == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}
	if config.Database.DBName == "" {
		return nil, fmt.Errorf("DB_NAME is required")
	}

	return config, nil
}

func (c *Config) GetDatabaseConnectionString() string {
	// fmt.Sprintf is like printf - creates a formatted string
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

func getEnv(key, defaultValue string) string {

	if value := os.Getenv(key); value != "" {
		return value
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
