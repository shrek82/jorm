package tests

import (
	"bytes"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/shrek82/jorm/core"
	"github.com/shrek82/jorm/logger"
)

func TestSQLErrorLogging(t *testing.T) {
	// Setup DB
	db, err := core.Open("sqlite3", ":memory:", nil)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer db.Close()

	// Setup Logger capturing output
	buf := &bytes.Buffer{}
	l := logger.NewStdLogger()
	l.SetOutput(buf)
	l.SetLevel(logger.LevelError) // Only log errors
	db.SetLogger(l)

	// Execute invalid SQL
	_, err = db.Exec("SELECT * FROM non_existent_table")
	if err == nil {
		t.Fatal("Expected error from invalid SQL, got nil")
	}

	// Check logs
	output := buf.String()
	if !strings.Contains(output, "[JORM-ERROR]") {
		t.Errorf("Expected [JORM-ERROR] in logs, got: %s", output)
	}
	if !strings.Contains(output, "no such table: non_existent_table") {
		t.Errorf("Expected error message in logs, got: %s", output)
	}
	// Verify exact format with SQL and repeated error
	// Note: The error message includes "raw sql execution failed: " prefix because it's wrapped in query.go
	expectedPart := "| SQL: SELECT * FROM non_existent_table | Args: [] |  SQL execution error: raw sql execution failed: no such table: non_existent_table"
	if !strings.Contains(output, expectedPart) {
		t.Errorf("Expected log format containing '%s', got: %s", expectedPart, output)
	}
}
