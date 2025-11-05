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
	// DebugModeEnv is the environment variable for debug mode.
	DebugModeEnv = "DEBUG_MODE"

	// DBHostEnv is the environment variable for database host.
	DBHostEnv = "DB_HOST"

	// DBPortEnv is the environment variable for database port.
	DBPortEnv = "DB_PORT"

	// DBUserEnv is the environment variable for database user.
	DBUserEnv = "DB_USER"

	// DBPassEnv is the environment variable for database password.
	DBPassEnv = "DB_PASS"

	// DBNameEnv is the environment variable for database name.
	DBNameEnv = "DB_NAME"

	// HTTPServerPortEnv is the environment variable for HTTP server port.
	HTTPServerPortEnv = "HTTP_SERVER_PORT"

	// Env is the environment variable for environment name.
	Env = "ENV"

	// MetricsServerPortEnv is the environment variable for metrics server port.
	MetricsServerPortEnv = "METRICS_SERVER_PORT"

	// LocalhostEnv is the constant for localhost.
	LocalhostEnv = "localhost"

	// EnvFilePath is the environment variable for .env file path (only for local/test environment).
	EnvFilePath = "ENV_PATH"

	// DefaultEnvFilePath is the default path to the .env file.
	DefaultEnvFilePath = ".env"

	// AWSRegionEnv is the environment variable for AWS region.
	AWSRegionEnv = "AWS_REGION"

	// AWSEndpointEnv is the environment variable for AWS endpoint.
	AWSEndpointEnv = "AWS_ENDPOINT"

	// SQSQueueURLEnv is the environment variable for SQS queue URL.
	SQSQueueURLEnv = "SQS_QUEUE_URL"
)

var (
	// ErrMissingConfig is returned when required configuration values are missing.
	ErrMissingConfig = errors.New("missing config data")
)

// Config represents the application configuration.
type Config struct {
	DebugMode     bool
	Database      DB
	HTTPServer    Server
	MetricsServer Server
	AWS           AWSConfig
}

// AWSConfig represents AWS-specific configuration settings.
type AWSConfig struct {
	Region      string
	Endpoint    string
	SQSQueueURL string
}

// DB represents database configuration settings.
type DB struct {
	Host     string
	User     string
	Password string
	Name     string
	Port     string
}

// Server represents server configuration settings.
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

	// Validate AWS configuration
	if err := allNonEmpty(map[string]string{
		SQSQueueURLEnv: c.AWS.SQSQueueURL,
	}); err != nil {
		return fmt.Errorf("AWS configuration incomplete: %w", err)
	}

	return nil
}

func getEnvAsBool(name string, defaultValue bool) bool {
	if val, err := strconv.ParseBool(os.Getenv(name)); err == nil {
		return val
	}
	return defaultValue
}

// ApplyEnvFile loads environment variables from the specified .env files.
func ApplyEnvFile(files ...string) error {
	err := godotenv.Load(files...)
	if err != nil {
		return fmt.Errorf("failed to load env file: %w", err)
	}
	return nil
}

// LoadFromEnv loads configuration from environment variables and validates it.
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
		AWS: AWSConfig{
			Region:      os.Getenv(AWSRegionEnv),
			Endpoint:    os.Getenv(AWSEndpointEnv),
			SQSQueueURL: os.Getenv(SQSQueueURLEnv),
		},
	}

	if err := conf.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	return conf, nil
}
