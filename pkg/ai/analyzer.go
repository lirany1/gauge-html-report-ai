package ai

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/lirany1/gauge-html-report-ai/pkg/models"
)

// Analyzer provides intelligent analysis of test results
type Analyzer struct {
	llmClient *LLMClient
	useRealAI bool
}

// NewAnalyzer creates a new AI analyzer
func NewAnalyzer() *Analyzer {
	// Try to load LLM config from environment
	llmConfig := LoadLLMConfigFromEnv()
	llmClient := NewLLMClient(llmConfig)

	return &Analyzer{
		llmClient: llmClient,
		useRealAI: llmClient != nil,
	}
}

// ErrorType represents different categories of test failures
type ErrorType string

const (
	ErrorTypeAssertion   ErrorType = "Assertion Failure"
	ErrorTypeTimeout     ErrorType = "Timeout"
	ErrorTypeNetwork     ErrorType = "Network Error"
	ErrorTypeNullPointer ErrorType = "Null Pointer"
	ErrorTypeFileSystem  ErrorType = "File System"
	ErrorTypeDatabase    ErrorType = "Database"
	ErrorTypeEnvironment ErrorType = "Environment"
	ErrorTypeUnknown     ErrorType = "Unknown Error"
)

// FailureGroup represents a group of similar failures
type FailureGroup struct {
	Signature         string
	ErrorType         ErrorType
	RootCause         string
	Count             int
	AffectedScenarios []string
	AffectedSpecs     []string
	Severity          string // "critical", "high", "medium", "low"
	SuggestedFix      string
	// Context for LLM analysis
	ErrorMessage string
	StackTrace   string
	StepText     string
	SpecName     string
}

// ExecutiveSummary contains high-level insights
type ExecutiveSummary struct {
	HealthStatus   string // "Excellent", "Good", "Fair", "Poor"
	KeyInsights    []string
	CriticalIssues []string
	TrendIndicator string // "Improving", "Stable", "Declining"
	Recommendation string
}

// ClassifyError determines the type of error based on message and stack trace
func (a *Analyzer) ClassifyError(errorMsg, stackTrace string) ErrorType {
	combined := strings.ToLower(errorMsg + " " + stackTrace)

	// Assertion patterns
	assertionPatterns := []string{
		"assertion", "assert", "expected", "actual", "should be",
		"must be", "equals", "not equal",
	}
	for _, pattern := range assertionPatterns {
		if strings.Contains(combined, pattern) {
			return ErrorTypeAssertion
		}
	}

	// Timeout patterns
	if strings.Contains(combined, "timeout") ||
		strings.Contains(combined, "timed out") ||
		strings.Contains(combined, "deadline exceeded") {
		return ErrorTypeTimeout
	}

	// Network patterns
	networkPatterns := []string{
		"connection refused", "network", "socket", "http",
		"connection reset", "connection closed", "dns",
	}
	for _, pattern := range networkPatterns {
		if strings.Contains(combined, pattern) {
			return ErrorTypeNetwork
		}
	}

	// Null pointer patterns
	if strings.Contains(combined, "null") ||
		strings.Contains(combined, "nil") ||
		strings.Contains(combined, "none") {
		return ErrorTypeNullPointer
	}

	// File system patterns
	filePatterns := []string{
		"file not found", "no such file", "permission denied",
		"directory", "path",
	}
	for _, pattern := range filePatterns {
		if strings.Contains(combined, pattern) {
			return ErrorTypeFileSystem
		}
	}

	// Database patterns
	dbPatterns := []string{
		"database", "sql", "query", "transaction",
		"duplicate key", "constraint",
	}
	for _, pattern := range dbPatterns {
		if strings.Contains(combined, pattern) {
			return ErrorTypeDatabase
		}
	}

	// Environment patterns
	envPatterns := []string{
		"environment", "config", "configuration",
		"property", "variable not set",
	}
	for _, pattern := range envPatterns {
		if strings.Contains(combined, pattern) {
			return ErrorTypeEnvironment
		}
	}

	return ErrorTypeUnknown
}

// GenerateErrorSignature creates a unique signature for similar errors
func (a *Analyzer) GenerateErrorSignature(errorMsg, errorType string) string {
	// Remove dynamic parts (numbers, timestamps, IDs)
	cleaned := errorMsg

	// Remove numbers
	re := regexp.MustCompile(`\d+`)
	cleaned = re.ReplaceAllString(cleaned, "N")

	// Remove file paths
	re = regexp.MustCompile(`/[^\s]+`)
	cleaned = re.ReplaceAllString(cleaned, "/PATH")

	// Remove UUIDs
	re = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	cleaned = re.ReplaceAllString(cleaned, "UUID")

	// Combine with error type
	signature := fmt.Sprintf("%s:%s", errorType, cleaned)

	// Generate hash
	hash := md5.Sum([]byte(signature))
	return hex.EncodeToString(hash[:])
}

// GroupFailures analyzes failures and groups similar ones
func (a *Analyzer) GroupFailures(suite *models.EnhancedSuiteResult) []*FailureGroup {
	groups := make(map[string]*FailureGroup)

	for _, spec := range suite.SpecResults {
		for _, scenario := range spec.Scenarios {
			if !scenario.Failed {
				continue
			}

			// Get first failed step
			var errorMsg, stackTrace, stepText string
			for _, step := range scenario.Steps {
				if step.Failed {
					errorMsg = step.ErrorMessage
					stackTrace = step.StackTrace
					stepText = step.StepText
					break
				}
			}

			if errorMsg == "" {
				continue
			}

			// Classify error
			errorType := a.ClassifyError(errorMsg, stackTrace)

			// Generate signature
			signature := a.GenerateErrorSignature(errorMsg, string(errorType))

			// Add to group or create new one
			if group, exists := groups[signature]; exists {
				group.Count++
				group.AffectedScenarios = append(group.AffectedScenarios, scenario.ScenarioHeading)
				if !contains(group.AffectedSpecs, spec.SpecHeading) {
					group.AffectedSpecs = append(group.AffectedSpecs, spec.SpecHeading)
				}
			} else {
				groups[signature] = &FailureGroup{
					Signature:         signature,
					ErrorType:         errorType,
					RootCause:         a.extractRootCause(errorMsg),
					Count:             1,
					AffectedScenarios: []string{scenario.ScenarioHeading},
					AffectedSpecs:     []string{spec.SpecHeading},
					Severity:          a.calculateSeverity(errorType, 1),
					SuggestedFix:      a.generateFixSuggestion(errorType, errorMsg, stackTrace, stepText, spec.SpecHeading),
					// Store context for potential LLM analysis
					ErrorMessage: errorMsg,
					StackTrace:   stackTrace,
					StepText:     stepText,
					SpecName:     spec.SpecHeading,
				}
			}
		}
	}

	// Convert map to slice and update severity based on count
	result := make([]*FailureGroup, 0, len(groups))
	for _, group := range groups {
		group.Severity = a.calculateSeverity(group.ErrorType, group.Count)
		result = append(result, group)
	}

	return result
}

// extractRootCause extracts the main error message
func (a *Analyzer) extractRootCause(errorMsg string) string {
	// Take first line or first 150 characters
	lines := strings.Split(errorMsg, "\n")
	if len(lines) > 0 {
		rootCause := strings.TrimSpace(lines[0])
		if len(rootCause) > 150 {
			return rootCause[:150] + "..."
		}
		return rootCause
	}
	return errorMsg
}

// calculateSeverity determines severity based on error type and frequency
func (a *Analyzer) calculateSeverity(errorType ErrorType, count int) string {
	// Critical: Multiple failures of certain types
	if count >= 3 {
		return "critical"
	}

	// High: Important error types
	switch errorType {
	case ErrorTypeAssertion:
		if count >= 2 {
			return "high"
		}
		return "medium"
	case ErrorTypeTimeout, ErrorTypeNetwork:
		return "high"
	case ErrorTypeDatabase:
		return "critical"
	case ErrorTypeNullPointer:
		return "high"
	default:
		return "medium"
	}
}

// generateFixSuggestion provides actionable fix recommendations
func (a *Analyzer) generateFixSuggestion(errorType ErrorType, errorMsg, stackTrace, stepText, specName string) string {
	// Try LLM first if available
	if a.useRealAI && a.llmClient != nil {
		llmSuggestion, err := a.llmClient.GenerateFixSuggestion(errorMsg, stackTrace, stepText, specName)
		if err == nil && llmSuggestion != "" {
			return llmSuggestion // Return AI-generated suggestion
		}
		// Log error but continue to fallback (could add logging here)
	}

	// Fallback to pattern-based suggestion
	return a.getPatternBasedSuggestion(errorType)
}

// getPatternBasedSuggestion returns rule-based fix suggestions
func (a *Analyzer) getPatternBasedSuggestion(errorType ErrorType) string {
	switch errorType {
	case ErrorTypeAssertion:
		return "Review test expectations and verify they match actual behavior. Check if application logic changed or test data is outdated."
	case ErrorTypeTimeout:
		return "Increase timeout values or investigate performance degradation. Check for slow external dependencies or resource constraints."
	case ErrorTypeNetwork:
		return "Verify network connectivity, check service availability, and ensure proper error handling for network failures."
	case ErrorTypeNullPointer:
		return "Add null checks before accessing objects. Verify object initialization and data flow in the application."
	case ErrorTypeFileSystem:
		return "Verify file paths, check file permissions, and ensure required files exist before test execution."
	case ErrorTypeDatabase:
		return "Check database connection, verify schema integrity, and ensure test data is properly set up."
	case ErrorTypeEnvironment:
		return "Review environment configuration, check required properties are set, and verify environment setup scripts."
	default:
		return "Review error logs and stack trace for more details. Consider adding more specific error handling."
	}
}

// GenerateExecutiveSummary creates a high-level summary
func (a *Analyzer) GenerateExecutiveSummary(suite *models.EnhancedSuiteResult, failureGroups []*FailureGroup) *ExecutiveSummary {
	summary := &ExecutiveSummary{
		KeyInsights:    make([]string, 0),
		CriticalIssues: make([]string, 0),
	}

	// Determine health status
	successRate := suite.SuccessRate
	switch {
	case successRate >= 95:
		summary.HealthStatus = "Excellent"
	case successRate >= 85:
		summary.HealthStatus = "Good"
	case successRate >= 70:
		summary.HealthStatus = "Fair"
	default:
		summary.HealthStatus = "Poor"
	}

	// Add key insights
	if suite.FailedScenariosCount == 0 {
		summary.KeyInsights = append(summary.KeyInsights,
			"✅ All tests passed successfully - no failures detected")
	} else {
		summary.KeyInsights = append(summary.KeyInsights,
			fmt.Sprintf("⚠️ %d scenario(s) failed out of %d total",
				suite.FailedScenariosCount, suite.TotalScenariosCount))
	}

	// Analyze failure distribution
	if len(failureGroups) > 0 {
		uniqueErrors := len(failureGroups)
		totalFailures := suite.FailedScenariosCount

		if uniqueErrors == 1 {
			summary.KeyInsights = append(summary.KeyInsights,
				"🔍 Single root cause identified - focused fix possible")
		} else if uniqueErrors < totalFailures {
			summary.KeyInsights = append(summary.KeyInsights,
				fmt.Sprintf("🔍 %d unique failure patterns detected", uniqueErrors))
		}
	}

	// Check for flaky tests
	if len(suite.FlakyTests) > 0 {
		summary.KeyInsights = append(summary.KeyInsights,
			fmt.Sprintf("🔄 %d flaky test(s) detected - needs stabilization", len(suite.FlakyTests)))
	}

	// Identify critical issues
	for _, group := range failureGroups {
		if group.Severity == "critical" || group.Severity == "high" {
			summary.CriticalIssues = append(summary.CriticalIssues,
				fmt.Sprintf("%s: %s (affects %d scenario(s))",
					group.ErrorType, group.RootCause, group.Count))
		}
	}

	// Trend indicator (based on success rate)
	if suite.Trends != nil && len(suite.Trends.HistoricalRuns) > 1 {
		current := suite.SuccessRate
		previous := suite.Trends.HistoricalRuns[len(suite.Trends.HistoricalRuns)-2].SuccessRate

		diff := current - previous
		if diff > 5 {
			summary.TrendIndicator = "📈 Improving"
		} else if diff < -5 {
			summary.TrendIndicator = "📉 Declining"
		} else {
			summary.TrendIndicator = "📊 Stable"
		}
	} else {
		summary.TrendIndicator = "📊 Baseline"
	}

	// Generate recommendation
	summary.Recommendation = a.generateRecommendation(summary)

	return summary
}

// generateRecommendation creates actionable recommendations
func (a *Analyzer) generateRecommendation(summary *ExecutiveSummary) string {
	if summary.HealthStatus == "Excellent" {
		return "Continue maintaining high quality standards. Monitor for any new flaky tests."
	}

	if len(summary.CriticalIssues) > 0 {
		return "Address critical failures immediately before proceeding with new deployments."
	}

	if summary.TrendIndicator == "📉 Declining" {
		return "Investigate declining success rate. Review recent changes and consider rolling back if necessary."
	}

	return "Focus on stabilizing failing scenarios. Prioritize fixes based on failure frequency."
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
