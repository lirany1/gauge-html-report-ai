package builder

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/getgauge/gauge-proto/go/gauge_messages"
	"github.com/google/uuid"
	"github.com/lirany1/gauge-html-report-ai/pkg/ai"
	"github.com/lirany1/gauge-html-report-ai/pkg/analytics"
	"github.com/lirany1/gauge-html-report-ai/pkg/config"
	"github.com/lirany1/gauge-html-report-ai/pkg/logger"
	"github.com/lirany1/gauge-html-report-ai/pkg/models"
	"github.com/lirany1/gauge-html-report-ai/pkg/storage"
)

// ReportBuilder handles building the HTML report
type ReportBuilder struct {
	reportsDir string
	themePath  string
	config     *config.Config
	db         *storage.Database
	analytics  *analytics.Engine
	ai         *ai.Analyzer
}

// NewReportBuilder creates a new report builder with analytics integration
func NewReportBuilder(reportsDir, themePath string) *ReportBuilder {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Warnf("Failed to load config, using defaults: %v", err)
		cfg = config.DefaultConfig()
	}

	// Initialize database
	db, err := storage.NewDatabase(reportsDir)
	if err != nil {
		logger.Warnf("Failed to initialize database: %v", err)
		logger.Warnf("Historical data and trends will not be available")
		db = nil
	}

	// Initialize analytics engine
	analyticsEngine := analytics.NewEngine(cfg, db)

	// Initialize AI analyzer
	aiAnalyzer := ai.NewAnalyzer()

	return &ReportBuilder{
		reportsDir: reportsDir,
		themePath:  themePath,
		config:     cfg,
		db:         db,
		analytics:  analyticsEngine,
		ai:         aiAnalyzer,
	}
}

// Close releases database resources
func (rb *ReportBuilder) Close() error {
	if rb.db != nil {
		return rb.db.Close()
	}
	return nil
}

// BuildReport generates the HTML report from suite results
func (rb *ReportBuilder) BuildReport(suiteResult *gauge_messages.ProtoSuiteResult) error {
	// Create reports directory
	reportDir := filepath.Join(rb.reportsDir, "html-report")
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return fmt.Errorf("failed to create report directory: %w", err)
	}

	// Convert proto result to enhanced suite result
	enhanced := rb.convertToEnhancedSuite(suiteResult)

	// Run analytics
	enhanced.Analytics = rb.analytics.Analyze(enhanced)
	enhanced.Trends = rb.analytics.GenerateTrends(enhanced)
	enhanced.FlakyTests = rb.analytics.DetectFlakyTests(enhanced)

	// Run AI analysis
	failureGroups := rb.ai.GroupFailures(enhanced)
	executiveSummary := rb.ai.GenerateExecutiveSummary(enhanced, failureGroups)

	// Convert AI failure groups to model format
	modelFailureGroups := make([]*models.FailureGroup, len(failureGroups))
	for i, fg := range failureGroups {
		modelFailureGroups[i] = &models.FailureGroup{
			Signature:         fg.Signature,
			ErrorType:         string(fg.ErrorType),
			RootCause:         fg.RootCause,
			Count:             fg.Count,
			AffectedScenarios: fg.AffectedScenarios,
			AffectedSpecs:     fg.AffectedSpecs,
			Severity:          fg.Severity,
			SuggestedFix:      fg.SuggestedFix,
		}
	}

	// Convert executive summary to model format
	modelExecSummary := &models.ExecutiveSummary{
		HealthStatus:   executiveSummary.HealthStatus,
		KeyInsights:    executiveSummary.KeyInsights,
		CriticalIssues: executiveSummary.CriticalIssues,
		TrendIndicator: executiveSummary.TrendIndicator,
		Recommendation: executiveSummary.Recommendation,
	}

	enhanced.AIInsights = &models.AIInsights{
		ExecutiveSummary: modelExecSummary,
		FailureGroups:    modelFailureGroups,
	}

	// Save to database for historical tracking
	if rb.db != nil {
		executionID := uuid.New().String()
		if err := rb.analytics.SaveExecutionData(enhanced, executionID); err != nil {
			logger.Warnf("Failed to save execution data: %v", err)
		} else {
			logger.Infof("Saved execution data with ID: %s", executionID)
		}
	}

	// Copy theme assets
	if err := rb.copyAssets(reportDir); err != nil {
		logger.Warnf("Failed to copy assets: %v", err)
	}

	// Generate index.html
	if err := rb.generateIndexHTML(reportDir, enhanced); err != nil {
		return fmt.Errorf("failed to generate index.html: %w", err)
	}

	logger.Infof("Successfully generated html-report to => %s/index.html", reportDir)
	return nil
}

// convertToEnhancedSuite converts proto result to enhanced model
func (rb *ReportBuilder) convertToEnhancedSuite(proto *gauge_messages.ProtoSuiteResult) *models.EnhancedSuiteResult {
	// Get tags (proto returns comma-separated string, we need to split it)
	tags := make([]string, 0)
	if proto.GetTags() != "" {
		// Split comma-separated tags
		for _, tag := range splitTags(proto.GetTags()) {
			tags = append(tags, tag)
		}
	}

	suite := &models.EnhancedSuiteResult{
		ProjectName:   proto.GetProjectName(),
		Environment:   proto.GetEnvironment(),
		Tags:          tags,
		ExecutionTime: time.Duration(proto.GetExecutionTime()) * time.Millisecond,
		Timestamp:     time.Now(),
		SpecResults:   make([]*models.SpecResult, 0),
	}

	// Convert spec results
	for _, protoSpec := range proto.GetSpecResults() {
		spec := rb.convertSpecResult(protoSpec)
		suite.SpecResults = append(suite.SpecResults, spec)

		// Update counts
		suite.TotalSpecsCount++
		if spec.Failed {
			suite.FailedSpecsCount++
		} else if spec.Skipped {
			suite.SkippedSpecsCount++
		} else {
			suite.PassedSpecsCount++
		}

		// Count scenarios
		for _, scenario := range spec.Scenarios {
			suite.TotalScenariosCount++
			if scenario.Failed {
				suite.FailedScenariosCount++
			} else if scenario.Skipped {
				suite.SkippedScenariosCount++
			} else {
				suite.PassedScenariosCount++
			}
		}
	}

	// Calculate success rate
	if suite.TotalScenariosCount > 0 {
		suite.SuccessRate = float64(suite.PassedScenariosCount) / float64(suite.TotalScenariosCount) * 100
	}

	// Convert hook failures
	if proto.GetPreHookFailure() != nil {
		suite.BeforeSuiteFailure = rb.convertHookFailure(proto.GetPreHookFailure())
	}
	if proto.GetPostHookFailure() != nil {
		suite.AfterSuiteFailure = rb.convertHookFailure(proto.GetPostHookFailure())
	}

	return suite
}

// convertSpecResult converts a proto spec result
func (rb *ReportBuilder) convertSpecResult(proto *gauge_messages.ProtoSpecResult) *models.SpecResult {
	spec := &models.SpecResult{
		SpecHeading:   proto.GetProtoSpec().GetSpecHeading(),
		FileName:      proto.GetProtoSpec().GetFileName(),
		Tags:          proto.GetProtoSpec().GetTags(),
		ExecutionTime: time.Duration(proto.GetExecutionTime()) * time.Millisecond,
		Failed:        proto.GetFailed(),
		Skipped:       proto.GetSkipped(),
		Scenarios:     make([]*models.ScenarioResult, 0),
		Errors:        make([]models.BuildError, 0),
	}

	// Convert scenarios from proto items
	for _, item := range proto.GetProtoSpec().GetItems() {
		if item.GetItemType() == gauge_messages.ProtoItem_Scenario {
			scenario := rb.convertScenario(item.GetScenario())
			spec.Scenarios = append(spec.Scenarios, scenario)
		}
	}

	return spec
}

// convertScenario converts a proto scenario
func (rb *ReportBuilder) convertScenario(proto *gauge_messages.ProtoScenario) *models.ScenarioResult {
	failed := proto.GetFailed()
	skipped := proto.GetSkipped()

	// Debug logging
	logger.Debugf("Converting scenario '%s': Failed=%v, Skipped=%v", proto.GetScenarioHeading(), failed, skipped)

	scenario := &models.ScenarioResult{
		ScenarioHeading: proto.GetScenarioHeading(),
		Tags:            proto.GetTags(),
		ExecutionTime:   time.Duration(proto.GetExecutionTime()) * time.Millisecond,
		Failed:          failed,
		Skipped:         skipped,
		Steps:           make([]*models.StepResult, 0),
	}

	// Convert scenario items (steps)
	for _, item := range proto.GetScenarioItems() {
		if item.GetItemType() == gauge_messages.ProtoItem_Step {
			step := rb.convertStep(item.GetStep())
			scenario.Steps = append(scenario.Steps, step)
			// If any step failed, mark scenario as failed
			if step.Failed {
				scenario.Failed = true
				logger.Debugf("Scenario '%s' marked as failed due to step: %s", scenario.ScenarioHeading, step.StepText)
			}
		}
	}

	return scenario
}

// convertStep converts a proto step
func (rb *ReportBuilder) convertStep(proto *gauge_messages.ProtoStep) *models.StepResult {
	// Get step text from parsed step text
	stepText := ""
	if proto.GetParsedText() != "" {
		stepText = proto.GetParsedText()
	}

	execResult := proto.GetStepExecutionResult().GetExecutionResult()

	step := &models.StepResult{
		StepText:      stepText,
		ExecutionTime: time.Duration(execResult.GetExecutionTime()) * time.Millisecond,
		Failed:        execResult.GetFailed(),
		Skipped:       proto.GetStepExecutionResult().GetSkipped(),
	}

	// Extract error information
	if step.Failed {
		step.ErrorMessage = execResult.GetErrorMessage()
		step.StackTrace = execResult.GetStackTrace()
	}

	return step
}

// splitTags splits a comma-separated string into a slice
func splitTags(tags string) []string {
	if tags == "" {
		return []string{}
	}

	result := make([]string, 0)
	for _, tag := range splitString(tags, ",") {
		trimmed := trimSpace(tag)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// splitString is a simple string split helper
func splitString(s, sep string) []string {
	if s == "" {
		return []string{}
	}

	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i < len(s) && s[i:i+1] == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

// trimSpace trims leading and trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && isSpace(s[start]) {
		start++
	}

	for end > start && isSpace(s[end-1]) {
		end--
	}

	return s[start:end]
}

// isSpace checks if a byte is whitespace
func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// convertHookFailure converts a proto hook failure
func (rb *ReportBuilder) convertHookFailure(proto *gauge_messages.ProtoHookFailure) *models.HookFailure {
	return &models.HookFailure{
		ErrorMessage: proto.GetErrorMessage(),
		StackTrace:   proto.GetStackTrace(),
	}
}

// generateIndexHTML creates the main index.html file
func (rb *ReportBuilder) generateIndexHTML(reportDir string, suite *models.EnhancedSuiteResult) error {
	// Create HTML from template
	tmpl := rb.getTemplate()

	indexPath := filepath.Join(reportDir, "index.html")
	f, err := os.Create(indexPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Pass the suite struct directly to the template
	return tmpl.Execute(f, suite)
}

// getTemplate returns the HTML template
func (rb *ReportBuilder) getTemplate() *template.Template {
	// Define template functions
	funcMap := template.FuncMap{
		"getStatus": func(spec *models.SpecResult) string {
			return spec.GetStatus()
		},
		"getSpecHeading": func(spec *models.SpecResult) string {
			return spec.SpecHeading
		},
		"getScenarioCount": func(spec *models.SpecResult) int {
			return len(spec.Scenarios)
		},
		"getTags": func(spec *models.SpecResult) []string {
			return spec.Tags
		},
		"formatDuration": func(d time.Duration) string {
			return analytics.FormatDuration(d)
		},
		"formatSuccessRate": func(rate float64) string {
			return fmt.Sprintf("%.1f", rate)
		},
		"formatTimestamp": func(t time.Time) string {
			return t.Format("January 2, 2006 at 3:04 PM")
		},
		"getFailedScenariosCount": func(spec *models.SpecResult) int {
			return spec.GetFailedScenariosCount()
		},
		"getPassedScenariosCount": func(spec *models.SpecResult) int {
			return spec.GetPassedScenariosCount()
		},
		"getSkippedScenariosCount": func(spec *models.SpecResult) int {
			return spec.GetSkippedScenariosCount()
		},
	}

	tmplStr := rb.getTemplateString()
	return template.Must(template.New("index").Funcs(funcMap).Parse(tmplStr))
}

// copyAssets copies CSS, JS, and other assets to the report directory
func (rb *ReportBuilder) copyAssets(reportDir string) error {
	// Create asset directories
	dirs := []string{"css", "js", "images"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(reportDir, dir), 0755); err != nil {
			return err
		}
	}

	// Copy main CSS
	if err := rb.copyFile("main.css", filepath.Join(reportDir, "css", "main.css")); err != nil {
		logger.Warnf("Could not copy main.css: %v", err)
	}

	// Copy main JS
	if err := rb.copyFile("main.js", filepath.Join(reportDir, "js", "main.js")); err != nil {
		logger.Warnf("Could not copy main.js: %v", err)
	}

	return nil
}

// copyFile copies a file from theme to destination
func (rb *ReportBuilder) copyFile(filename, dest string) error {
	// For now, just create empty files - in full implementation,
	// we would copy from the theme directory
	return os.WriteFile(dest, []byte("/* Asset file */"), 0644)
}

// formatDuration formats a duration to a readable string
func formatDuration(d time.Duration) string {
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

// getTemplateString returns the Executive Dashboard HTML template
func (rb *ReportBuilder) getTemplateString() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.ProjectName}} - Test Execution Report</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js" defer></script>
    <style>
        @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap');
        body { font-family: 'Inter', sans-serif; }
        .trend-up { color: #10b981; }
        .trend-down { color: #ef4444; }
        .trend-neutral { color: #6b7280; }
    </style>
</head>
<body class="bg-gray-50" x-data="{ activeTab: 'overview', showFilters: false }">
    <!-- Header -->
    <header class="bg-white border-b border-gray-200 sticky top-0 z-50 shadow-sm">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
            <div class="flex items-center justify-between">
                <div>
                    <h1 class="text-2xl font-bold text-gray-900">{{.ProjectName}}</h1>
                    <p class="text-sm text-gray-500 mt-1">Test Execution Report - {{formatTimestamp .Timestamp}}</p>
                </div>
                <div class="flex items-center gap-4">
                    <span class="px-3 py-1 rounded-full text-sm font-medium {{if gt .SuccessRate 80.0}}bg-green-100 text-green-800{{else if gt .SuccessRate 50.0}}bg-yellow-100 text-yellow-800{{else}}bg-red-100 text-red-800{{end}}">
                        {{formatSuccessRate .SuccessRate}}% Success Rate
                    </span>
                    <span class="text-sm text-gray-600">{{.Environment}}</span>
                    
                    <!-- Theme Toggle -->
                    <button id="themeToggle" class="p-2 rounded-lg border border-gray-300 hover:bg-gray-50 transition-colors" title="Toggle Dark Mode">
                        <svg class="h-5 w-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"/>
                        </svg>
                    </button>
                    
                    <!-- Export Button -->
                    <button id="exportBtn" class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium">
                        <svg class="h-4 w-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                        </svg>
                        Export
                    </button>
                </div>
            </div>
        </div>
    </header>

    <!-- Main Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        
        {{if .AIInsights}}
        {{if .AIInsights.ExecutiveSummary}}
        <!-- Executive Insights (Intelligent Analysis) -->
        <section class="mb-8">
            <div class="bg-gradient-to-r from-indigo-50 via-purple-50 to-pink-50 border border-indigo-200 rounded-lg shadow-md overflow-hidden">
                <!-- Header -->
                <div class="bg-gradient-to-r from-indigo-500 to-purple-600 px-6 py-4">
                    <div class="flex items-center">
                        <svg class="h-6 w-6 text-white mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
                        </svg>
                        <div>
                            <h2 class="text-xl font-bold text-white">� Executive Insights</h2>
                            <p class="text-indigo-100 text-sm mt-1">Pattern-based intelligent analysis of your test execution</p>
                        </div>
                    </div>
                </div>

                <!-- Health Status Banner -->
                <div class="px-6 py-4 {{if eq .AIInsights.ExecutiveSummary.HealthStatus "Excellent"}}bg-green-100 border-b border-green-200{{else if eq .AIInsights.ExecutiveSummary.HealthStatus "Good"}}bg-blue-100 border-b border-blue-200{{else if eq .AIInsights.ExecutiveSummary.HealthStatus "Fair"}}bg-yellow-100 border-b border-yellow-200{{else}}bg-red-100 border-b border-red-200{{end}}">
                    <div class="flex items-center justify-between">
                        <div class="flex items-center">
                            <div class="{{if eq .AIInsights.ExecutiveSummary.HealthStatus "Excellent"}}bg-green-500{{else if eq .AIInsights.ExecutiveSummary.HealthStatus "Good"}}bg-blue-500{{else if eq .AIInsights.ExecutiveSummary.HealthStatus "Fair"}}bg-yellow-500{{else}}bg-red-500{{end}} rounded-full p-2 mr-3">
                                {{if eq .AIInsights.ExecutiveSummary.HealthStatus "Excellent"}}
                                <svg class="h-6 w-6 text-white" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
                                </svg>
                                {{else if eq .AIInsights.ExecutiveSummary.HealthStatus "Good"}}
                                <svg class="h-6 w-6 text-white" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
                                </svg>
                                {{else if eq .AIInsights.ExecutiveSummary.HealthStatus "Fair"}}
                                <svg class="h-6 w-6 text-white" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
                                </svg>
                                {{else}}
                                <svg class="h-6 w-6 text-white" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/>
                                </svg>
                                {{end}}
                            </div>
                            <div>
                                <h3 class="text-lg font-bold {{if eq .AIInsights.ExecutiveSummary.HealthStatus "Excellent"}}text-green-900{{else if eq .AIInsights.ExecutiveSummary.HealthStatus "Good"}}text-blue-900{{else if eq .AIInsights.ExecutiveSummary.HealthStatus "Fair"}}text-yellow-900{{else}}text-red-900{{end}}">
                                    Test Suite Health: {{.AIInsights.ExecutiveSummary.HealthStatus}}
                                </h3>
                                <p class="text-sm {{if eq .AIInsights.ExecutiveSummary.HealthStatus "Excellent"}}text-green-700{{else if eq .AIInsights.ExecutiveSummary.HealthStatus "Good"}}text-blue-700{{else if eq .AIInsights.ExecutiveSummary.HealthStatus "Fair"}}text-yellow-700{{else}}text-red-700{{end}}">
                                    {{.AIInsights.ExecutiveSummary.TrendIndicator}}
                                </p>
                            </div>
                        </div>
                        <div class="text-right">
                            <span class="px-3 py-1 rounded-full text-xs font-semibold {{if eq .AIInsights.ExecutiveSummary.HealthStatus "Excellent"}}bg-green-200 text-green-800{{else if eq .AIInsights.ExecutiveSummary.HealthStatus "Good"}}bg-blue-200 text-blue-800{{else if eq .AIInsights.ExecutiveSummary.HealthStatus "Fair"}}bg-yellow-200 text-yellow-800{{else}}bg-red-200 text-red-800{{end}}">
                                {{formatSuccessRate .SuccessRate}}% Success Rate
                            </span>
                        </div>
                    </div>
                </div>

                <!-- Content -->
                <div class="px-6 py-6">
                    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                        <!-- Key Insights -->
                        <div>
                            <h4 class="text-sm font-semibold text-gray-900 mb-3 flex items-center">
                                <svg class="h-5 w-5 text-indigo-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                                </svg>
                                Key Insights
                            </h4>
                            <div class="space-y-2">
                                {{range .AIInsights.ExecutiveSummary.KeyInsights}}
                                <div class="flex items-start bg-white rounded-lg p-3 border border-gray-200">
                                    <span class="text-sm text-gray-700">{{.}}</span>
                                </div>
                                {{end}}
                            </div>
                        </div>

                        <!-- Critical Issues & Recommendation -->
                        <div>
                            {{if .AIInsights.ExecutiveSummary.CriticalIssues}}
                            <h4 class="text-sm font-semibold text-red-900 mb-3 flex items-center">
                                <svg class="h-5 w-5 text-red-500 mr-2" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
                                </svg>
                                Critical Issues
                            </h4>
                            <div class="space-y-2 mb-4">
                                {{range .AIInsights.ExecutiveSummary.CriticalIssues}}
                                <div class="flex items-start bg-red-50 rounded-lg p-3 border-l-4 border-red-500">
                                    <span class="text-sm text-red-800">{{.}}</span>
                                </div>
                                {{end}}
                            </div>
                            {{end}}

                            <!-- Recommendation -->
                            <div class="bg-gradient-to-br from-indigo-50 to-purple-50 rounded-lg p-4 border border-indigo-200">
                                <h4 class="text-sm font-semibold text-indigo-900 mb-2 flex items-center">
                                    <svg class="h-5 w-5 text-indigo-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
                                    </svg>
                                    💡 Recommendation
                                </h4>
                                <p class="text-sm text-indigo-800">{{.AIInsights.ExecutiveSummary.Recommendation}}</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </section>
        {{end}}
        {{end}}
        
        <!-- Executive Summary -->
        <section class="mb-8">
            <h2 class="text-lg font-semibold text-gray-900 mb-4">Executive Summary</h2>
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                <!-- Total Scenarios -->
                <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 hover:shadow-md transition-shadow">
                    <div class="flex items-center justify-between">
                        <div class="flex-1">
                            <p class="text-sm font-medium text-gray-600">Total Scenarios</p>
                            <p class="text-3xl font-bold text-gray-900 mt-2">{{.TotalScenariosCount}}</p>
                            <p class="text-xs text-gray-500 mt-1">{{.TotalSpecsCount}} specifications</p>
                        </div>
                        <div class="bg-blue-100 rounded-full p-3">
                            <svg class="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"></path>
                            </svg>
                        </div>
                    </div>
                </div>

                <!-- Passed -->
                <div class="bg-white rounded-lg shadow-sm border border-green-200 p-6 hover:shadow-md transition-shadow">
                    <div class="flex items-center justify-between">
                        <div class="flex-1">
                            <p class="text-sm font-medium text-gray-600">Passed</p>
                            <p class="text-3xl font-bold text-green-600 mt-2">{{.PassedScenariosCount}}</p>
                            <p class="text-xs text-green-600 mt-1">✓ All tests successful</p>
                        </div>
                        <div class="bg-green-100 rounded-full p-3">
                            <svg class="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                            </svg>
                        </div>
                    </div>
                </div>

                <!-- Failed -->
                <div class="bg-white rounded-lg shadow-sm border {{if gt .FailedScenariosCount 0}}border-red-300{{else}}border-gray-200{{end}} p-6 hover:shadow-md transition-shadow">
                    <div class="flex items-center justify-between">
                        <div class="flex-1">
                            <p class="text-sm font-medium text-gray-600">Failed</p>
                            <p class="text-3xl font-bold text-red-600 mt-2">{{.FailedScenariosCount}}</p>
                            {{if gt .FailedScenariosCount 0}}
                            <p class="text-xs text-red-600 mt-1">⚠ Requires attention</p>
                            {{else}}
                            <p class="text-xs text-gray-500 mt-1">No failures</p>
                            {{end}}
                        </div>
                        <div class="bg-red-100 rounded-full p-3">
                            <svg class="w-6 h-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                            </svg>
                        </div>
                    </div>
                </div>

                <!-- Duration -->
                <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6 hover:shadow-md transition-shadow">
                    <div class="flex items-center justify-between">
                        <div class="flex-1">
                            <p class="text-sm font-medium text-gray-600">Duration</p>
                            <p class="text-3xl font-bold text-gray-900 mt-2">{{formatDuration .ExecutionTime}}</p>
                            {{if gt .SkippedScenariosCount 0}}
                            <p class="text-xs text-gray-500 mt-1">{{.SkippedScenariosCount}} skipped</p>
                            {{else}}
                            <p class="text-xs text-gray-500 mt-1">Total execution time</p>
                            {{end}}
                        </div>
                        <div class="bg-purple-100 rounded-full p-3">
                            <svg class="w-6 h-6 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                            </svg>
                        </div>
                    </div>
                </div>
            </div>
        </section>

        {{if .Trends}}
        {{if .Trends.HistoricalRuns}}
        <!-- Trends Section -->
        <section class="mb-8">
            <h2 class="text-lg font-semibold text-gray-900 mb-4">Historical Trends</h2>
            <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div>
                        <h3 class="text-sm font-medium text-gray-700 mb-4">Success Rate Trend</h3>
                        <div style="position: relative; height: 250px; width: 100%;">
                            <canvas id="successRateChart"></canvas>
                        </div>
                    </div>
                    <div>
                        <h3 class="text-sm font-medium text-gray-700 mb-4">Execution Time Trend</h3>
                        <div style="position: relative; height: 250px; width: 100%;">
                            <canvas id="executionTimeChart"></canvas>
                        </div>
                    </div>
                </div>
            </div>
        </section>
        {{end}}
        {{end}}

        {{if .FlakyTests}}
        {{if gt (len .FlakyTests) 0}}
        <!-- Flaky Tests Warning -->
        <section class="mb-8">
            <div class="bg-gradient-to-r from-yellow-50 to-orange-50 border border-yellow-300 rounded-lg shadow-sm overflow-hidden">
                <!-- Header -->
                <div class="bg-gradient-to-r from-yellow-400 to-orange-400 px-6 py-4">
                    <div class="flex items-center">
                        <svg class="h-6 w-6 text-white mr-3" fill="currentColor" viewBox="0 0 20 20">
                            <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
                        </svg>
                        <div>
                            <h2 class="text-xl font-bold text-white">⚠️ Flaky Test Detection</h2>
                            <p class="text-yellow-50 text-sm mt-1">{{len .FlakyTests}} test(s) showing inconsistent behavior</p>
                        </div>
                    </div>
                </div>
                
                <!-- Alert Message -->
                <div class="px-6 py-4 bg-yellow-100 border-b border-yellow-200">
                    <div class="flex items-start">
                        <svg class="h-5 w-5 text-yellow-600 mt-0.5 mr-2 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                            <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd"/>
                        </svg>
                        <div class="text-sm text-yellow-800">
                            <p class="font-medium">Tests with unstable results detected</p>
                            <p class="mt-1">These scenarios pass and fail intermittently. Investigate timing issues, race conditions, or external dependencies.</p>
                        </div>
                    </div>
                </div>

                <!-- Flaky Tests List -->
                <div class="px-6 py-4">
                    <div class="space-y-4">
                        {{range .FlakyTests}}
                        <div class="bg-white border-l-4 border-yellow-500 rounded-r-lg shadow-sm hover:shadow-md transition-shadow">
                            <div class="p-4">
                                <!-- Test Header -->
                                <div class="flex items-start justify-between mb-3">
                                    <div class="flex-1">
                                        <div class="flex items-center mb-2">
                                            <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 mr-2">
                                                🔄 FLAKY
                                            </span>
                                            {{if ge .FlakyScore 0.7}}
                                            <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-100 text-red-700">
                                                HIGH RISK
                                            </span>
                                            {{else if ge .FlakyScore 0.4}}
                                            <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-orange-100 text-orange-700">
                                                MODERATE
                                            </span>
                                            {{else}}
                                            <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-700">
                                                LOW RISK
                                            </span>
                                            {{end}}
                                        </div>
                                        <h4 class="text-base font-semibold text-gray-900 mb-1">{{.ScenarioName}}</h4>
                                        <p class="text-sm text-gray-600">📋 Specification: <span class="font-medium">{{.SpecName}}</span></p>
                                    </div>
                                </div>

                                <!-- Metrics Grid -->
                                <div class="grid grid-cols-1 md:grid-cols-3 gap-3 mb-3">
                                    <!-- Flaky Score -->
                                    <div class="bg-gradient-to-br from-yellow-50 to-yellow-100 rounded-lg p-3 border border-yellow-200">
                                        <div class="flex items-center justify-between">
                                            <div>
                                                <p class="text-xs text-yellow-700 font-medium">Flaky Score</p>
                                                <p class="text-2xl font-bold text-yellow-900">{{printf "%.2f" .FlakyScore}}</p>
                                                <p class="text-xs text-yellow-600 mt-1">
                                                    {{if ge .FlakyScore 0.7}}Very unstable{{else if ge .FlakyScore 0.4}}Moderately unstable{{else}}Slightly unstable{{end}}
                                                </p>
                                            </div>
                                            <svg class="h-8 w-8 text-yellow-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/>
                                            </svg>
                                        </div>
                                    </div>

                                    <!-- Failure Rate -->
                                    <div class="bg-gradient-to-br from-orange-50 to-orange-100 rounded-lg p-3 border border-orange-200">
                                        <div class="flex items-center justify-between">
                                            <div>
                                                <p class="text-xs text-orange-700 font-medium">Failure Rate</p>
                                                <p class="text-2xl font-bold text-orange-900">{{printf "%.1f" .FailureRate}}%</p>
                                                <p class="text-xs text-orange-600 mt-1">
                                                    {{if ge .FailureRate 50.0}}High failure rate{{else if ge .FailureRate 25.0}}Moderate failures{{else}}Low failure rate{{end}}
                                                </p>
                                            </div>
                                            <svg class="h-8 w-8 text-orange-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                                            </svg>
                                        </div>
                                    </div>

                                    <!-- Test Runs -->
                                    <div class="bg-gradient-to-br from-blue-50 to-blue-100 rounded-lg p-3 border border-blue-200">
                                        <div class="flex items-center justify-between">
                                            <div>
                                                <p class="text-xs text-blue-700 font-medium">Total Runs</p>
                                                <p class="text-2xl font-bold text-blue-900">{{.Occurrences}}</p>
                                                <p class="text-xs text-blue-600 mt-1">Last 30 days</p>
                                            </div>
                                            <svg class="h-8 w-8 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/>
                                            </svg>
                                        </div>
                                    </div>
                                </div>

                                <!-- Recommendations -->
                                <div class="bg-blue-50 border border-blue-200 rounded-lg p-3">
                                    <p class="text-xs font-semibold text-blue-900 mb-2">💡 Recommended Actions:</p>
                                    <ul class="text-xs text-blue-800 space-y-1 ml-4">
                                        {{if ge .FlakyScore 0.7}}
                                        <li class="list-disc">⚠️ High priority: Investigate immediately to prevent CI/CD disruption</li>
                                        <li class="list-disc">Check for race conditions, timing issues, or shared state problems</li>
                                        <li class="list-disc">Consider quarantining this test until stabilized</li>
                                        {{else if ge .FlakyScore 0.4}}
                                        <li class="list-disc">Review test for external dependencies (network, database, file system)</li>
                                        <li class="list-disc">Add explicit waits or synchronization mechanisms</li>
                                        <li class="list-disc">Verify test isolation and proper cleanup</li>
                                        {{else}}
                                        <li class="list-disc">Monitor trend - may indicate emerging stability issues</li>
                                        <li class="list-disc">Review recent code changes that might affect this scenario</li>
                                        <li class="list-disc">Check test environment consistency</li>
                                        {{end}}
                                    </ul>
                                </div>
                            </div>
                        </div>
                        {{end}}
                    </div>
                </div>

                <!-- Footer with Best Practices -->
                <div class="bg-gray-50 px-6 py-4 border-t border-gray-200">
                    <p class="text-xs text-gray-600">
                        <span class="font-semibold">💡 Pro Tip:</span> Flaky tests reduce confidence in your test suite. 
                        Address tests with scores above 0.4 to maintain reliable CI/CD pipelines.
                    </p>
                </div>
            </div>
        </section>
        {{end}}
        {{end}}

        {{if .AIInsights}}
        {{if .AIInsights.FailureGroups}}
        {{if gt (len .AIInsights.FailureGroups) 0}}
        <!-- AI Failure Analysis -->
        <section class="mb-8">
            <div class="bg-gradient-to-r from-red-50 to-orange-50 border border-red-200 rounded-lg shadow-md overflow-hidden">
                <!-- Header -->
                <div class="bg-gradient-to-r from-red-500 to-orange-500 px-6 py-4">
                    <div class="flex items-center">
                        <svg class="h-6 w-6 text-white mr-3" fill="currentColor" viewBox="0 0 20 20">
                            <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
                        </svg>
                        <div>
                            <h2 class="text-xl font-bold text-white">🔍 Intelligent Failure Analysis</h2>
                            <p class="text-red-50 text-sm mt-1">{{len .AIInsights.FailureGroups}} unique failure pattern(s) detected</p>
                        </div>
                    </div>
                </div>

                <!-- Failure Groups -->
                <div class="px-6 py-4">
                    <div class="space-y-4">
                        {{range .AIInsights.FailureGroups}}
                        <div class="bg-white rounded-lg border-l-4 {{if eq .Severity "critical"}}border-red-600{{else if eq .Severity "high"}}border-orange-500{{else}}border-yellow-500{{end}} shadow-sm hover:shadow-md transition-shadow">
                            <div class="p-5">
                                <!-- Header -->
                                <div class="flex items-start justify-between mb-4">
                                    <div class="flex-1">
                                        <div class="flex items-center gap-2 mb-2">
                                            <!-- Severity Badge -->
                                            <span class="px-2.5 py-1 rounded text-xs font-bold {{if eq .Severity "critical"}}bg-red-100 text-red-800{{else if eq .Severity "high"}}bg-orange-100 text-orange-800{{else}}bg-yellow-100 text-yellow-800{{end}}">
                                                {{if eq .Severity "critical"}}🔴 CRITICAL{{else if eq .Severity "high"}}🟠 HIGH{{else}}🟡 MEDIUM{{end}}
                                            </span>
                                            
                                            <!-- Error Type Badge -->
                                            <span class="px-2.5 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                                                {{.ErrorType}}
                                            </span>
                                            
                                            <!-- Count Badge -->
                                            <span class="px-2.5 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                                                {{.Count}} occurrence(s)
                                            </span>
                                        </div>
                                        <h3 class="text-base font-semibold text-gray-900">{{.RootCause}}</h3>
                                    </div>
                                </div>

                                <!-- Impact -->
                                <div class="mb-4">
                                    <h4 class="text-sm font-semibold text-gray-700 mb-2">📊 Impact:</h4>
                                    <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                                        <div class="bg-gray-50 rounded p-3 border border-gray-200">
                                            <p class="text-xs text-gray-600 mb-1">Affected Scenarios</p>
                                            <div class="space-y-1">
                                                {{range .AffectedScenarios}}
                                                <p class="text-sm text-gray-800">• {{.}}</p>
                                                {{end}}
                                            </div>
                                        </div>
                                        <div class="bg-gray-50 rounded p-3 border border-gray-200">
                                            <p class="text-xs text-gray-600 mb-1">Affected Specifications</p>
                                            <div class="space-y-1">
                                                {{range .AffectedSpecs}}
                                                <p class="text-sm text-gray-800">• {{.}}</p>
                                                {{end}}
                                            </div>
                                        </div>
                                    </div>
                                </div>

                                <!-- Suggested Fix -->
                                <div class="bg-gradient-to-r from-blue-50 to-indigo-50 rounded-lg p-4 border border-blue-200">
                                    <h4 class="text-sm font-semibold text-blue-900 mb-2 flex items-center">
                                        <svg class="h-4 w-4 text-blue-600 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/>
                                        </svg>
                                        💡 Suggested Fix
                                    </h4>
                                    <p class="text-sm text-blue-800">{{.SuggestedFix}}</p>
                                </div>
                            </div>
                        </div>
                        {{end}}
                    </div>
                </div>

                <!-- Footer -->
                <div class="bg-gray-50 px-6 py-4 border-t border-gray-200">
                    <p class="text-xs text-gray-600">
                        <span class="font-semibold">🤖 AI Analysis:</span> Failures have been automatically grouped by error signature. 
                        Address critical and high-severity issues first to maximize test stability.
                    </p>
                </div>
            </div>
        </section>
        {{end}}
        {{end}}
        {{end}}

        {{if .Analytics}}
        {{if .Analytics.SlowestSpecs}}
        {{if gt (len .Analytics.SlowestSpecs) 0}}
        <!-- Performance Dashboard -->
        <section class="mb-8">
            <h2 class="text-lg font-semibold text-gray-900 mb-4">
                <span class="inline-flex items-center gap-2">
                    <svg class="w-5 h-5 text-purple-600" fill="currentColor" viewBox="0 0 20 20">
                        <path d="M2 11a1 1 0 011-1h2a1 1 0 011 1v5a1 1 0 01-1 1H3a1 1 0 01-1-1v-5zM8 7a1 1 0 011-1h2a1 1 0 011 1v9a1 1 0 01-1 1H9a1 1 0 01-1-1V7zM14 4a1 1 0 011-1h2a1 1 0 011 1v12a1 1 0 01-1 1h-2a1 1 0 01-1-1V4z"/>
                    </svg>
                    Performance Dashboard
                </span>
            </h2>
            
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
                <!-- Runtime Distribution Chart -->
                <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
                    <h3 class="text-sm font-semibold text-gray-700 mb-4">Scenario Runtime Distribution</h3>
                    <div style="position: relative; height: 280px; width: 100%;">
                        <canvas id="runtimeDistributionChart"></canvas>
                    </div>
                </div>

                <!-- Top Slowest Scenarios -->
                <div class="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
                    <h3 class="text-sm font-semibold text-gray-700 mb-4">Top 5 Slowest Specifications</h3>
                    <div class="space-y-3">
                        {{range .Analytics.SlowestSpecs}}
                        <div class="flex items-center gap-3">
                            <div class="flex-shrink-0 w-8 h-8 rounded-full bg-orange-100 flex items-center justify-center">
                                <svg class="w-4 h-4 text-orange-600" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z" clip-rule="evenodd"/>
                                </svg>
                            </div>
                            <div class="flex-1 min-w-0">
                                <p class="text-sm font-medium text-gray-900 truncate">{{.SpecName}}</p>
                                <p class="text-xs text-gray-500">{{.ScenarioCount}} scenario(s)</p>
                            </div>
                            <div class="flex-shrink-0">
                                <span class="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-semibold bg-orange-100 text-orange-800">
                                    {{formatDuration .ExecutionTime}}
                                </span>
                            </div>
                        </div>
                        {{end}}
                    </div>
                </div>
            </div>

            <!-- Performance Recommendations -->
            <div class="bg-gradient-to-r from-purple-50 to-blue-50 rounded-lg border border-purple-200 p-6">
                <div class="flex items-start gap-3">
                    <div class="flex-shrink-0">
                        <svg class="w-6 h-6 text-purple-600" fill="currentColor" viewBox="0 0 20 20">
                            <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd"/>
                        </svg>
                    </div>
                    <div class="flex-1">
                        <h3 class="text-sm font-semibold text-purple-900 mb-2">Performance Recommendations</h3>
                        <ul class="space-y-2 text-sm text-purple-800">
                            <li class="flex items-start gap-2">
                                <svg class="w-4 h-4 text-purple-600 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
                                </svg>
                                <span><strong>Average scenario time:</strong> {{formatDuration .Analytics.AverageScenarioTime}} - {{if lt .Analytics.AverageScenarioTime.Milliseconds 500}}Excellent performance!{{else if lt .Analytics.AverageScenarioTime.Milliseconds 1000}}Good performance{{else}}Consider optimizing slow scenarios{{end}}</span>
                            </li>
                            {{if gt (len .Analytics.SlowestSpecs) 0}}
                            {{$slowest := index .Analytics.SlowestSpecs 0}}
                            <li class="flex items-start gap-2">
                                <svg class="w-4 h-4 text-purple-600 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
                                </svg>
                                <span><strong>Focus area:</strong> "{{$slowest.SpecName}}" takes {{formatDuration $slowest.ExecutionTime}} - review for optimization opportunities</span>
                            </li>
                            {{end}}
                            <li class="flex items-start gap-2">
                                <svg class="w-4 h-4 text-purple-600 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                                    <path fill-rule="evenodd" d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" clip-rule="evenodd"/>
                                </svg>
                                <span><strong>Total execution time:</strong> {{formatDuration .Analytics.TotalExecutionTime}} across {{.TotalScenariosCount}} scenarios</span>
                            </li>
                        </ul>
                    </div>
                </div>
            </div>
        </section>
        {{end}}
        {{end}}
        {{end}}

        <!-- Test Specifications -->
        <section class="mb-8">
            <h2 class="text-lg font-semibold text-gray-900 mb-4">Test Specifications</h2>
            <div class="bg-white rounded-lg shadow-sm border border-gray-200">
                <div class="divide-y divide-gray-200">
                    {{range .SpecResults}}
                    <div class="p-6 hover:bg-gray-50 transition-colors" x-data="{ expanded: false }">
                        <div class="flex items-center justify-between cursor-pointer" @click="expanded = !expanded">
                            <div class="flex items-center gap-4 flex-1">
                                <!-- Status Icon with Badge -->
                                <div class="flex-shrink-0">
                                    {{if .Failed}}
                                    <div class="relative">
                                        <span class="inline-flex items-center justify-center w-10 h-10 rounded-full bg-red-100 border-2 border-red-200">
                                            <svg class="w-5 h-5 text-red-600" fill="currentColor" viewBox="0 0 20 20">
                                                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/>
                                            </svg>
                                        </span>
                                        <span class="absolute -top-1 -right-1 bg-red-600 text-white text-xs font-bold rounded-full w-5 h-5 flex items-center justify-center">!</span>
                                    </div>
                                    {{else if .Skipped}}
                                    <span class="inline-flex items-center justify-center w-10 h-10 rounded-full bg-gray-100 border-2 border-gray-200">
                                        <svg class="w-5 h-5 text-gray-600" fill="currentColor" viewBox="0 0 20 20">
                                            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM7 9a1 1 0 000 2h6a1 1 0 100-2H7z" clip-rule="evenodd"/>
                                        </svg>
                                    </span>
                                    {{else}}
                                    <span class="inline-flex items-center justify-center w-10 h-10 rounded-full bg-green-100 border-2 border-green-200">
                                        <svg class="w-5 h-5 text-green-600" fill="currentColor" viewBox="0 0 20 20">
                                            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
                                        </svg>
                                    </span>
                                    {{end}}
                                </div>
                                <!-- Spec Details -->
                                <div class="flex-1">
                                    <div class="flex items-center gap-2">
                                        <h3 class="text-base font-semibold text-gray-900">{{.SpecHeading}}</h3>
                                        {{if .Failed}}
                                        <span class="px-2 py-0.5 bg-red-100 text-red-700 text-xs font-medium rounded">FAILED</span>
                                        {{else if .Skipped}}
                                        <span class="px-2 py-0.5 bg-gray-100 text-gray-700 text-xs font-medium rounded">SKIPPED</span>
                                        {{else}}
                                        <span class="px-2 py-0.5 bg-green-100 text-green-700 text-xs font-medium rounded">PASSED</span>
                                        {{end}}
                                    </div>
                                    <div class="mt-1 flex items-center gap-4 text-sm text-gray-500">
                                        <span>{{len .Scenarios}} scenario{{if ne (len .Scenarios) 1}}s{{end}}</span>
                                        <span>•</span>
                                        <span>{{formatDuration .ExecutionTime}}</span>
                                        {{if .Tags}}
                                        <span>•</span>
                                        <div class="flex gap-1 flex-wrap">
                                            {{range .Tags}}
                                            <span class="px-2 py-0.5 bg-blue-50 text-blue-700 rounded text-xs">{{.}}</span>
                                            {{end}}
                                        </div>
                                        {{end}}
                                    </div>
                                </div>
                            </div>
                            <!-- Results Summary -->
                            <div class="flex items-center gap-6">
                                <div class="text-right">
                                    <div class="flex items-center gap-3 text-sm font-medium">
                                        {{if gt (getPassedScenariosCount .) 0}}
                                        <div class="flex items-center gap-1 text-green-600">
                                            <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                                                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
                                            </svg>
                                            <span>{{getPassedScenariosCount .}}</span>
                                        </div>
                                        {{end}}
                                        {{if gt (getFailedScenariosCount .) 0}}
                                        <div class="flex items-center gap-1 text-red-600">
                                            <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                                                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/>
                                            </svg>
                                            <span>{{getFailedScenariosCount .}}</span>
                                        </div>
                                        {{end}}
                                        {{if gt (getSkippedScenariosCount .) 0}}
                                        <div class="flex items-center gap-1 text-gray-500">
                                            <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                                                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM7 9a1 1 0 000 2h6a1 1 0 100-2H7z" clip-rule="evenodd"/>
                                            </svg>
                                            <span>{{getSkippedScenariosCount .}}</span>
                                        </div>
                                        {{end}}
                                    </div>
                                </div>
                                <svg class="w-5 h-5 text-gray-400 transition-transform" :class="{ 'rotate-180': expanded }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
                                </svg>
                            </div>
                        </div>

                        <!-- Expanded Scenarios -->
                        <div x-show="expanded" x-collapse class="mt-4 pl-14">
                            <div class="space-y-3">
                                {{range .Scenarios}}
                                <div class="border-l-4 {{if .Failed}}border-red-500 bg-red-50{{else if .Skipped}}border-gray-300 bg-gray-50{{else}}border-green-500 bg-green-50{{end}} pl-4 py-3 rounded-r">
                                    <div class="flex items-start justify-between gap-4">
                                        <div class="flex-1">
                                            <div class="flex items-center gap-2">
                                                {{if .Failed}}
                                                <svg class="w-4 h-4 text-red-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                                                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/>
                                                </svg>
                                                <h4 class="text-sm font-semibold text-red-900">{{.ScenarioHeading}}</h4>
                                                <span class="px-2 py-0.5 bg-red-200 text-red-800 text-xs font-bold rounded">FAILED</span>
                                                {{else if .Skipped}}
                                                <svg class="w-4 h-4 text-gray-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                                                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM7 9a1 1 0 000 2h6a1 1 0 100-2H7z" clip-rule="evenodd"/>
                                                </svg>
                                                <h4 class="text-sm font-semibold text-gray-900">{{.ScenarioHeading}}</h4>
                                                <span class="px-2 py-0.5 bg-gray-200 text-gray-700 text-xs font-bold rounded">SKIPPED</span>
                                                {{else}}
                                                <svg class="w-4 h-4 text-green-600 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                                                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
                                                </svg>
                                                <h4 class="text-sm font-semibold text-green-900">{{.ScenarioHeading}}</h4>
                                                <span class="px-2 py-0.5 bg-green-200 text-green-800 text-xs font-bold rounded">PASSED</span>
                                                {{end}}
                                            </div>
                                            {{if .Failed}}
                                            <div class="mt-3 bg-white border border-red-300 rounded p-3">
                                                <p class="text-xs font-semibold text-red-900 mb-2">❌ Failure Details:</p>
                                                {{range .Steps}}
                                                {{if .Failed}}
                                                <div class="space-y-1">
                                                    <p class="text-xs font-medium text-gray-900">Step: <span class="text-red-700">{{.StepText}}</span></p>
                                                    {{if .ErrorMessage}}
                                                    <p class="text-xs text-red-600 bg-red-50 p-2 rounded border border-red-200">
                                                        <span class="font-semibold">Error:</span> {{.ErrorMessage}}
                                                    </p>
                                                    {{end}}
                                                    {{if .StackTrace}}
                                                    <details class="mt-2">
                                                        <summary class="text-xs text-gray-600 cursor-pointer hover:text-gray-900">View Stack Trace</summary>
                                                        <pre class="text-xs text-gray-700 bg-gray-50 p-2 rounded mt-1 overflow-x-auto">{{.StackTrace}}</pre>
                                                    </details>
                                                    {{end}}
                                                </div>
                                                {{end}}
                                                {{end}}
                                            </div>
                                            {{else if .Skipped}}
                                            <p class="text-xs text-gray-600 mt-2">⊝ This scenario was skipped during execution</p>
                                            {{end}}
                                        </div>
                                        <div class="flex items-center gap-2 flex-shrink-0">
                                            <svg class="w-3 h-3 text-gray-400" fill="currentColor" viewBox="0 0 20 20">
                                                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z" clip-rule="evenodd"/>
                                            </svg>
                                            <span class="text-xs font-medium text-gray-600">{{formatDuration .ExecutionTime}}</span>
                                        </div>
                                    </div>
                                </div>
                                {{end}}
                            </div>
                        </div>
                    </div>
                    {{end}}
                </div>
            </div>
        </section>

    </main>

    <!-- Footer -->
    <footer class="bg-white border-t border-gray-200 mt-12">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
            <p class="text-center text-sm text-gray-500">
                Generated by Enhanced Gauge HTML Report | {{formatTimestamp .Timestamp}}
            </p>
        </div>
    </footer>

    <script>
        {{if .Trends}}
        {{if .Trends.HistoricalRuns}}
        // Enhanced Success Rate Trend Chart with Failure Overlay
        const successRateCtx = document.getElementById('successRateChart');
        if (successRateCtx) {
            const trendData = [
                {{range .Trends.HistoricalRuns}}{
                    timestamp: '{{.Timestamp.Format "Jan 2, 15:04"}}',
                    successRate: {{.SuccessRate}},
                    total: {{.PassedCount}} + {{.FailedCount}} + {{.SkippedCount}},
                    passed: {{.PassedCount}},
                    failed: {{.FailedCount}},
                    duration: {{.ExecutionTime.Seconds}}
                },{{end}}
            ];

            new Chart(successRateCtx, {
                type: 'line',
                data: {
                    labels: trendData.map(d => d.timestamp),
                    datasets: [
                        {
                            label: 'Success Rate (%)',
                            data: trendData.map(d => d.successRate),
                            borderColor: 'rgb(16, 185, 129)',
                            backgroundColor: 'rgba(16, 185, 129, 0.1)',
                            tension: 0.4,
                            fill: true,
                            yAxisID: 'y',
                            borderWidth: 3,
                            pointRadius: 5,
                            pointHoverRadius: 7,
                            pointBackgroundColor: 'rgb(16, 185, 129)',
                            pointBorderColor: '#fff',
                            pointBorderWidth: 2
                        },
                        {
                            label: 'Failed Scenarios',
                            data: trendData.map(d => d.failed),
                            borderColor: 'rgb(239, 68, 68)',
                            backgroundColor: 'rgba(239, 68, 68, 0.1)',
                            tension: 0.4,
                            fill: true,
                            yAxisID: 'y1',
                            borderWidth: 2,
                            borderDash: [5, 5],
                            pointRadius: 4,
                            pointHoverRadius: 6,
                            pointBackgroundColor: 'rgb(239, 68, 68)',
                            pointBorderColor: '#fff',
                            pointBorderWidth: 2
                        }
                    ]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    interaction: {
                        mode: 'index',
                        intersect: false,
                    },
                    plugins: {
                        title: {
                            display: true,
                            text: 'Success Rate & Failures Trend',
                            font: { size: 14, weight: '600' },
                            color: '#374151'
                        },
                        legend: {
                            display: true,
                            position: 'bottom',
                            labels: {
                                usePointStyle: true,
                                padding: 15,
                                font: { size: 11 }
                            }
                        },
                        tooltip: {
                            backgroundColor: 'rgba(0, 0, 0, 0.8)',
                            titleFont: { size: 13, weight: 'bold' },
                            bodyFont: { size: 12 },
                            bodySpacing: 6,
                            padding: 12,
                            cornerRadius: 8,
                            callbacks: {
                                title: function(context) {
                                    return trendData[context[0].dataIndex].timestamp;
                                },
                                afterLabel: function(context) {
                                    const data = trendData[context.dataIndex];
                                    if (context.datasetIndex === 0) {
                                        return [
                                            'Total: ' + data.total + ' scenarios',
                                            'Passed: ' + data.passed,
                                            'Failed: ' + data.failed,
                                            'Duration: ' + data.duration.toFixed(1) + 's'
                                        ];
                                    }
                                    return null;
                                }
                            }
                        }
                    },
                    scales: {
                        y: {
                            type: 'linear',
                            display: true,
                            position: 'left',
                            min: 0,
                            max: 100,
                            title: {
                                display: true,
                                text: 'Success Rate (%)',
                                font: { size: 11, weight: '600' },
                                color: 'rgb(16, 185, 129)'
                            },
                            ticks: {
                                callback: function(value) {
                                    return value + '%';
                                },
                                font: { size: 10 }
                            },
                            grid: {
                                color: 'rgba(0, 0, 0, 0.05)'
                            }
                        },
                        y1: {
                            type: 'linear',
                            display: true,
                            position: 'right',
                            min: 0,
                            title: {
                                display: true,
                                text: 'Failed Count',
                                font: { size: 11, weight: '600' },
                                color: 'rgb(239, 68, 68)'
                            },
                            ticks: {
                                stepSize: 1,
                                font: { size: 10 }
                            },
                            grid: {
                                drawOnChartArea: false,
                            }
                        },
                        x: {
                            ticks: {
                                font: { size: 10 },
                                maxRotation: 45,
                                minRotation: 45
                            },
                            grid: {
                                display: false
                            }
                        }
                    }
                }
            });
        }

        // Enhanced Execution Time Trend Chart with Performance Zones
        const executionTimeCtx = document.getElementById('executionTimeChart');
        if (executionTimeCtx) {
            const durations = trendData.map(d => d.duration);
            const avgDuration = durations.reduce((a, b) => a + b, 0) / durations.length;

            new Chart(executionTimeCtx, {
                type: 'line',
                data: {
                    labels: trendData.map(d => d.timestamp),
                    datasets: [
                        {
                            label: 'Duration (seconds)',
                            data: durations,
                            borderColor: 'rgb(139, 92, 246)',
                            backgroundColor: 'rgba(139, 92, 246, 0.1)',
                            tension: 0.4,
                            fill: true,
                            borderWidth: 3,
                            pointRadius: 5,
                            pointHoverRadius: 7,
                            pointBackgroundColor: 'rgb(139, 92, 246)',
                            pointBorderColor: '#fff',
                            pointBorderWidth: 2
                        },
                        {
                            label: 'Average (' + avgDuration.toFixed(1) + 's)',
                            data: new Array(durations.length).fill(avgDuration),
                            borderColor: 'rgba(156, 163, 175, 0.6)',
                            backgroundColor: 'transparent',
                            borderWidth: 2,
                            borderDash: [10, 5],
                            pointRadius: 0,
                            pointHoverRadius: 0
                        }
                    ]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    interaction: {
                        mode: 'index',
                        intersect: false,
                    },
                    plugins: {
                        title: {
                            display: true,
                            text: 'Execution Time Trend',
                            font: { size: 14, weight: '600' },
                            color: '#374151'
                        },
                        legend: {
                            display: true,
                            position: 'bottom',
                            labels: {
                                usePointStyle: true,
                                padding: 15,
                                font: { size: 11 }
                            }
                        },
                        tooltip: {
                            backgroundColor: 'rgba(0, 0, 0, 0.8)',
                            titleFont: { size: 13, weight: 'bold' },
                            bodyFont: { size: 12 },
                            bodySpacing: 6,
                            padding: 12,
                            cornerRadius: 8,
                            callbacks: {
                                title: function(context) {
                                    return trendData[context[0].dataIndex].timestamp;
                                },
                                label: function(context) {
                                    if (context.datasetIndex === 0) {
                                        const duration = context.parsed.y;
                                        const diff = duration - avgDuration;
                                        const diffPercent = ((diff / avgDuration) * 100).toFixed(1);
                                        const arrow = diff > 0 ? '↑' : diff < 0 ? '↓' : '→';
                                        return 'Duration: ' + duration.toFixed(2) + 's (' + arrow + ' ' + Math.abs(diffPercent) + '% vs avg)';
                                    }
                                    return context.dataset.label;
                                },
                                afterLabel: function(context) {
                                    if (context.datasetIndex === 0) {
                                        const data = trendData[context.dataIndex];
                                        return [
                                            'Success Rate: ' + data.successRate.toFixed(1) + '%',
                                            'Scenarios: ' + data.total
                                        ];
                                    }
                                    return null;
                                }
                            }
                        }
                    },
                    scales: {
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: 'Duration (seconds)',
                                font: { size: 11, weight: '600' },
                                color: 'rgb(139, 92, 246)'
                            },
                            ticks: {
                                callback: function(value) {
                                    return value.toFixed(1) + 's';
                                },
                                font: { size: 10 }
                            },
                            grid: {
                                color: 'rgba(0, 0, 0, 0.05)'
                            }
                        },
                        x: {
                            ticks: {
                                font: { size: 10 },
                                maxRotation: 45,
                                minRotation: 45
                            },
                            grid: {
                                display: false
                            }
                        }
                    }
                }
            });
        }

        // Runtime Distribution Chart
        const runtimeDistCtx = document.getElementById('runtimeDistributionChart');
        if (runtimeDistCtx) {
            // Collect all scenario durations
            const scenarios = [
                {{range .SpecResults}}
                    {{range .Scenarios}}
                        {{.ExecutionTime.Milliseconds}},
                    {{end}}
                {{end}}
            ];

            // Categorize into buckets
            const buckets = {
                fast: scenarios.filter(d => d < 100).length,
                medium: scenarios.filter(d => d >= 100 && d < 500).length,
                slow: scenarios.filter(d => d >= 500 && d < 1000).length,
                verySlow: scenarios.filter(d => d >= 1000).length
            };

            new Chart(runtimeDistCtx, {
                type: 'doughnut',
                data: {
                    labels: [
                        'Fast (< 100ms)',
                        'Medium (100-500ms)',
                        'Slow (500ms-1s)',
                        'Very Slow (> 1s)'
                    ],
                    datasets: [{
                        data: [buckets.fast, buckets.medium, buckets.slow, buckets.verySlow],
                        backgroundColor: [
                            'rgba(16, 185, 129, 0.8)',
                            'rgba(59, 130, 246, 0.8)',
                            'rgba(251, 146, 60, 0.8)',
                            'rgba(239, 68, 68, 0.8)'
                        ],
                        borderColor: [
                            'rgb(16, 185, 129)',
                            'rgb(59, 130, 246)',
                            'rgb(251, 146, 60)',
                            'rgb(239, 68, 68)'
                        ],
                        borderWidth: 2
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            position: 'bottom',
                            labels: {
                                padding: 15,
                                font: { size: 11 },
                                generateLabels: function(chart) {
                                    const data = chart.data;
                                    return data.labels.map((label, i) => {
                                        const value = data.datasets[0].data[i];
                                        const total = scenarios.length;
                                        const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : 0;
                                        return {
                                            text: label + ': ' + value + ' (' + percentage + '%)',
                                            fillStyle: data.datasets[0].backgroundColor[i],
                                            hidden: false,
                                            index: i
                                        };
                                    });
                                }
                            }
                        },
                        tooltip: {
                            backgroundColor: 'rgba(0, 0, 0, 0.8)',
                            titleFont: { size: 13, weight: 'bold' },
                            bodyFont: { size: 12 },
                            padding: 12,
                            cornerRadius: 8,
                            callbacks: {
                                label: function(context) {
                                    const label = context.label || '';
                                    const value = context.parsed;
                                    const total = scenarios.length;
                                    const percentage = ((value / total) * 100).toFixed(1);
                                    return label + ': ' + value + ' scenarios (' + percentage + '%)';
                                }
                            }
                        }
                    }
                }
            });
        }
        {{end}}
        {{end}}
    </script>
</body>
</html>`
}
