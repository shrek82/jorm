package tests

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/shrek82/jorm/logger"
)

func TestStructuredLogger(t *testing.T) {
	t.Run("TextFormat", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := logger.NewStdLogger()
		l.SetLevel(logger.LogLevelInfo)
		l.SetOutput(buf)
		l.SetFormat(logger.LogFormatText)
		l.Info("hello %s", "world")

		output := buf.String()
		if !strings.Contains(output, "INFO") || !strings.Contains(output, "hello world") {
			t.Errorf("Unexpected text output: %s", output)
		}
	})

	t.Run("JSONFormat", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := logger.NewStdLogger()
		l.SetLevel(logger.LogLevelInfo)
		l.SetOutput(buf)
		l.SetFormat(logger.LogFormatJSON)
		l.Info("hello %s", "world")

		var data map[string]any
		if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
			t.Fatalf("Failed to unmarshal JSON output: %v", err)
		}

		if data["level"] != "INFO" || data["msg"] != "hello world" {
			t.Errorf("Unexpected JSON output: %v", data)
		}
		if _, ok := data["time"]; !ok {
			t.Errorf("Missing time field in JSON output")
		}
	})

	t.Run("WithFields", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := logger.NewStdLogger()
		l.SetLevel(logger.LogLevelInfo)
		l.SetOutput(buf)
		l.SetFormat(logger.LogFormatJSON)
		l2 := l.WithFields(map[string]any{"request_id": "123"})
		l2.Info("processed")

		var data map[string]any
		if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
			t.Fatalf("Failed to unmarshal JSON output: %v", err)
		}

		if data["request_id"] != "123" || data["msg"] != "processed" {
			t.Errorf("Unexpected JSON output with fields: %v", data)
		}
	})

	t.Run("SQLJSON", func(t *testing.T) {
		buf := &bytes.Buffer{}
		l := logger.NewStdLogger()
		l.SetLevel(logger.LogLevelInfo)
		l.SetOutput(buf)
		l.SetFormat(logger.LogFormatJSON)
		l.SQL("SELECT * FROM users", time.Millisecond*10, "arg1", 1)

		var data map[string]any
		if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
			t.Fatalf("Failed to unmarshal JSON output: %v", err)
		}

		if data["level"] != "SQL" || data["sql"] != "SELECT * FROM users" {
			t.Errorf("Unexpected SQL JSON output: %v", data)
		}
		if data["duration"] == "" {
			t.Errorf("Missing duration in SQL JSON output")
		}
	})

	t.Run("LevelOutput", func(t *testing.T) {
		mainBuf := &bytes.Buffer{}
		errorBuf := &bytes.Buffer{}
		l := logger.NewStdLogger()
		l.SetLevel(logger.LogLevelInfo)
		l.SetOutput(mainBuf)
		l.SetLevelOutput(logger.LogLevelError, errorBuf)

		l.Info("this is info")
		l.Error("this is error")

		mainOutput := mainBuf.String()
		errorOutput := errorBuf.String()

		if !strings.Contains(mainOutput, "INFO") || !strings.Contains(mainOutput, "this is info") {
			t.Errorf("Main buffer missing INFO: %s", mainOutput)
		}
		if !strings.Contains(mainOutput, "ERROR") || !strings.Contains(mainOutput, "this is error") {
			t.Errorf("Main buffer missing ERROR: %s", mainOutput)
		}

		if strings.Contains(errorOutput, "INFO") {
			t.Errorf("Error buffer should not contain INFO: %s", errorOutput)
		}
		if !strings.Contains(errorOutput, "ERROR") || !strings.Contains(errorOutput, "this is error") {
			t.Errorf("Error buffer missing ERROR: %s", errorOutput)
		}
	})

	t.Run("LevelOutputOnly", func(t *testing.T) {
		errorBuf := &bytes.Buffer{}
		l := logger.NewStdLogger()
		l.SetLevel(logger.LogLevelInfo)
		l.SetOutput(nil) // Disable default output
		l.SetLevelOutput(logger.LogLevelError, errorBuf)

		l.Info("this is info")
		l.Error("this is error")

		errorOutput := errorBuf.String()

		if strings.Contains(errorOutput, "INFO") {
			t.Errorf("Error buffer should not contain INFO: %s", errorOutput)
		}
		if !strings.Contains(errorOutput, "ERROR") || !strings.Contains(errorOutput, "this is error") {
			t.Errorf("Error buffer missing ERROR: %s", errorOutput)
		}
	})
}
