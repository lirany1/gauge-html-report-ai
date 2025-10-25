package export

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/your-org/gauge-html-report-enhanced/pkg/config"
	"github.com/your-org/gauge-html-report-enhanced/pkg/models"
)

// Exporter handles exporting reports to various formats
type Exporter struct {
	config *config.Config
}

// NewExporter creates a new exporter
func NewExporter(cfg *config.Config) *Exporter {
	return &Exporter{config: cfg}
}

// Export exports the report to the specified format
func (e *Exporter) Export(suite *models.EnhancedSuiteResult, outputDir, format string) error {
	switch format {
	case "pdf":
		return e.exportPDF(suite, outputDir)
	case "json":
		return e.exportJSON(suite, outputDir)
	case "xml":
		return e.exportXML(suite, outputDir)
	default:
		return nil
	}
}

func (e *Exporter) exportPDF(suite *models.EnhancedSuiteResult, outputDir string) error {
	// PDF export using simple HTML to PDF conversion approach
	// Note: For production use, consider using libraries like wkhtmltopdf or chromedp
	pdfPath := filepath.Join(outputDir, "report.pdf")
	
	// Create a simple PDF content (placeholder implementation)
	content := fmt.Sprintf(`PDF Test Report
Project: %s
Execution Time: %s
Success Rate: %.1f%%
Total Scenarios: %d
Passed: %d
Failed: %d
Skipped: %d

Generated on: %s
`, 
		suite.ProjectName,
		suite.ExecutionTime.String(),
		suite.SuccessRate,
		suite.TotalScenariosCount,
		suite.PassedScenariosCount,
		suite.FailedScenariosCount,
		suite.SkippedScenariosCount,
		time.Now().Format("2006-01-02 15:04:05"))
	
	return os.WriteFile(pdfPath, []byte(content), 0644)
}

func (e *Exporter) exportJSON(suite *models.EnhancedSuiteResult, outputDir string) error {
	jsonPath := filepath.Join(outputDir, "report.json")
	
	// Create JSON export data
	exportData := map[string]interface{}{
		"project":        suite.ProjectName,
		"timestamp":      suite.Timestamp,
		"executionTime":  suite.ExecutionTime.String(),
		"successRate":    suite.SuccessRate,
		"environment":    suite.Environment,
		"summary": map[string]int{
			"total":   suite.TotalScenariosCount,
			"passed":  suite.PassedScenariosCount,
			"failed":  suite.FailedScenariosCount,
			"skipped": suite.SkippedScenariosCount,
		},
		"specifications": suite.SpecResults,
		"analytics":      suite.Analytics,
		"aiInsights":     suite.AIInsights,
		"exportedAt":     time.Now(),
	}
	
	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	
	return os.WriteFile(jsonPath, jsonData, 0644)
}

func (e *Exporter) exportXML(suite *models.EnhancedSuiteResult, outputDir string) error {
	xmlPath := filepath.Join(outputDir, "report.xml")
	
	// Create XML export structure
	type XMLReport struct {
		XMLName      xml.Name `xml:"testReport"`
		Project      string   `xml:"project"`
		Timestamp    string   `xml:"timestamp"`
		ExecutionTime string  `xml:"executionTime"`
		SuccessRate  float64  `xml:"successRate"`
		Environment  string   `xml:"environment"`
		Summary      struct {
			Total   int `xml:"total"`
			Passed  int `xml:"passed"`
			Failed  int `xml:"failed"`
			Skipped int `xml:"skipped"`
		} `xml:"summary"`
		ExportedAt string `xml:"exportedAt"`
	}
	
	xmlData := XMLReport{
		Project:       suite.ProjectName,
		Timestamp:     suite.Timestamp.Format(time.RFC3339),
		ExecutionTime: suite.ExecutionTime.String(),
		SuccessRate:   suite.SuccessRate,
		Environment:   suite.Environment,
		ExportedAt:    time.Now().Format(time.RFC3339),
	}
	xmlData.Summary.Total = suite.TotalScenariosCount
	xmlData.Summary.Passed = suite.PassedScenariosCount
	xmlData.Summary.Failed = suite.FailedScenariosCount
	xmlData.Summary.Skipped = suite.SkippedScenariosCount
	
	xmlBytes, err := xml.MarshalIndent(xmlData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %w", err)
	}
	
	// Add XML header
	xmlContent := []byte(xml.Header + string(xmlBytes))
	return os.WriteFile(xmlPath, xmlContent, 0644)
}
