// Package config загружает конфигурацию приложения из переменных окружения.
package config

import (
	"log"
	"os"
	"strconv"
)

// DBSSLmode определяет режим SSL-подключения к PostgreSQL.
type DBSSLmode string

const (
	// SSLDisable - SSL-шифрование отключено.
	SSLDisable DBSSLmode = "disable"
	// SSLRequire - SSL обязателен, но сертификат сервера не проверяется.
	SSLRequire DBSSLmode = "require"
	// SSLVerifyFull - SSL обязателен, сертификат сервера проверяется.
	SSLVerifyFull DBSSLmode = "verify-full"
)

// ServerConfig - конфигурация HTTP-сервера.
type ServerConfig struct {
	Addr string
}

// LoadServer загружает конфигурацию сервера из окружения.
func LoadServer() ServerConfig {
	return ServerConfig{
		Addr: getEnv("SERVER_ADDR", ":8080"),
	}
}

// IsValid возвращает true, если значение является допустимым режимом SSL.
func (m DBSSLmode) IsValid() bool {
	switch m {
	case SSLDisable, SSLRequire, SSLVerifyFull:
		return true
	default:
		return false
	}
}

// DBConfig - набор параметров для подключения к базе данных.
type DBConfig struct {
	Host     string
	User     string
	Password string
	Name     string
	SSLmode  DBSSLmode
	Port     int
}

// LoadDB загружает конфигурацию бд из окружения и возвращает DBConfig.
func LoadDB() DBConfig {
	port, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		log.Fatalf("invalid DB_PORT %v", err)
	}

	rmode := getEnv("DB_SSLMODE", string(SSLDisable))
	mode := DBSSLmode(rmode)
	if !mode.IsValid() {
		log.Printf("warning: invalid DB_SSLMODE=%q; using default %q", rmode, SSLDisable)
		mode = SSLDisable
	}

	return DBConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     port,
		User:     getEnv("DB_USER", "assigner"),
		Password: getEnv("DB_PASSWORD", "assigner"),
		Name:     getEnv("DB_NAME", "assigner"),
		SSLmode:  mode,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
