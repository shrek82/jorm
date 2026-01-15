package tests

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shrek82/jorm/core"
	"github.com/shrek82/jorm/logger"
)

func TestQueryStructuredLogging(t *testing.T) {
	db, err := core.Open("sqlite3", ":memory:", nil)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	buf := &bytes.Buffer{}
	l := logger.NewStdLogger()
	l.SetLevel(logger.LogLevelInfo)
	l.SetOutput(buf)
	l.SetFormat(logger.LogFormatJSON)
	db.SetLogger(l)

	type LogUser struct {
		ID   int64 `jorm:"pk auto"`
		Name string
	}
	db.AutoMigrate(&LogUser{})

	// Test with custom fields for a specific query
	db.Model(&LogUser{}).WithFields(map[string]any{"trace_id": "trace-123"}).Insert(&LogUser{Name: "Alice"})

	output := buf.String()
	// The output might contain multiple JSON objects (AutoMigrate also logs)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	found := false
	for _, line := range lines {
		var data map[string]any
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			continue
		}
		if data["trace_id"] == "trace-123" && data["level"] == "SQL" {
			found = true
			if !strings.Contains(data["sql"].(string), "INSERT") {
				t.Errorf("Expected INSERT SQL, got: %v", data["sql"])
			}
			break
		}
	}

	if !found {
		t.Errorf("Structured log with trace_id not found in output: %s", output)
	}
}
