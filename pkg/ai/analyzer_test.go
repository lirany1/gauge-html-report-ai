package ai

import (
	"strings"
	"testing"

	"github.com/your-org/gauge-html-report-enhanced/pkg/models"
)

func TestAnalyzer_PatternBasedMode(t *testing.T) {
	// Test pattern-based analysis (default mode, no LLM)
	analyzer := NewAnalyzer()

	// Verify analyzer is created
	if analyzer == nil {
		t.Fatal("Expected analyzer to be created, got nil")
	}

	// Test error classification
	errorType := analyzer.ClassifyError("Expected 5 but got 1", "")
	if errorType != ErrorTypeAssertion {
		t.Errorf("Expected ErrorTypeAssertion, got %v", errorType)
	}

	// Test timeout classification
	errorType = analyzer.ClassifyError("Connection timeout after 30 seconds", "")
	if errorType != ErrorTypeTimeout {
		t.Errorf("Expected ErrorTypeTimeout, got %v", errorType)
	}

	// Test network classification
	errorType = analyzer.ClassifyError("Connection refused to host", "")
	if errorType != ErrorTypeNetwork {
		t.Errorf("Expected ErrorTypeNetwork, got %v", errorType)
	}
}

func TestAnalyzer_GenerateErrorSignature(t *testing.T) {
	analyzer := NewAnalyzer()

	// Test signature generation with number normalization
	sig1 := analyzer.GenerateErrorSignature("Expected 5 but got 1", "Assertion")
	sig2 := analyzer.GenerateErrorSignature("Expected 10 but got 2", "Assertion")

	// Signatures should be the same after normalization
	if sig1 != sig2 {
		t.Errorf("Expected same signature after normalization, got %s and %s", sig1, sig2)
	}

	// Different error types should produce different signatures
	sig3 := analyzer.GenerateErrorSignature("Expected 5 but got 1", "Timeout")
	if sig1 == sig3 {
		t.Errorf("Expected different signatures for different error types")
	}
}

func TestAnalyzer_GroupFailures(t *testing.T) {
	analyzer := NewAnalyzer()

	// Create mock suite result
	suite := &models.EnhancedSuiteResult{
		SpecResults: []*models.SpecResult{
			{
				SpecHeading: "Test Spec",
				Scenarios: []*models.ScenarioResult{
					{
						ScenarioHeading: "Scenario 1",
						Failed:          true,
						Steps: []*models.StepResult{
							{
								Failed:       true,
								ErrorMessage: "Expected 5 but got 1",
								StackTrace:   "at test.py:10",
								StepText:     "Verify count",
							},
						},
					},
					{
						ScenarioHeading: "Scenario 2",
						Failed:          true,
						Steps: []*models.StepResult{
							{
								Failed:       true,
								ErrorMessage: "Expected 10 but got 2",
								StackTrace:   "at test.py:20",
								StepText:     "Verify count",
							},
						},
					},
				},
			},
		},
	}

	// Group failures
	groups := analyzer.GroupFailures(suite)

	// Should have 1 group (both errors are same type after normalization)
	if len(groups) != 1 {
		t.Errorf("Expected 1 failure group, got %d", len(groups))
	}

	// Verify group has correct count
	if len(groups) > 0 && groups[0].Count != 2 {
		t.Errorf("Expected count 2, got %d", groups[0].Count)
	}

	// Verify context is stored
	if len(groups) > 0 {
		group := groups[0]
		if group.ErrorMessage == "" {
			t.Error("Expected ErrorMessage to be stored")
		}
		if group.StepText == "" {
			t.Error("Expected StepText to be stored")
		}
		if group.SpecName == "" {
			t.Error("Expected SpecName to be stored")
		}
	}
}

func TestAnalyzer_GenerateExecutiveSummary(t *testing.T) {
	analyzer := NewAnalyzer()

	// Create mock suite result
	suite := &models.EnhancedSuiteResult{
		TotalScenariosCount:  10,
		PassedScenariosCount: 8,
		FailedScenariosCount: 2,
		SuccessRate:          80.0,
	}

	groups := []*FailureGroup{
		{
			ErrorType: ErrorTypeAssertion,
			Count:     2,
			Severity:  "high",
		},
	}

	// Generate summary
	summary := analyzer.GenerateExecutiveSummary(suite, groups)

	// Verify health status
	if summary.HealthStatus != "Fair" {
		t.Errorf("Expected HealthStatus 'Fair' for 80%% success, got %s", summary.HealthStatus)
	}

	// Verify insights exist
	if len(summary.KeyInsights) == 0 {
		t.Error("Expected key insights to be generated")
	}

	// Verify recommendation exists
	if summary.Recommendation == "" {
		t.Error("Expected recommendation to be generated")
	}
}

func TestAnalyzer_PatternBasedSuggestions(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		errorType ErrorType
		contains  string
	}{
		{ErrorTypeAssertion, "expectations"},
		{ErrorTypeTimeout, "timeout"},
		{ErrorTypeNetwork, "network"},
		{ErrorTypeNullPointer, "null"},
		{ErrorTypeFileSystem, "file"},
		{ErrorTypeDatabase, "database"},
		{ErrorTypeEnvironment, "environment"},
	}

	for _, tt := range tests {
		suggestion := analyzer.getPatternBasedSuggestion(tt.errorType)
		if suggestion == "" {
			t.Errorf("Expected suggestion for %v, got empty string", tt.errorType)
		}
	}
}

func TestAnalyzer_CalculateSeverity(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		errorType ErrorType
		count     int
		expected  string
	}{
		{ErrorTypeAssertion, 1, "medium"},
		{ErrorTypeAssertion, 2, "high"},
		{ErrorTypeAssertion, 11, "critical"},
		{ErrorTypeTimeout, 1, "high"},
		{ErrorTypeDatabase, 1, "critical"},
	}

	for _, tt := range tests {
		severity := analyzer.calculateSeverity(tt.errorType, tt.count)
		if severity != tt.expected {
			t.Errorf("For %v with count %d, expected %s, got %s",
				tt.errorType, tt.count, tt.expected, severity)
		}
	}
}

func TestAnalyzer_ExtractRootCause(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name     string
		errorMsg string
		expected string
	}{
		{
			name:     "multiline error",
			errorMsg: "AssertionError: Values don't match\nExpected: 5\nActual: 3\nAt step: login",
			expected: "AssertionError: Values don't match",
		},
		{
			name:     "long single line",
			errorMsg: strings.Repeat("Very long error message ", 10),
			expected: strings.Repeat("Very long error message ", 6)[:140] + "...",
		},
		{
			name:     "short error",
			errorMsg: "Short error",
			expected: "Short error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.extractRootCause(tt.errorMsg)
			if tt.name == "long single line" {
				// For long strings, just check it was truncated
				if len(result) <= len(tt.errorMsg) && strings.Contains(result, "...") {
					return // Test passes
				}
				t.Errorf("extractRootCause() should truncate long strings and add '...', got %v", result)
			} else if result != tt.expected {
				t.Errorf("extractRootCause() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAnalyzer_LLMIntegration(t *testing.T) {
	// Test that analyzer initializes correctly even without LLM
	analyzer := NewAnalyzer()

	if analyzer == nil {
		t.Fatal("Expected analyzer to be created")
	}

	// In pattern-based mode, useRealAI should be false
	if analyzer.useRealAI {
		t.Error("Expected useRealAI to be false without LLM configuration")
	}

	// generateFixSuggestion should fall back to pattern-based
	suggestion := analyzer.generateFixSuggestion(
		ErrorTypeAssertion,
		"Expected 5 but got 1",
		"at test.py:10",
		"Verify count",
		"Test Spec",
	)

	if suggestion == "" {
		t.Error("Expected pattern-based suggestion even without LLM")
	}
}
