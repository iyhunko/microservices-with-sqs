package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestInitJSONLogger(t *testing.T) {
	// Capture the output
	var buf bytes.Buffer

	// Create a JSON handler that writes to our buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))

	// Log a test message
	slog.Info("test message", slog.String("key", "value"), slog.Int("number", 42))

	// Verify the output is valid JSON
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v\nOutput: %s", err, buf.String())
	}

	// Verify the log entry contains expected fields
	if logEntry["msg"] != "test message" {
		t.Errorf("Expected msg to be 'test message', got '%v'", logEntry["msg"])
	}

	if logEntry["key"] != "value" {
		t.Errorf("Expected key to be 'value', got '%v'", logEntry["key"])
	}

	if logEntry["number"] != float64(42) {
		t.Errorf("Expected number to be 42, got '%v'", logEntry["number"])
	}

	if logEntry["level"] != "INFO" {
		t.Errorf("Expected level to be 'INFO', got '%v'", logEntry["level"])
	}

	// Verify time field exists
	if _, ok := logEntry["time"]; !ok {
		t.Error("Expected 'time' field in JSON log output")
	}
}
