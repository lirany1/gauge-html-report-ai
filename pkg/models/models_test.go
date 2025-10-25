package models

import (
	"testing"
	"time"
)

func TestSpecResult_GetHTMLFileName(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		expected string
	}{
		{
			name:     "spec file with .spec extension",
			fileName: "/path/to/example.spec",
			expected: "example.html",
		},
		{
			name:     "spec file without extension",
			fileName: "/path/to/example",
			expected: "example.html",
		},
		{
			name:     "spec file with multiple dots",
			fileName: "/path/to/example.test.spec",
			expected: "example.test.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &SpecResult{FileName: tt.fileName}
			result := spec.GetHTMLFileName()
			if result != tt.expected {
				t.Errorf("GetHTMLFileName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSpecResult_GetStatus(t *testing.T) {
	tests := []struct {
		name     string
		failed   bool
		skipped  bool
		expected string
	}{
		{
			name:     "passed spec",
			failed:   false,
			skipped:  false,
			expected: "passed",
		},
		{
			name:     "failed spec",
			failed:   true,
			skipped:  false,
			expected: "failed",
		},
		{
			name:     "skipped spec",
			failed:   false,
			skipped:  true,
			expected: "skipped",
		},
		{
			name:     "failed and skipped spec (failed takes precedence)",
			failed:   true,
			skipped:  true,
			expected: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &SpecResult{
				Failed:  tt.failed,
				Skipped: tt.skipped,
			}
			result := spec.GetStatus()
			if result != tt.expected {
				t.Errorf("GetStatus() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSpecResult_GetFailedScenariosCount(t *testing.T) {
	spec := &SpecResult{
		Scenarios: []*ScenarioResult{
			{Failed: true},
			{Failed: false},
			{Failed: true},
			{Failed: false, Skipped: true},
		},
	}

	expected := 2
	result := spec.GetFailedScenariosCount()
	if result != expected {
		t.Errorf("GetFailedScenariosCount() = %v, want %v", result, expected)
	}
}

func TestSpecResult_GetPassedScenariosCount(t *testing.T) {
	spec := &SpecResult{
		Scenarios: []*ScenarioResult{
			{Failed: true},
			{Failed: false, Skipped: false},
			{Failed: true},
			{Failed: false, Skipped: true},
			{Failed: false, Skipped: false},
		},
	}

	expected := 2
	result := spec.GetPassedScenariosCount()
	if result != expected {
		t.Errorf("GetPassedScenariosCount() = %v, want %v", result, expected)
	}
}

func TestSpecResult_GetSkippedScenariosCount(t *testing.T) {
	spec := &SpecResult{
		Scenarios: []*ScenarioResult{
			{Failed: true},
			{Failed: false, Skipped: false},
			{Failed: false, Skipped: true},
			{Failed: false, Skipped: true},
		},
	}

	expected := 2
	result := spec.GetSkippedScenariosCount()
	if result != expected {
		t.Errorf("GetSkippedScenariosCount() = %v, want %v", result, expected)
	}
}

func TestEnhancedSuiteResult_Creation(t *testing.T) {
	suite := &EnhancedSuiteResult{
		ProjectName:           "Test Project",
		Environment:           "development",
		Tags:                  []string{"smoke", "regression"},
		ExecutionTime:         5 * time.Minute,
		Timestamp:             time.Now(),
		SuccessRate:           85.5,
		PassedSpecsCount:      17,
		FailedSpecsCount:      3,
		SkippedSpecsCount:     0,
		TotalSpecsCount:       20,
		PassedScenariosCount:  45,
		FailedScenariosCount:  8,
		SkippedScenariosCount: 2,
		TotalScenariosCount:   55,
	}

	if suite.ProjectName != "Test Project" {
		t.Errorf("ProjectName = %v, want %v", suite.ProjectName, "Test Project")
	}

	if suite.SuccessRate != 85.5 {
		t.Errorf("SuccessRate = %v, want %v", suite.SuccessRate, 85.5)
	}

	if suite.TotalSpecsCount != 20 {
		t.Errorf("TotalSpecsCount = %v, want %v", suite.TotalSpecsCount, 20)
	}
}

func TestAnalytics_Creation(t *testing.T) {
	analytics := &Analytics{
		TotalExecutionTime:  10 * time.Minute,
		AverageSpecTime:     30 * time.Second,
		AverageScenarioTime: 5 * time.Second,
		SlowestSpecs: []*SpecPerformance{
			{SpecName: "Slow Spec", ExecutionTime: 2 * time.Minute, ScenarioCount: 5},
		},
		FastestSpecs: []*SpecPerformance{
			{SpecName: "Fast Spec", ExecutionTime: 10 * time.Second, ScenarioCount: 2},
		},
		TagDistribution:     map[string]int{"smoke": 10, "regression": 8},
		FailureDistribution: map[string]int{"assertion": 5, "timeout": 2},
	}

	if analytics.TotalExecutionTime != 10*time.Minute {
		t.Errorf("TotalExecutionTime = %v, want %v", analytics.TotalExecutionTime, 10*time.Minute)
	}

	if analytics.TagDistribution["smoke"] != 10 {
		t.Errorf("TagDistribution[smoke] = %v, want %v", analytics.TagDistribution["smoke"], 10)
	}

	if len(analytics.SlowestSpecs) != 1 {
		t.Errorf("SlowestSpecs length = %v, want %v", len(analytics.SlowestSpecs), 1)
	}
}

func TestFlakyTest_Creation(t *testing.T) {
	flakyTest := &FlakyTest{
		SpecName:          "Flaky E2E Test",
		ScenarioName:      "Login Flow",
		FlakyScore:        0.75,
		FailureRate:       0.25,
		ConsecutivePasses: 3,
		ConsecutiveFails:  1,
		LastSeen:          time.Now(),
		Occurrences:       12,
	}

	if flakyTest.SpecName != "Flaky E2E Test" {
		t.Errorf("SpecName = %v, want %v", flakyTest.SpecName, "Flaky E2E Test")
	}

	if flakyTest.FlakyScore != 0.75 {
		t.Errorf("FlakyScore = %v, want %v", flakyTest.FlakyScore, 0.75)
	}

	if flakyTest.Occurrences != 12 {
		t.Errorf("Occurrences = %v, want %v", flakyTest.Occurrences, 12)
	}
}

func TestStepResult_FailureHandling(t *testing.T) {
	step := &StepResult{
		StepText:      "When user logs in",
		ExecutionTime: 2 * time.Second,
		Failed:        true,
		ErrorMessage:  "Login failed: Invalid credentials",
		StackTrace:    "at login.py:45\n  at auth.py:123",
		Screenshots:   [][]byte{[]byte("screenshot1"), []byte("screenshot2")},
		Messages:      []string{"Attempting login", "Login failed"},
	}

	if step.Failed != true {
		t.Errorf("Failed = %v, want %v", step.Failed, true)
	}

	if step.ErrorMessage != "Login failed: Invalid credentials" {
		t.Errorf("ErrorMessage = %v, want expected message", step.ErrorMessage)
	}

	if len(step.Screenshots) != 2 {
		t.Errorf("Screenshots count = %v, want %v", len(step.Screenshots), 2)
	}

	if len(step.Messages) != 2 {
		t.Errorf("Messages count = %v, want %v", len(step.Messages), 2)
	}
}
