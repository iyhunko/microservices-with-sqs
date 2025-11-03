package config_test

import (
	"testing"

	"github.com/iyhunko/microservices-with-sqs/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromEnv(t *testing.T) {
	t.Setenv(config.DebugModeEnv, "true")
	t.Setenv(config.DBHostEnv, "localhost")
	t.Setenv(config.DBUserEnv, "user")
	t.Setenv(config.DBPassEnv, "pass")
	t.Setenv(config.DBNameEnv, "testdb")
	t.Setenv(config.DBPortEnv, "5432")
	t.Setenv(config.HTTPServerPortEnv, "8080")
	t.Setenv(config.MetricsServerPortEnv, "9090")

	conf, err := config.LoadFromEnv()
	require.NoError(t, err, "loading config should not return error")

	assert.True(t, conf.DebugMode, "DebugMode should be true")
	assert.Equal(t, "localhost", conf.Database.Host, "DB Host should be 'localhost'")
	assert.Equal(t, "user", conf.Database.User, "DB User should be 'user'")
	assert.Equal(t, "pass", conf.Database.Password, "DB Password should be 'pass'")
	assert.Equal(t, "testdb", conf.Database.Name, "DB Name should be 'testdb'")
	assert.Equal(t, "5432", conf.Database.Port, "DB Port should be '5432'")
	assert.Equal(t, "8080", conf.HTTPServer.Port, "HTTP Server Port should be '8080'")
	assert.Equal(t, "9090", conf.MetricsServer.Port, "Metrics Server Port should be '9090'")
}

func TestGetEnvAsBool(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		want         bool
	}{
		{"GetEnvAsBool_True", "true", false, true},
		{"GetEnvAsBool_False", "false", true, false},
		{"GetEnvAsBool_Invalid", "invalid", true, true},
		{"GetEnvAsBool_Empty", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ENV", tt.envValue)
			got := config.GetEnvAsBool("TEST_ENV", tt.defaultValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAllNumbers(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]string
		wantErr bool
	}{
		{"AllNumbers_Valid", map[string]string{"key1": "123", "key2": "456", "key3": "789"}, false},
		{"AllNumbers_Invalid", map[string]string{"key1": "123", "key2": "abc", "key3": "789"}, true},
		{"AllNumbers_EmptyString", map[string]string{"key1": "123", "key2": "", "key3": "789"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.AllNumbers(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAllNonEmpty(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]string
		wantErr bool
	}{
		{"AllNonEmpty_Valid", map[string]string{"key1": "host", "key2": "user", "key3": "pass"}, false},
		{"AllNonEmpty_EmptyString", map[string]string{"key1": "host", "key2": "", "key3": "pass"}, true},
		{"AllNonEmpty_AllEmpty", map[string]string{"key1": "", "key2": "", "key3": ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.AllNonEmpty(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
