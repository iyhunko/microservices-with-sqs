package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
)

// TestInitJSONLogger_OutputFormat verifies that InitJSONLogger sets up
// JSON formatted output for slog
func TestInitJSONLogger_OutputFormat(t *testing.T) {
	// Save original stdout to restore later
	oldStdout := os.Stdout

	// Create a pipe to capture output
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Replace stdout with our write pipe
	os.Stdout = w

	// Initialize the JSON logger
	InitJSONLogger()

	// Log a test message
	slog.Info("test initialization", slog.String("service", "test"), slog.Int("port", 8080))

	// Close the write pipe and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Parse the output as JSON
	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v\nOutput: %s", err, output)
	}

	// Verify expected fields
	if logEntry["msg"] != "test initialization" {
		t.Errorf("Expected msg to be 'test initialization', got '%v'", logEntry["msg"])
	}

	if logEntry["service"] != "test" {
		t.Errorf("Expected service to be 'test', got '%v'", logEntry["service"])
	}

	if logEntry["port"] != float64(8080) {
		t.Errorf("Expected port to be 8080, got '%v'", logEntry["port"])
	}

	if logEntry["level"] != "INFO" {
		t.Errorf("Expected level to be 'INFO', got '%v'", logEntry["level"])
	}

	// Verify time field exists
	if _, ok := logEntry["time"]; !ok {
		t.Error("Expected 'time' field in JSON log output")
	}
}
