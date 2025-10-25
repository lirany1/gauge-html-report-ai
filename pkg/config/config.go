package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config holds the configuration for enhanced report generation
type Config struct {
	// General settings
	ProjectName string
	ReportsDir  string
	ThemePath   string
	MinifyHTML  bool

	// Analytics settings
	EnableAnalytics    bool
	EnableTrends       bool
	HistoricalData     bool
	TrendWindowDays    int
	FlakyTestDetection bool

	// Export settings
	ExportFormats     []string
	PDFTemplate       string
	MaxScreenshotSize string

	// Notification settings
	EnableNotifications  bool
	NotificationChannels []string
	SlackWebhook         string
	EmailRecipients      []string

	// Integration settings
	JiraIntegration    bool
	JiraProjectKey     string
	JenkinsIntegration bool
	JenkinsBuildURL    string

	// Performance settings
	MaxConcurrentGen int
	CacheEnabled     bool
	CacheTTL         time.Duration

	// UI settings
	DefaultTheme     string
	EnableDarkMode   bool
	ShowTimeline     bool
	ShowTrends       bool
	CollapseSections bool

	// Search settings
	EnableFullTextSearch bool
	EnableRegex          bool
	SearchIndexPath      string

	// Custom fields
	CustomFields map[string]interface{}
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return &Config{
		ProjectName:          getProjectName(),
		ReportsDir:           "reports",
		ThemePath:            "enhanced-default",
		MinifyHTML:           false,
		EnableAnalytics:      true,
		EnableTrends:         true,
		HistoricalData:       true,
		TrendWindowDays:      30,
		FlakyTestDetection:   true,
		ExportFormats:        []string{"html"},
		PDFTemplate:          "default",
		MaxScreenshotSize:    "2MB",
		EnableNotifications:  false,
		NotificationChannels: []string{},
		MaxConcurrentGen:     4,
		CacheEnabled:         true,
		CacheTTL:             24 * time.Hour,
		DefaultTheme:         "light",
		EnableDarkMode:       true,
		ShowTimeline:         true,
		ShowTrends:           true,
		CollapseSections:     false,
		EnableFullTextSearch: true,
		EnableRegex:          true,
		SearchIndexPath:      "search_index.json",
		CustomFields:         make(map[string]interface{}),
	}
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return NewConfig()
}

// LoadConfig loads configuration from file or returns default
func LoadConfig() (*Config, error) {
	cfg := NewConfig()

	// Try to load from config file
	configPaths := []string{
		"gauge-report-config.yml",
		"gauge-report-config.yaml",
		"gauge-report-config.json",
		".gauge/report-config.yml",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			if err := cfg.LoadFromFile(path); err == nil {
				cfg.LoadFromEnv() // Override with env vars
				return cfg, nil
			}
		}
	}

	// No config file found, load from env only
	cfg.LoadFromEnv()
	return cfg, nil
}

// LoadFromFile loads configuration from a file (YAML, JSON, or TOML)
func (c *Config) LoadFromFile(path string) error {
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return err
	}

	return v.Unmarshal(c)
}

// LoadFromEnv loads configuration from environment variables
func (c *Config) LoadFromEnv() {
	if dir := os.Getenv("GAUGE_REPORTS_DIR"); dir != "" {
		c.ReportsDir = dir
	}

	if theme := os.Getenv("GAUGE_HTML_THEME"); theme != "" {
		c.ThemePath = theme
	}

	if analytics := os.Getenv("GAUGE_ENABLE_ANALYTICS"); analytics == "true" {
		c.EnableAnalytics = true
	}

	if minify := os.Getenv("GAUGE_MINIFY_REPORTS"); minify == "true" {
		c.MinifyHTML = true
	}

	if webhook := os.Getenv("SLACK_WEBHOOK_URL"); webhook != "" {
		c.SlackWebhook = webhook
		c.EnableNotifications = true
		c.NotificationChannels = append(c.NotificationChannels, "slack")
	}

	if jiraKey := os.Getenv("JIRA_PROJECT_KEY"); jiraKey != "" {
		c.JiraProjectKey = jiraKey
		c.JiraIntegration = true
	}

	if buildURL := os.Getenv("BUILD_URL"); buildURL != "" {
		c.JenkinsBuildURL = buildURL
		c.JenkinsIntegration = true
	}
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	v := viper.New()
	v.SetConfigFile(path)

	// Map config to viper
	v.Set("project_name", c.ProjectName)
	v.Set("reports_dir", c.ReportsDir)
	v.Set("theme_path", c.ThemePath)
	v.Set("minify_html", c.MinifyHTML)
	v.Set("enable_analytics", c.EnableAnalytics)
	v.Set("enable_trends", c.EnableTrends)
	v.Set("export_formats", c.ExportFormats)

	return v.WriteConfig()
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Add validation logic
	return nil
}

// getProjectName tries to get project name from current directory
func getProjectName() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "Gauge Project"
	}
	return filepath.Base(cwd)
}
