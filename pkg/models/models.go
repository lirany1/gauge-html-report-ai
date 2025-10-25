package models

import (
	"path/filepath"
	"strings"
	"time"
)

// EnhancedSuiteResult represents the complete test suite execution with analytics
type EnhancedSuiteResult struct {
	ProjectName   string
	Environment   string
	Tags          []string
	ExecutionTime time.Duration
	Timestamp     time.Time
	SuccessRate   float64

	// Counts
	PassedSpecsCount      int
	FailedSpecsCount      int
	SkippedSpecsCount     int
	TotalSpecsCount       int
	PassedScenariosCount  int
	FailedScenariosCount  int
	SkippedScenariosCount int
	TotalScenariosCount   int

	// Results
	SpecResults        []*SpecResult
	BeforeSuiteFailure *HookFailure
	AfterSuiteFailure  *HookFailure
	Messages           []string
	Screenshots        [][]byte

	// Enhanced analytics
	Analytics          *Analytics
	Trends             *TrendData
	FlakyTests         []*FlakyTest
	PerformanceMetrics *PerformanceMetrics
	AIInsights         *AIInsights
}

// SpecResult represents a single specification execution
type SpecResult struct {
	SpecHeading   string
	FileName      string
	Tags          []string
	ExecutionTime time.Duration
	Failed        bool
	Skipped       bool
	Scenarios     []*ScenarioResult
	Errors        []BuildError
	Messages      []string
	Screenshots   [][]byte
}

// ScenarioResult represents a single scenario execution
type ScenarioResult struct {
	ScenarioHeading string
	Tags            []string
	ExecutionTime   time.Duration
	Failed          bool
	Skipped         bool
	Steps           []*StepResult
	TableRows       int
	Messages        []string
	Screenshots     [][]byte
}

// StepResult represents a single step execution
type StepResult struct {
	StepText      string
	ExecutionTime time.Duration
	Failed        bool
	Skipped       bool
	ErrorMessage  string
	StackTrace    string
	Screenshots   [][]byte
	Messages      []string
}

// HookFailure represents a hook execution failure
type HookFailure struct {
	ErrorMessage string
	StackTrace   string
	Screenshot   []byte
}

// BuildError represents a parse or validation error
type BuildError struct {
	Type    string
	Message string
	Line    int
	Column  int
}

// Analytics holds advanced analytics data
type Analytics struct {
	TotalExecutionTime  time.Duration
	AverageSpecTime     time.Duration
	AverageScenarioTime time.Duration
	SlowestSpecs        []*SpecPerformance
	FastestSpecs        []*SpecPerformance
	MostFailedSpecs     []*SpecFailureCount
	TagDistribution     map[string]int
	FailureDistribution map[string]int
	TimelineData        []*TimelineEntry
}

// TrendData holds historical trend information
type TrendData struct {
	HistoricalRuns     []*HistoricalRun
	SuccessRateTrend   []float64
	ExecutionTimeTrend []time.Duration
	FlakyTestTrend     []int
	FailureRateTrend   []float64
	Predictions        *TrendPredictions
}

// HistoricalRun represents a single historical test run
type HistoricalRun struct {
	Timestamp     time.Time
	SuccessRate   float64
	ExecutionTime time.Duration
	PassedCount   int
	FailedCount   int
	SkippedCount  int
	BuildNumber   string
	GitCommit     string
}

// TrendPredictions holds ML-based predictions
type TrendPredictions struct {
	NextRunPrediction  *RunPrediction
	QualityTrend       string // "improving", "degrading", "stable"
	EstimatedFixTime   time.Duration
	RecommendedActions []string
}

// RunPrediction predicts next run outcomes
type RunPrediction struct {
	PredictedSuccessRate float64
	PredictedDuration    time.Duration
	Confidence           float64
}

// FlakyTest represents a test that behaves inconsistently
type FlakyTest struct {
	SpecName          string
	ScenarioName      string
	FlakyScore        float64
	FailureRate       float64
	ConsecutivePasses int
	ConsecutiveFails  int
	LastSeen          time.Time
	Occurrences       int
}

// PerformanceMetrics holds performance analysis data
type PerformanceMetrics struct {
	TotalCPUTime    time.Duration
	TotalMemoryUsed int64
	PeakMemoryUsage int64
	ThreadCount     int
	Bottlenecks     []*Bottleneck
}

// Bottleneck represents a performance bottleneck
type Bottleneck struct {
	Location        string
	Type            string // "slow_step", "memory_leak", "cpu_intensive"
	Severity        string // "low", "medium", "high", "critical"
	Impact          time.Duration
	Recommendations []string
}

// SpecPerformance tracks specification performance
type SpecPerformance struct {
	SpecName      string
	ExecutionTime time.Duration
	ScenarioCount int
}

// SpecFailureCount tracks specification failures
type SpecFailureCount struct {
	SpecName     string
	FailureCount int
	LastFailure  time.Time
}

// TimelineEntry represents a point in the execution timeline
type TimelineEntry struct {
	Timestamp time.Time
	Event     string // "spec_start", "spec_end", "failure", "success"
	SpecName  string
	Duration  time.Duration
	Status    string
}

// GetHTMLFileName returns the HTML filename for a spec
func (s *SpecResult) GetHTMLFileName() string {
	base := filepath.Base(s.FileName)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return name + ".html"
}

// GetStatus returns the status string for a spec
func (s *SpecResult) GetStatus() string {
	if s.Failed {
		return "failed"
	}
	if s.Skipped {
		return "skipped"
	}
	return "passed"
}

// GetFailedScenariosCount returns count of failed scenarios
func (s *SpecResult) GetFailedScenariosCount() int {
	count := 0
	for _, scenario := range s.Scenarios {
		if scenario.Failed {
			count++
		}
	}
	return count
}

// GetPassedScenariosCount returns count of passed scenarios
func (s *SpecResult) GetPassedScenariosCount() int {
	count := 0
	for _, scenario := range s.Scenarios {
		if !scenario.Failed && !scenario.Skipped {
			count++
		}
	}
	return count
}

// GetSkippedScenariosCount returns count of skipped scenarios
func (s *SpecResult) GetSkippedScenariosCount() int {
	count := 0
	for _, scenario := range s.Scenarios {
		if scenario.Skipped {
			count++
		}
	}
	return count
}

// AIInsights contains intelligent analysis and recommendations
type AIInsights struct {
	ExecutiveSummary *ExecutiveSummary
	FailureGroups    []*FailureGroup
}

// ExecutiveSummary provides high-level test health assessment
type ExecutiveSummary struct {
	HealthStatus   string
	KeyInsights    []string
	CriticalIssues []string
	TrendIndicator string
	Recommendation string
}

// FailureGroup represents grouped similar failures
type FailureGroup struct {
	Signature         string
	ErrorType         string
	RootCause         string
	Count             int
	AffectedScenarios []string
	AffectedSpecs     []string
	Severity          string
	SuggestedFix      string
}
