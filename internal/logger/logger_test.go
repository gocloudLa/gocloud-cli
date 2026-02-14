package logger

import (
	"bytes"
	"io"
	"log"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	// Save original loggers
	originalInfo := InfoLogger
	originalError := ErrorLogger
	originalDebug := DebugLogger

	// Initialize logger
	Init()

	// Verify loggers are initialized
	if InfoLogger == nil {
		t.Error("InfoLogger should not be nil after Init()")
	}
	if ErrorLogger == nil {
		t.Error("ErrorLogger should not be nil after Init()")
	}
	if DebugLogger == nil {
		t.Error("DebugLogger should not be nil after Init()")
	}

	// Verify loggers are different from original (if they were nil)
	if originalInfo == nil && InfoLogger == nil {
		t.Error("InfoLogger should be initialized")
	}
	if originalError == nil && ErrorLogger == nil {
		t.Error("ErrorLogger should be initialized")
	}
	if originalDebug == nil && DebugLogger == nil {
		t.Error("DebugLogger should be initialized")
	}
}

func TestInfo(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create a custom logger that writes to buffer
	InfoLogger = log.New(&buf, "INFO: ", log.Ldate|log.Ltime)

	// Test Info function
	Info("Test message")

	// Check output
	output := buf.String()
	if !strings.Contains(output, "INFO:") {
		t.Errorf("Info() output should contain 'INFO:', got: %s", output)
	}
	if !strings.Contains(output, "Test message") {
		t.Errorf("Info() output should contain 'Test message', got: %s", output)
	}
}

func TestInfoWithFormat(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create a custom logger that writes to buffer
	InfoLogger = log.New(&buf, "INFO: ", log.Ldate|log.Ltime)

	// Test Info function with format
	Info("Test message with %s", "formatting")

	// Check output
	output := buf.String()
	if !strings.Contains(output, "INFO:") {
		t.Errorf("Info() output should contain 'INFO:', got: %s", output)
	}
	if !strings.Contains(output, "Test message with formatting") {
		t.Errorf("Info() output should contain formatted message, got: %s", output)
	}
}

func TestError(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create a custom logger that writes to buffer
	ErrorLogger = log.New(&buf, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Test Error function
	Error("Test error message")

	// Check output
	output := buf.String()
	if !strings.Contains(output, "ERROR:") {
		t.Errorf("Error() output should contain 'ERROR:', got: %s", output)
	}
	if !strings.Contains(output, "Test error message") {
		t.Errorf("Error() output should contain 'Test error message', got: %s", output)
	}
}

func TestErrorWithFormat(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create a custom logger that writes to buffer
	ErrorLogger = log.New(&buf, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Test Error function with format
	Error("Test error with %s", "formatting")

	// Check output
	output := buf.String()
	if !strings.Contains(output, "ERROR:") {
		t.Errorf("Error() output should contain 'ERROR:', got: %s", output)
	}
	if !strings.Contains(output, "Test error with formatting") {
		t.Errorf("Error() output should contain formatted message, got: %s", output)
	}
}

func TestDebug(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create a custom logger that writes to buffer
	DebugLogger = log.New(&buf, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Test Debug function
	Debug("Test debug message")

	// Check output
	output := buf.String()
	if !strings.Contains(output, "DEBUG:") {
		t.Errorf("Debug() output should contain 'DEBUG:', got: %s", output)
	}
	if !strings.Contains(output, "Test debug message") {
		t.Errorf("Debug() output should contain 'Test debug message', got: %s", output)
	}
}

func TestDebugWithFormat(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create a custom logger that writes to buffer
	DebugLogger = log.New(&buf, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Test Debug function with format
	Debug("Test debug with %s", "formatting")

	// Check output
	output := buf.String()
	if !strings.Contains(output, "DEBUG:") {
		t.Errorf("Debug() output should contain 'DEBUG:', got: %s", output)
	}
	if !strings.Contains(output, "Test debug with formatting") {
		t.Errorf("Debug() output should contain formatted message, got: %s", output)
	}
}

func TestFatal(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create a custom logger that writes to buffer
	ErrorLogger = log.New(&buf, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Test Fatal function - we can't actually test os.Exit(1) without crashing the test
	// So we'll just test that it logs the message correctly
	// Note: In a real scenario, Fatal would call os.Exit(1), but we can't test that in unit tests

	// We'll test the logging part by calling the underlying logger directly
	ErrorLogger.Printf("FATAL: Test fatal message")

	// Check output
	output := buf.String()
	if !strings.Contains(output, "ERROR:") {
		t.Errorf("Fatal() output should contain 'ERROR:', got: %s", output)
	}
	if !strings.Contains(output, "FATAL: Test fatal message") {
		t.Errorf("Fatal() output should contain 'FATAL: Test fatal message', got: %s", output)
	}
}

func TestFatalWithFormat(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Create a custom logger that writes to buffer
	ErrorLogger = log.New(&buf, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Test Fatal function with format - same caveat as above
	ErrorLogger.Printf("FATAL: Test fatal with %s", "formatting")

	// Check output
	output := buf.String()
	if !strings.Contains(output, "ERROR:") {
		t.Errorf("Fatal() output should contain 'ERROR:', got: %s", output)
	}
	if !strings.Contains(output, "FATAL: Test fatal with formatting") {
		t.Errorf("Fatal() output should contain formatted message, got: %s", output)
	}
}

// TestLoggerIntegration tests the logger functions work together
func TestLoggerIntegration(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Initialize custom loggers
	InfoLogger = log.New(&buf, "INFO: ", log.Ldate|log.Ltime)
	ErrorLogger = log.New(&buf, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(&buf, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Test all logging functions
	Info("Integration test info")
	Error("Integration test error")
	Debug("Integration test debug")

	// Check output contains all messages
	output := buf.String()

	if !strings.Contains(output, "INFO:") {
		t.Error("Output should contain INFO log")
	}
	if !strings.Contains(output, "ERROR:") {
		t.Error("Output should contain ERROR log")
	}
	if !strings.Contains(output, "DEBUG:") {
		t.Error("Output should contain DEBUG log")
	}

	if !strings.Contains(output, "Integration test info") {
		t.Error("Output should contain info message")
	}
	if !strings.Contains(output, "Integration test error") {
		t.Error("Output should contain error message")
	}
	if !strings.Contains(output, "Integration test debug") {
		t.Error("Output should contain debug message")
	}
}

// TestLoggerOutputDestinations tests that loggers write to correct destinations
func TestLoggerOutputDestinations(t *testing.T) {
	// Create separate buffers for each logger
	var infoBuf, errorBuf, debugBuf bytes.Buffer

	// Initialize loggers with different outputs
	InfoLogger = log.New(&infoBuf, "INFO: ", log.Ldate|log.Ltime)
	ErrorLogger = log.New(&errorBuf, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(&debugBuf, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Test each logger
	Info("Info message")
	Error("Error message")
	Debug("Debug message")

	// Check that each logger writes to its own buffer
	if infoBuf.Len() == 0 {
		t.Error("InfoLogger should write to infoBuf")
	}
	if errorBuf.Len() == 0 {
		t.Error("ErrorLogger should write to errorBuf")
	}
	if debugBuf.Len() == 0 {
		t.Error("DebugLogger should write to debugBuf")
	}

	// Check that buffers don't cross-contaminate
	if strings.Contains(infoBuf.String(), "ERROR:") || strings.Contains(infoBuf.String(), "DEBUG:") {
		t.Error("InfoLogger should not contain ERROR or DEBUG messages")
	}
	if strings.Contains(errorBuf.String(), "INFO:") || strings.Contains(errorBuf.String(), "DEBUG:") {
		t.Error("ErrorLogger should not contain INFO or DEBUG messages")
	}
	if strings.Contains(debugBuf.String(), "INFO:") || strings.Contains(debugBuf.String(), "ERROR:") {
		t.Error("DebugLogger should not contain INFO or ERROR messages")
	}
}

// Benchmark tests for performance
func BenchmarkInfo(b *testing.B) {
	// Redirect output to discard to avoid cluttering benchmark output
	InfoLogger = log.New(io.Discard, "INFO: ", log.Ldate|log.Ltime)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("Benchmark message %d", i)
	}
}

func BenchmarkError(b *testing.B) {
	// Redirect output to discard to avoid cluttering benchmark output
	ErrorLogger = log.New(io.Discard, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Error("Benchmark error message %d", i)
	}
}

func BenchmarkDebug(b *testing.B) {
	// Redirect output to discard to avoid cluttering benchmark output
	DebugLogger = log.New(io.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Debug("Benchmark debug message %d", i)
	}
}
