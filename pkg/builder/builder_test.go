package builder

import (
	"os"
	"testing"
)

func TestNewReportBuilder(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gauge_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	builder := NewReportBuilder(tempDir, tempDir)
	if builder == nil {
		t.Error("Expected builder to be created, got nil")
	}
}

func TestReportBuilder_Close(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gauge_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	builder := NewReportBuilder(tempDir, tempDir)
	err = builder.Close()
	if err != nil {
		t.Errorf("Expected no error closing builder, got %v", err)
	}
}
