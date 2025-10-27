package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/getgauge/gauge-proto/go/gauge_messages"
	"github.com/lirany1/gauge-html-report-ai/pkg/analytics"
	"github.com/lirany1/gauge-html-report-ai/pkg/config"
	"github.com/lirany1/gauge-html-report-ai/pkg/export"
	"github.com/lirany1/gauge-html-report-ai/pkg/logger"
	"github.com/lirany1/gauge-html-report-ai/pkg/models"
	"github.com/lirany1/gauge-html-report-ai/pkg/renderer"
	"github.com/lirany1/gauge-html-report-ai/pkg/themes"
	"google.golang.org/protobuf/proto"
)

// Generator handles enhanced HTML report generation
type Generator struct {
	config    *config.Config
	analytics *analytics.Engine
	renderer  *renderer.Renderer
	exporter  *export.Exporter
	themes    *themes.Manager
}

// NewGenerator creates a new enhanced report generator
func NewGenerator(cfg *config.Config) *Generator {
	return &Generator{
		config:    cfg,
		analytics: analytics.NewEngine(cfg, nil), // Pass nil for database since generator doesn't use it
		renderer:  renderer.NewRenderer(cfg),
		exporter:  export.NewExporter(cfg),
		themes:    themes.NewManager(cfg),
	}
}

// GenerateFromFile generates a report from a saved protobuf file
func (g *Generator) GenerateFromFile(inputFile, outputDir string) error {
	logger.Infof("Reading test results from %s", inputFile)

	// Read protobuf file
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Unmarshal protobuf
	protoResult := &gauge_messages.ProtoSuiteResult{}
	if err := proto.Unmarshal(data, protoResult); err != nil {
		return fmt.Errorf("failed to unmarshal proto data: %w", err)
	}

	// Transform to enhanced model
	suiteResult := g.transformProtoToEnhanced(protoResult)

	// Generate report
	return g.Generate(suiteResult, outputDir)
}

// Generate creates an enhanced HTML report from suite results
func (g *Generator) Generate(suite *models.EnhancedSuiteResult, outputDir string) error {
	startTime := time.Now()
	logger.Info("Starting enhanced report generation...")

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Run analytics if enabled
	if g.config.EnableAnalytics {
		logger.Info("Running analytics...")
		suite.Analytics = g.analytics.Analyze(suite)
	}

	// Generate historical trends if enabled
	if g.config.EnableTrends {
		logger.Info("Generating trend data...")
		suite.Trends = g.analytics.GenerateTrends(suite)
	}

	// Detect flaky tests if enabled
	if g.config.FlakyTestDetection {
		logger.Info("Detecting flaky tests...")
		suite.FlakyTests = g.analytics.DetectFlakyTests(suite)
	}

	// Copy theme assets
	logger.Info("Copying theme assets...")
	if err := g.themes.CopyAssets(g.config.ThemePath, outputDir); err != nil {
		return fmt.Errorf("failed to copy theme assets: %w", err)
	}

	// Render main report
	logger.Info("Rendering HTML report...")
	if err := g.renderMainReport(suite, outputDir); err != nil {
		return fmt.Errorf("failed to render report: %w", err)
	}

	// Render individual spec pages (parallel)
	logger.Info("Rendering specification pages...")
	if err := g.renderSpecPages(suite, outputDir); err != nil {
		return fmt.Errorf("failed to render spec pages: %w", err)
	}

	// Generate search index
	logger.Info("Generating search index...")
	if err := g.generateSearchIndex(suite, outputDir); err != nil {
		logger.Warnf("Failed to generate search index: %v", err)
	}

	// Export to additional formats
	if len(g.config.ExportFormats) > 1 {
		logger.Info("Exporting to additional formats...")
		if err := g.exportToFormats(suite, outputDir); err != nil {
			logger.Warnf("Failed to export to some formats: %v", err)
		}
	}

	// Copy screenshots
	logger.Info("Copying screenshots...")
	if err := g.copyScreenshots(suite, outputDir); err != nil {
		logger.Warnf("Failed to copy some screenshots: %v", err)
	}

	duration := time.Since(startTime)
	logger.Infof("✓ Report generated successfully in %v", duration)
	logger.Infof("Open: file://%s/index.html", outputDir)

	return nil
}

// transformProtoToEnhanced converts protobuf format to enhanced model
func (g *Generator) transformProtoToEnhanced(proto *gauge_messages.ProtoSuiteResult) *models.EnhancedSuiteResult {
	suite := &models.EnhancedSuiteResult{
		ProjectName:        proto.GetProjectName(),
		Environment:        proto.GetEnvironment(),
		Tags:               []string{proto.GetTags()}, // Tags is a string, convert to slice
		ExecutionTime:      time.Duration(proto.GetExecutionTime()) * time.Millisecond,
		Timestamp:          time.Now(), // Use current time since timestamp is string
		SuccessRate:        calculateSuccessRate(proto),
		SpecResults:        make([]*models.SpecResult, 0),
		BeforeSuiteFailure: transformHookFailure(proto.GetPreHookFailure()),
		AfterSuiteFailure:  transformHookFailure(proto.GetPostHookFailure()),
		Messages:           proto.GetPreHookMessages(),
		//nolint:staticcheck // Using deprecated Gauge proto method until framework provides alternative
		Screenshots: proto.GetPreHookScreenshots(),
	}

	// Transform spec results
	for _, protoSpec := range proto.GetSpecResults() {
		spec := g.transformSpec(protoSpec)
		suite.SpecResults = append(suite.SpecResults, spec)

		// Update counters
		if spec.Failed {
			suite.FailedSpecsCount++
		} else if spec.Skipped {
			suite.SkippedSpecsCount++
		} else {
			suite.PassedSpecsCount++
		}

		suite.TotalScenariosCount += len(spec.Scenarios)
		for _, scenario := range spec.Scenarios {
			if scenario.Failed {
				suite.FailedScenariosCount++
			} else if scenario.Skipped {
				suite.SkippedScenariosCount++
			} else {
				suite.PassedScenariosCount++
			}
		}
	}

	return suite
}

// transformSpec converts a proto spec to enhanced model
func (g *Generator) transformSpec(protoSpec *gauge_messages.ProtoSpecResult) *models.SpecResult {
	spec := &models.SpecResult{
		SpecHeading:   protoSpec.GetProtoSpec().GetSpecHeading(),
		FileName:      protoSpec.GetProtoSpec().GetFileName(),
		Tags:          protoSpec.GetProtoSpec().GetTags(),
		ExecutionTime: time.Duration(protoSpec.GetExecutionTime()) * time.Millisecond,
		Failed:        protoSpec.GetFailed(),
		Skipped:       protoSpec.GetSkipped(),
		Scenarios:     make([]*models.ScenarioResult, 0),
	}

	// Transform scenarios
	for _, protoItem := range protoSpec.GetProtoSpec().GetItems() {
		if protoItem.GetItemType() == gauge_messages.ProtoItem_Scenario {
			scenario := g.transformScenario(protoItem.GetScenario())
			spec.Scenarios = append(spec.Scenarios, scenario)
		}
	}

	return spec
}

// transformScenario converts a proto scenario to enhanced model
func (g *Generator) transformScenario(protoScenario *gauge_messages.ProtoScenario) *models.ScenarioResult {
	return &models.ScenarioResult{
		ScenarioHeading: protoScenario.GetScenarioHeading(),
		Tags:            protoScenario.GetTags(),
		ExecutionTime:   time.Duration(protoScenario.GetExecutionTime()) * time.Millisecond,
		//nolint:staticcheck // Using deprecated Gauge proto method until framework provides alternative
		Failed: protoScenario.GetFailed(),
		//nolint:staticcheck // Using deprecated Gauge proto method until framework provides alternative
		Skipped: protoScenario.GetSkipped(),
		Steps:   make([]*models.StepResult, 0),
	}
}

// renderMainReport generates the main index.html page
func (g *Generator) renderMainReport(suite *models.EnhancedSuiteResult, outputDir string) error {
	indexPath := filepath.Join(outputDir, "index.html")
	return g.renderer.RenderIndex(suite, indexPath)
}

// renderSpecPages generates individual pages for each specification
func (g *Generator) renderSpecPages(suite *models.EnhancedSuiteResult, outputDir string) error {
	var wg sync.WaitGroup
	errors := make(chan error, len(suite.SpecResults))

	// Limit concurrency
	semaphore := make(chan struct{}, g.config.MaxConcurrentGen)

	for _, spec := range suite.SpecResults {
		wg.Add(1)
		go func(s *models.SpecResult) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			specPath := filepath.Join(outputDir, s.GetHTMLFileName())
			if err := g.renderer.RenderSpec(suite, s, specPath); err != nil {
				errors <- err
			}
		}(spec)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var firstError error
	for err := range errors {
		if firstError == nil {
			firstError = err
		}
		logger.Errorf("Failed to render spec page: %v", err)
	}

	return firstError
}

// generateSearchIndex creates a searchable index of all test results
func (g *Generator) generateSearchIndex(suite *models.EnhancedSuiteResult, outputDir string) error {
	index := buildSearchIndex(suite)

	indexPath := filepath.Join(outputDir, "js", g.config.SearchIndexPath)
	if err := os.MkdirAll(filepath.Dir(indexPath), 0755); err != nil {
		return err
	}

	data, err := json.Marshal(index)
	if err != nil {
		return err
	}

	return os.WriteFile(indexPath, data, 0644)
}

// exportToFormats exports the report to additional formats (PDF, JSON, etc.)
func (g *Generator) exportToFormats(suite *models.EnhancedSuiteResult, outputDir string) error {
	for _, format := range g.config.ExportFormats {
		if format == "html" {
			continue // Already generated
		}

		logger.Infof("Exporting to %s...", format)
		if err := g.exporter.Export(suite, outputDir, format); err != nil {
			logger.Warnf("Failed to export to %s: %v", format, err)
		}
	}
	return nil
}

// copyScreenshots copies screenshot files to the output directory
func (g *Generator) copyScreenshots(suite *models.EnhancedSuiteResult, outputDir string) error {
	screenshotDir := filepath.Join(outputDir, "screenshots")
	if err := os.MkdirAll(screenshotDir, 0755); err != nil {
		return err
	}

	// TODO: Implement screenshot copying
	return nil
}

// buildSearchIndex creates a search index from suite results
func buildSearchIndex(suite *models.EnhancedSuiteResult) map[string]interface{} {
	index := make(map[string]interface{})
	specs := make([]map[string]interface{}, 0)

	for _, spec := range suite.SpecResults {
		specData := map[string]interface{}{
			"heading":  spec.SpecHeading,
			"fileName": spec.FileName,
			"tags":     spec.Tags,
			"failed":   spec.Failed,
			"skipped":  spec.Skipped,
		}
		specs = append(specs, specData)
	}

	index["specs"] = specs
	index["tags"] = collectAllTags(suite)

	return index
}

// collectAllTags collects all unique tags from the suite
func collectAllTags(suite *models.EnhancedSuiteResult) []string {
	tagMap := make(map[string]bool)
	for _, spec := range suite.SpecResults {
		for _, tag := range spec.Tags {
			tagMap[tag] = true
		}
	}

	tags := make([]string, 0, len(tagMap))
	for tag := range tagMap {
		tags = append(tags, tag)
	}
	return tags
}

// calculateSuccessRate calculates the success rate from proto results
func calculateSuccessRate(proto *gauge_messages.ProtoSuiteResult) float64 {
	totalSpecs := len(proto.GetSpecResults())
	if totalSpecs == 0 {
		return 0.0
	}

	passedSpecs := 0
	for _, spec := range proto.GetSpecResults() {
		if !spec.GetFailed() && !spec.GetSkipped() {
			passedSpecs++
		}
	}

	return float64(passedSpecs) / float64(totalSpecs) * 100
}

// transformHookFailure converts proto hook failure to model
func transformHookFailure(proto *gauge_messages.ProtoHookFailure) *models.HookFailure {
	if proto == nil {
		return nil
	}

	return &models.HookFailure{
		ErrorMessage: proto.GetErrorMessage(),
		StackTrace:   proto.GetStackTrace(),
		//nolint:staticcheck // Using deprecated Gauge proto method until framework provides alternative
		Screenshot: proto.GetScreenShot(),
	}
}
