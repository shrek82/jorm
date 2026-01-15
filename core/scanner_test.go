package core

import (
	"testing"
	"time"
)

func TestTimeScanner(t *testing.T) {
	ts := &TimeScanner{}

	// Test string
	dateStr := "2026-01-15 16:08:38"
	err := ts.Scan(dateStr)
	if err != nil {
		t.Errorf("Failed to scan string: %v", err)
	}
	if !ts.Valid {
		t.Error("Expected Valid=true")
	}
	// Verify value (Local time)
	expected, _ := time.ParseInLocation("2006-01-02 15:04:05", dateStr, time.Local)
	if !ts.Value.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, ts.Value)
	}

	// Test bytes
	err = ts.Scan([]byte(dateStr))
	if err != nil {
		t.Errorf("Failed to scan bytes: %v", err)
	}
	if !ts.Valid {
		t.Error("Expected Valid=true for bytes")
	}
	if !ts.Value.Equal(expected) {
		t.Errorf("Expected %v, got %v", expected, ts.Value)
	}

	// Test zero date
	err = ts.Scan("0000-00-00 00:00:00")
	if err != nil {
		t.Errorf("Failed to scan zero date: %v", err)
	}
	if ts.Valid {
		t.Error("Expected Valid=false for zero date")
	}

	// Test empty string
	err = ts.Scan("")
	if err != nil {
		t.Errorf("Failed to scan empty string: %v", err)
	}
	if ts.Valid {
		t.Error("Expected Valid=false for empty string")
	}

	// Test time.Time (direct)
	now := time.Now()
	err = ts.Scan(now)
	if err != nil {
		t.Errorf("Failed to scan time.Time: %v", err)
	}
	if !ts.Value.Equal(now) {
		t.Errorf("Expected %v, got %v", now, ts.Value)
	}

	// Test nil
	err = ts.Scan(nil)
	if err != nil {
		t.Errorf("Failed to scan nil: %v", err)
	}
	if ts.Valid {
		t.Error("Expected Valid=false for nil")
	}
}
