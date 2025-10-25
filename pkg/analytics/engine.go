package analytics

import (
	"fmt"
	"sort"
	"time"

	"github.com/your-org/gauge-html-report-enhanced/pkg/config"
	"github.com/your-org/gauge-html-report-enhanced/pkg/models"
	"github.com/your-org/gauge-html-report-enhanced/pkg/storage"
)

// Engine handles analytics processing with database integration
type Engine struct {
	config *config.Config
	db     *storage.Database
}

// NewEngine creates a new analytics engine with database support
func NewEngine(cfg *config.Config, db *storage.Database) *Engine {
	return &Engine{
		config: cfg,
		db:     db,
	}
}

// Analyze performs comprehensive analysis on test results
func (e *Engine) Analyze(suite *models.EnhancedSuiteResult) *models.Analytics {
	analytics := &models.Analytics{
		TotalExecutionTime:  suite.ExecutionTime,
		TagDistribution:     e.calculateTagDistribution(suite),
		FailureDistribution: e.calculateFailureDistribution(suite),
		SlowestSpecs:        e.findSlowestSpecs(suite, 5),
		MostFailedSpecs:     e.findMostFailedSpecs(suite, 5),
		TimelineData:        e.generateTimeline(suite),
	}

	// Calculate averages
	totalScenarios := 0
	totalSpecTime := time.Duration(0)
	totalScenarioTime := time.Duration(0)

	for _, spec := range suite.SpecResults {
		totalSpecTime += spec.ExecutionTime
		for _, scenario := range spec.Scenarios {
			totalScenarioTime += scenario.ExecutionTime
			totalScenarios++
		}
	}

	if len(suite.SpecResults) > 0 {
		analytics.AverageSpecTime = totalSpecTime / time.Duration(len(suite.SpecResults))
	}
	if totalScenarios > 0 {
		analytics.AverageScenarioTime = totalScenarioTime / time.Duration(totalScenarios)
	}

	return analytics
}

// calculateTagDistribution counts scenarios by tag
func (e *Engine) calculateTagDistribution(suite *models.EnhancedSuiteResult) map[string]int {
	distribution := make(map[string]int)

	for _, spec := range suite.SpecResults {
		for _, scenario := range spec.Scenarios {
			for _, tag := range scenario.Tags {
				distribution[tag]++
			}
		}
	}

	return distribution
}

// calculateFailureDistribution categorizes failures
func (e *Engine) calculateFailureDistribution(suite *models.EnhancedSuiteResult) map[string]int {
	distribution := make(map[string]int)

	for _, spec := range suite.SpecResults {
		for _, scenario := range spec.Scenarios {
			if scenario.Failed {
				// Look at step errors
				hasError := false
				for _, step := range scenario.Steps {
					if step.Failed && step.ErrorMessage != "" {
						errorType := categorizeError(step.ErrorMessage)
						distribution[errorType]++
						hasError = true
						break
					}
				}
				if !hasError {
					distribution["Unknown"]++
				}
			}
		}
	}

	return distribution
}

// categorizeError classifies error messages
func categorizeError(message string) string {
	// Simple categorization logic
	messageLower := message
	if len(messageLower) > 100 {
		messageLower = messageLower[:100]
	}

	// Common error patterns
	patterns := map[string][]string{
		"Assertion":  {"assert", "expected", "actual"},
		"Timeout":    {"timeout", "timed out", "deadline"},
		"Connection": {"connection", "network", "unreachable"},
		"NotFound":   {"not found", "404", "missing"},
		"Permission": {"permission", "denied", "unauthorized", "403"},
		"Null":       {"null", "nil", "undefined"},
		"TypeError":  {"type error", "cannot convert", "invalid type"},
	}

	for category, keywords := range patterns {
		for _, keyword := range keywords {
			if len(messageLower) >= len(keyword) {
				// Simple substring check
				found := false
				for i := 0; i <= len(messageLower)-len(keyword); i++ {
					if messageLower[i:i+len(keyword)] == keyword {
						found = true
						break
					}
				}
				if found {
					return category
				}
			}
		}
	}

	return "Other"
}

// findSlowestSpecs returns the N slowest specs
func (e *Engine) findSlowestSpecs(suite *models.EnhancedSuiteResult, n int) []*models.SpecPerformance {
	var specs []*models.SpecPerformance

	for _, spec := range suite.SpecResults {
		specs = append(specs, &models.SpecPerformance{
			SpecName:      spec.SpecHeading,
			ExecutionTime: spec.ExecutionTime,
			ScenarioCount: len(spec.Scenarios),
		})
	}

	// Sort by execution time descending
	sort.Slice(specs, func(i, j int) bool {
		return specs[i].ExecutionTime > specs[j].ExecutionTime
	})

	// Take top N
	if len(specs) > n {
		specs = specs[:n]
	}

	return specs
}

// findMostFailedSpecs returns the N most failed specs
func (e *Engine) findMostFailedSpecs(suite *models.EnhancedSuiteResult, n int) []*models.SpecFailureCount {
	var failedSpecs []*models.SpecFailureCount

	for _, spec := range suite.SpecResults {
		if spec.Failed {
			failedCount := spec.GetFailedScenariosCount()
			if failedCount > 0 {
				failedSpecs = append(failedSpecs, &models.SpecFailureCount{
					SpecName:     spec.SpecHeading,
					FailureCount: failedCount,
					LastFailure:  time.Now(),
				})
			}
		}
	}

	// Sort by failure count descending
	sort.Slice(failedSpecs, func(i, j int) bool {
		return failedSpecs[i].FailureCount > failedSpecs[j].FailureCount
	})

	// Take top N
	if len(failedSpecs) > n {
		failedSpecs = failedSpecs[:n]
	}

	return failedSpecs
}

// generateTimeline creates execution timeline
func (e *Engine) generateTimeline(suite *models.EnhancedSuiteResult) []*models.TimelineEntry {
	var timeline []*models.TimelineEntry
	currentTime := suite.Timestamp

	for _, spec := range suite.SpecResults {
		// Spec start
		timeline = append(timeline, &models.TimelineEntry{
			Timestamp: currentTime,
			Event:     "spec_start",
			SpecName:  spec.SpecHeading,
			Duration:  spec.ExecutionTime,
			Status:    spec.GetStatus(),
		})

		currentTime = currentTime.Add(spec.ExecutionTime)

		// Spec end
		event := "success"
		if spec.Failed {
			event = "failure"
		}
		timeline = append(timeline, &models.TimelineEntry{
			Timestamp: currentTime,
			Event:     event,
			SpecName:  spec.SpecHeading,
			Duration:  spec.ExecutionTime,
			Status:    spec.GetStatus(),
		})
	}

	return timeline
}

// GenerateTrends creates historical trend data using database
func (e *Engine) GenerateTrends(suite *models.EnhancedSuiteResult) *models.TrendData {
	if e.db == nil {
		// Return empty trends if no database
		return &models.TrendData{
			HistoricalRuns: make([]*models.HistoricalRun, 0),
		}
	}

	// Get trend data for last 30 days
	trends, err := e.db.GetTrendData(30)
	if err != nil {
		return &models.TrendData{
			HistoricalRuns: make([]*models.HistoricalRun, 0),
		}
	}

	// Convert to HistoricalRun format
	historicalRuns := make([]*models.HistoricalRun, len(trends))
	successRateTrend := make([]float64, len(trends))
	executionTimeTrend := make([]time.Duration, len(trends))

	for i, trend := range trends {
		historicalRuns[i] = &models.HistoricalRun{
			Timestamp:     trend.Timestamp,
			SuccessRate:   trend.SuccessRate,
			ExecutionTime: time.Duration(trend.Duration) * time.Millisecond,
			PassedCount:   trend.Passed,
			FailedCount:   trend.Failed,
			BuildNumber:   "",
			GitCommit:     "",
		}
		successRateTrend[i] = trend.SuccessRate
		executionTimeTrend[i] = time.Duration(trend.Duration) * time.Millisecond
	}

	trendData := &models.TrendData{
		HistoricalRuns:     historicalRuns,
		SuccessRateTrend:   successRateTrend,
		ExecutionTimeTrend: executionTimeTrend,
	}

	// Calculate trend predictions
	if len(historicalRuns) >= 2 {
		current := historicalRuns[len(historicalRuns)-1]
		previous := historicalRuns[len(historicalRuns)-2]

		successChange := current.SuccessRate - previous.SuccessRate
		qualityTrend := "stable"
		if successChange > 5.0 {
			qualityTrend = "improving"
		} else if successChange < -5.0 {
			qualityTrend = "degrading"
		}

		trendData.Predictions = &models.TrendPredictions{
			QualityTrend: qualityTrend,
			NextRunPrediction: &models.RunPrediction{
				PredictedSuccessRate: current.SuccessRate,
				PredictedDuration:    current.ExecutionTime,
				Confidence:           0.7,
			},
		}
	}

	return trendData
}

// DetectFlakyTests identifies inconsistent tests using database history
func (e *Engine) DetectFlakyTests(suite *models.EnhancedSuiteResult) []*models.FlakyTest {
	flakyTests := make([]*models.FlakyTest, 0)

	if e.db == nil {
		return flakyTests
	}

	flakyThreshold := 0.3 // Score > 0.3 is considered flaky

	for _, spec := range suite.SpecResults {
		for _, scenario := range spec.Scenarios {
			score, err := e.db.CalculateFlakyScore(scenario.ScenarioHeading, 30)
			if err != nil {
				continue
			}

			if score > flakyThreshold {
				// Get history to calculate additional metrics
				history, _ := e.db.GetScenarioHistory(scenario.ScenarioHeading, 30)

				totalRuns := len(history)
				failedRuns := 0
				for _, h := range history {
					if h.Status == "failed" {
						failedRuns++
					}
				}

				failureRate := 0.0
				if totalRuns > 0 {
					failureRate = float64(failedRuns) / float64(totalRuns) * 100
				}

				flakyTests = append(flakyTests, &models.FlakyTest{
					SpecName:     spec.SpecHeading,
					ScenarioName: scenario.ScenarioHeading,
					FlakyScore:   score,
					FailureRate:  failureRate,
					LastSeen:     time.Now(),
					Occurrences:  totalRuns,
				})
			}
		}
	}

	return flakyTests
}

// SaveExecutionData saves current execution to database for historical tracking
func (e *Engine) SaveExecutionData(suite *models.EnhancedSuiteResult, executionID string) error {
	if e.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Calculate totals
	passed := suite.PassedScenariosCount
	failed := suite.FailedScenariosCount
	skipped := suite.SkippedScenariosCount
	total := suite.TotalScenariosCount
	successRate := suite.SuccessRate

	// Save execution record
	execution := &storage.ExecutionRecord{
		ID:               executionID,
		Timestamp:        suite.Timestamp,
		TotalScenarios:   total,
		PassedScenarios:  passed,
		FailedScenarios:  failed,
		SkippedScenarios: skipped,
		SuccessRate:      successRate,
		Duration:         int64(suite.ExecutionTime.Milliseconds()),
		Environment:      suite.Environment,
		Tags:             suite.Tags,
		Metadata:         make(map[string]interface{}),
	}

	if err := e.db.SaveExecution(execution); err != nil {
		return fmt.Errorf("failed to save execution: %w", err)
	}

	// Save individual scenario records
	for _, spec := range suite.SpecResults {
		for _, scenario := range spec.Scenarios {
			status := "passed"
			errorMessage := ""
			stackTrace := ""

			if scenario.Failed {
				status = "failed"
				// Get error message from first failed step
				for _, step := range scenario.Steps {
					if step.Failed {
						errorMessage = step.ErrorMessage
						stackTrace = step.StackTrace
						break
					}
				}
			} else if scenario.Skipped {
				status = "skipped"
			}

			scenarioRecord := &storage.ScenarioRecord{
				ExecutionID:  executionID,
				ScenarioName: scenario.ScenarioHeading,
				SpecName:     spec.SpecHeading,
				Status:       status,
				Duration:     int64(scenario.ExecutionTime.Milliseconds()),
				ErrorMessage: errorMessage,
				StackTrace:   stackTrace,
			}

			if err := e.db.SaveScenario(scenarioRecord); err != nil {
				// Log error but continue
				continue
			}
		}
	}

	return nil
}

// CalculateSuccessRate calculates the success rate percentage
func CalculateSuccessRate(passed, total int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(passed) / float64(total) * 100.0
}

// FormatDuration formats a duration to a readable string
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
}
