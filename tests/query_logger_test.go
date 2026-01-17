package tests

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/shrek82/jorm/logger"
)

func TestSQLLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	l := logger.NewStdLogger()
	l.SetOutput(buf)

	// Case 1: LevelInfo should NOT log SQL
	l.SetLevel(logger.LevelInfo)
	l.SQL("SELECT * FROM info", time.Millisecond, nil)
	if buf.Len() > 0 {
		t.Errorf("Expected no output for SQL at LevelInfo, got: %s", buf.String())
	}
	buf.Reset()

	// Case 3: LevelDebug SHOULD log SQL
	l.SetLevel(logger.LevelDebug)
	l.SQL("SELECT * FROM debug", time.Millisecond, nil)
	if !strings.Contains(buf.String(), "SELECT * FROM debug") {
		t.Errorf("Expected output for SQL at LevelDebug, got: %s", buf.String())
	}
}
