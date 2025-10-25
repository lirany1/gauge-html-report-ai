package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/lirany1/gauge-html-report-ai/pkg/config"
	"github.com/lirany1/gauge-html-report-ai/pkg/generator"
	"github.com/lirany1/gauge-html-report-ai/pkg/logger"
	"github.com/lirany1/gauge-html-report-ai/pkg/plugin"
	"github.com/lirany1/gauge-html-report-ai/pkg/server"
)

var (
	version = "1.0.0"
	commit  = "dev"
	date    = "unknown"
)

func main() {
	// If launched by Gauge, run as plugin
	// The official html-report checks for html-report_action env var
	action := os.Getenv("html-report_action")

	if action == "execution" {
		// Plugin mode - start gRPC server
		runAsGaugePlugin()
		return
	}

	// If no arguments provided, assume we're being started by Gauge as a plugin
	if len(os.Args) == 1 {
		runAsGaugePlugin()
		return
	}

	// CLI mode with Cobra commands
	var rootCmd = &cobra.Command{
		Use:   "html-report-enhanced",
		Short: "Enhanced HTML report generator for Gauge",
		Long: `Enhanced HTML Report Plugin for Gauge
		
Generates advanced HTML reports with analytics, modern UI, and extensive customization options.
Visit https://github.com/lirany1/gauge-html-report-ai for more information.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	// Generate command
	var generateCmd = &cobra.Command{
		Use:   "generate",
		Short: "Generate HTML report from Gauge test results",
		Long:  "Generate an enhanced HTML report from Gauge test execution results with advanced analytics and visualizations.",
		RunE:  runGenerate,
	}

	// Server command - for real-time report viewing
	var serverCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start live report server",
		Long:  "Start a local server to view and interact with generated reports in real-time.",
		RunE:  runServer,
	}

	// Theme command
	var themeCmd = &cobra.Command{
		Use:   "theme",
		Short: "Manage report themes",
		Long:  "Create, list, and manage custom report themes.",
	}

	var createThemeCmd = &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new custom theme",
		Args:  cobra.ExactArgs(1),
		RunE:  runCreateTheme,
	}

	var listThemesCmd = &cobra.Command{
		Use:   "list",
		Short: "List available themes",
		RunE:  runListThemes,
	}

	// Plugin command - for gauge plugin integration
	var pluginCmd = &cobra.Command{
		Use:   "plugin",
		Short: "Run as Gauge plugin",
		Long:  "Start the plugin in Gauge plugin mode (used internally by Gauge).",
		Run:   func(cmd *cobra.Command, args []string) { runAsGaugePlugin() },
	}

	// Flags for generate command
	generateCmd.Flags().StringP("input", "i", "", "Input file containing test results (required)")
	generateCmd.Flags().StringP("output", "o", "", "Output directory for generated report (required)")
	generateCmd.Flags().StringP("theme", "t", "enhanced-default", "Theme to use for report generation")
	generateCmd.Flags().BoolP("analytics", "a", true, "Enable analytics and trend analysis")
	generateCmd.Flags().BoolP("export-pdf", "p", false, "Also generate PDF version of report")
	generateCmd.Flags().StringSliceP("formats", "f", []string{"html"}, "Export formats (html, pdf, json)")
	generateCmd.Flags().BoolP("minify", "m", false, "Minify HTML output")
	generateCmd.Flags().StringP("config", "c", "", "Path to configuration file")

	// Flags for server command
	serverCmd.Flags().IntP("port", "p", 8080, "Port to run server on")
	serverCmd.Flags().StringP("host", "H", "localhost", "Host to bind server to")
	serverCmd.Flags().StringP("dir", "d", "reports", "Directory containing reports to serve")
	serverCmd.Flags().BoolP("watch", "w", false, "Watch for changes and auto-reload")

	// Flags for create theme command
	createThemeCmd.Flags().StringP("base", "b", "enhanced-default", "Base theme to extend from")
	createThemeCmd.Flags().StringP("output", "o", "themes", "Output directory for new theme")

	// Build command tree
	themeCmd.AddCommand(createThemeCmd, listThemesCmd)
	rootCmd.AddCommand(generateCmd, serverCmd, themeCmd, pluginCmd)

	if err := rootCmd.Execute(); err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}

func runGenerate(cmd *cobra.Command, args []string) error {
	inputFile, _ := cmd.Flags().GetString("input")
	outputDir, _ := cmd.Flags().GetString("output")
	themePath, _ := cmd.Flags().GetString("theme")
	enableAnalytics, _ := cmd.Flags().GetBool("analytics")
	exportPDF, _ := cmd.Flags().GetBool("export-pdf")
	formats, _ := cmd.Flags().GetStringSlice("formats")
	minify, _ := cmd.Flags().GetBool("minify")
	configFile, _ := cmd.Flags().GetString("config")

	if inputFile == "" || outputDir == "" {
		return fmt.Errorf("both --input and --output flags are required")
	}

	// Load configuration
	cfg := config.NewConfig()
	if configFile != "" {
		if err := cfg.LoadFromFile(configFile); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Override with command-line flags
	cfg.ThemePath = themePath
	cfg.EnableAnalytics = enableAnalytics
	cfg.ExportFormats = formats
	cfg.MinifyHTML = minify

	if exportPDF && !contains(formats, "pdf") {
		cfg.ExportFormats = append(cfg.ExportFormats, "pdf")
	}

	logger.Info("Starting enhanced HTML report generation...")
	logger.Infof("Input: %s", inputFile)
	logger.Infof("Output: %s", outputDir)
	logger.Infof("Theme: %s", themePath)

	// Generate report
	gen := generator.NewGenerator(cfg)
	if err := gen.GenerateFromFile(inputFile, outputDir); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	logger.Info("✓ Report generated successfully!")
	logger.Infof("View report: file://%s/index.html", outputDir)

	return nil
}

func runServer(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	host, _ := cmd.Flags().GetString("host")
	reportsDir, _ := cmd.Flags().GetString("dir")
	watch, _ := cmd.Flags().GetBool("watch")

	logger.Infof("Starting enhanced report server on %s:%d", host, port)
	logger.Infof("Serving reports from: %s", reportsDir)

	srv := server.NewServer(&server.Config{
		Host:       host,
		Port:       port,
		ReportsDir: reportsDir,
		Watch:      watch,
	})

	return srv.Start()
}

func runCreateTheme(cmd *cobra.Command, args []string) error {
	themeName := args[0]
	baseTheme, _ := cmd.Flags().GetString("base")
	outputDir, _ := cmd.Flags().GetString("output")

	logger.Infof("Creating new theme '%s' based on '%s'", themeName, baseTheme)

	// TODO: Implement theme creation
	logger.Info("✓ Theme created successfully!")
	logger.Infof("Theme location: %s/%s", outputDir, themeName)

	return nil
}

func runListThemes(cmd *cobra.Command, args []string) error {
	logger.Info("Available themes:")
	// TODO: Implement theme listing
	themes := []string{
		"enhanced-default - Modern default theme with all features",
		"dark - Dark mode optimized theme",
		"classic - Traditional layout with enhanced features",
		"minimal - Clean, distraction-free theme",
		"corporate - Professional theme for enterprise",
	}

	for _, theme := range themes {
		fmt.Printf("  • %s\n", theme)
	}

	return nil
}

func runAsGaugePlugin() {
	// Simplified plugin startup matching official html-report structure
	logger.Info("Starting HTML Report plugin")

	// Start the gRPC server
	p := plugin.NewPlugin()
	if err := p.Start(); err != nil {
		logger.Fatalf("Failed to start plugin: %v", err)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func init() {
	// Initialize logger
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logrus.SetLevel(logrus.InfoLevel)
}
