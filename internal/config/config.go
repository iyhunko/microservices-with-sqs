package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	DebugModeEnv = "DEBUG_MODE"

	DBHostEnv = "DB_HOST"

	DBPortEnv = "DB_PORT"

	DBUserEnv = "DB_USER"

	DBPassEnv = "DB_PASS"

	DBNameEnv = "DB_NAME"

	HTTPServerPortEnv = "HTTP_SERVER_PORT"

	Env = "ENV"

	MetricsServerPortEnv = "METRICS_SERVER_PORT"

	LocalhostEnv = "localhost"

	EnvFilePath = "ENV_PATH" // only for local/test environment

	DefaultEnvFilePath = ".env"
)

var (
	ErrMissingConfig = errors.New("missing config data")
)

type Config struct {
	DebugMode     bool
	Database      DB
	HTTPServer    Server
	MetricsServer Server
}

type DB struct {
	Host     string
	User     string
	Password string
	Name     string
	Port     string
}

type Server struct {
	Port string
}

func allNonEmpty(keyValues map[string]string) error {
	for key, value := range keyValues {
		if value == "" {
			slog.Error("configuration validation failed", slog.String("key", key), slog.String("error", "value is empty"))
			return fmt.Errorf("%w for key: %s", ErrMissingConfig, key)
		}
	}
	return nil
}

func allNumbers(keyValues map[string]string) error {
	for key, value := range keyValues {
		_, err := strconv.Atoi(value)
		if err != nil {
			slog.Error("configuration validation failed", slog.String("key", key), slog.String("value", value), slog.String("error", err.Error()))
			return fmt.Errorf("invalid number for key %s: %w", key, err)
		}
	}
	return nil
}

func (c *Config) validate() error {
	// Validate database configuration
	if err := allNonEmpty(map[string]string{
		DBHostEnv: c.Database.Host,
		DBUserEnv: c.Database.User,
		DBNameEnv: c.Database.Name,
	}); err != nil {
		return fmt.Errorf("database configuration incomplete: %w", err)
	}

	// Validate server ports
	if err := allNonEmpty(map[string]string{
		HTTPServerPortEnv:    c.HTTPServer.Port,
		MetricsServerPortEnv: c.MetricsServer.Port,
	}); err != nil {
		return fmt.Errorf("server port configuration incomplete: %w", err)
	}

	// Validate port numbers
	if err := allNumbers(map[string]string{
		DBPortEnv:            c.Database.Port,
		HTTPServerPortEnv:    c.HTTPServer.Port,
		MetricsServerPortEnv: c.MetricsServer.Port,
	}); err != nil {
		return fmt.Errorf("invalid port number: %w", err)
	}

	return nil
}

func getEnvAsBool(name string, defaultValue bool) bool {
	if val, err := strconv.ParseBool(os.Getenv(name)); err == nil {
		return val
	}
	return defaultValue
}

func ApplyEnvFile(files ...string) error {
	err := godotenv.Load(files...)
	if err != nil {
		return fmt.Errorf("failed to load env file: %w", err)
	}
	return nil
}

func LoadFromEnv() (*Config, error) {
	envPath := os.Getenv(EnvFilePath)
	if envPath == "" {
		envPath = DefaultEnvFilePath
	}
	err := ApplyEnvFile(envPath)
	if err != nil {
		// just log the error, maybe all envs are set in another way
		slog.Info("failed to load from .env", slog.Any("err", err))
	}

	conf := &Config{
		DebugMode: getEnvAsBool(DebugModeEnv, false),
		Database: DB{
			Host:     os.Getenv(DBHostEnv),
			User:     os.Getenv(DBUserEnv),
			Password: os.Getenv(DBPassEnv),
			Name:     os.Getenv(DBNameEnv),
			Port:     os.Getenv(DBPortEnv),
		},
		HTTPServer: Server{
			Port: os.Getenv(HTTPServerPortEnv),
		},
		MetricsServer: Server{
			Port: os.Getenv(MetricsServerPortEnv),
		},
	}

	if err := conf.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	return conf, nil
}
