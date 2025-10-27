package plugin

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/getgauge/gauge-proto/go/gauge_messages"
	"github.com/lirany1/gauge-html-report-ai/pkg/builder"
	"github.com/lirany1/gauge-html-report-ai/pkg/config"
	"github.com/lirany1/gauge-html-report-ai/pkg/logger"
	"google.golang.org/grpc"
)

// Plugin represents the Gauge HTML report plugin
type Plugin struct {
	gauge_messages.UnimplementedReporterServer
	config        *config.Config
	server        *grpc.Server
	stopChan      chan struct{}
	reportBuilder *builder.ReportBuilder
}

// NewPlugin creates a new plugin instance
func NewPlugin() *Plugin {
	cfg := config.NewConfig()
	cfg.LoadFromEnv()

	return &Plugin{
		config:   cfg,
		stopChan: make(chan struct{}),
	}
}

// Start starts the plugin as a gRPC server
func (p *Plugin) Start() error {
	address, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to resolve TCP address: %w", err)
	}

	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	p.server = grpc.NewServer(grpc.MaxRecvMsgSize(1024 * 1024 * 1024)) // 1GB max message size
	gauge_messages.RegisterReporterServer(p.server, p)

	port := listener.Addr().(*net.TCPAddr).Port

	// Start serving in a goroutine
	go func() {
		if err := p.server.Serve(listener); err != nil {
			logger.Errorf("gRPC server error: %v", err)
		}
		close(p.stopChan) // Signal that server has stopped
	}()

	// Write port to stdout in the exact format Gauge expects
	// Gauge's CustomWriter looks for "Listening on port:XXXXX"
	if _, err := fmt.Fprintf(os.Stdout, "Listening on port:%d\n", port); err != nil {
		// Error writing to stdout
	}
	if err := os.Stdout.Sync(); err != nil {
		// Error syncing stdout
	}

	logger.Infof("gRPC server ready on port %d", port)

	// Wait for shutdown signal
	<-p.stopChan
	logger.Info("Plugin shutdown complete")
	return nil
}

// NotifyExecutionStarting is called when execution starts
func (p *Plugin) NotifyExecutionStarting(ctx context.Context, info *gauge_messages.ExecutionStartingRequest) (*gauge_messages.Empty, error) {
	logger.Info("Execution starting...")

	// Initialize report builder here so it's ready for the entire execution
	if p.reportBuilder == nil {
		// Get project root from environment (Gauge sets this)
		projectRoot := os.Getenv("GAUGE_PROJECT_ROOT")
		if projectRoot == "" {
			projectRoot = "."
		}

		// Get reports directory from environment or use default
		reportsDir := os.Getenv("gauge_reports_dir")
		if reportsDir == "" {
			reportsDir = filepath.Join(projectRoot, "reports")
		} else if !filepath.IsAbs(reportsDir) {
			reportsDir = filepath.Join(projectRoot, reportsDir)
		}

		// Create report builder with database connection
		p.reportBuilder = builder.NewReportBuilder(reportsDir, "enhanced-default")
		logger.Infof("Report builder initialized with database")
	}

	return &gauge_messages.Empty{}, nil
}

// NotifyExecutionEnding is called when execution ends
func (p *Plugin) NotifyExecutionEnding(ctx context.Context, result *gauge_messages.ExecutionEndingRequest) (*gauge_messages.Empty, error) {
	logger.Info("Execution ending, generating report...")
	return &gauge_messages.Empty{}, nil
}

// Kill stops the plugin
func (p *Plugin) Kill(ctx context.Context, request *gauge_messages.KillProcessRequest) (*gauge_messages.Empty, error) {
	logger.Info("Shutting down plugin...")
	if p.server != nil {
		// Stop the server gracefully
		go func() {
			p.server.GracefulStop()
		}()
	}
	// Signal shutdown (stopChan might already be closed by Serve goroutine)
	select {
	case <-p.stopChan:
		// Already closed
	default:
		close(p.stopChan)
	}
	return &gauge_messages.Empty{}, nil
}

// Implementing remaining Reporter interface methods
func (p *Plugin) NotifySpecExecutionStarting(ctx context.Context, info *gauge_messages.SpecExecutionStartingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (p *Plugin) NotifySpecExecutionEnding(ctx context.Context, result *gauge_messages.SpecExecutionEndingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (p *Plugin) NotifyScenarioExecutionStarting(ctx context.Context, info *gauge_messages.ScenarioExecutionStartingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (p *Plugin) NotifyScenarioExecutionEnding(ctx context.Context, result *gauge_messages.ScenarioExecutionEndingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (p *Plugin) NotifyStepExecutionStarting(ctx context.Context, info *gauge_messages.StepExecutionStartingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (p *Plugin) NotifyStepExecutionEnding(ctx context.Context, result *gauge_messages.StepExecutionEndingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (p *Plugin) NotifyConceptExecutionStarting(ctx context.Context, info *gauge_messages.ConceptExecutionStartingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (p *Plugin) NotifyConceptExecutionEnding(ctx context.Context, result *gauge_messages.ConceptExecutionEndingRequest) (*gauge_messages.Empty, error) {
	return &gauge_messages.Empty{}, nil
}

func (p *Plugin) NotifySuiteResult(ctx context.Context, result *gauge_messages.SuiteExecutionResult) (*gauge_messages.Empty, error) {
	// This is where we generate the final report
	logger.Info("Suite execution complete, generating enhanced report...")

	if result.GetSuiteResult() != nil {
		// Use the initialized report builder
		if p.reportBuilder == nil {
			logger.Warn("Report builder not initialized, creating new instance...")
			// Get project root from environment (Gauge sets this)
			projectRoot := os.Getenv("GAUGE_PROJECT_ROOT")
			if projectRoot == "" {
				projectRoot = "."
			}

			// Get reports directory from environment or use default
			reportsDir := os.Getenv("gauge_reports_dir")
			if reportsDir == "" {
				reportsDir = filepath.Join(projectRoot, "reports")
			} else if !filepath.IsAbs(reportsDir) {
				reportsDir = filepath.Join(projectRoot, reportsDir)
			}

			p.reportBuilder = builder.NewReportBuilder(reportsDir, "enhanced-default")
		}

		// Build the HTML report
		err := p.reportBuilder.BuildReport(result.GetSuiteResult())
		if err != nil {
			logger.Errorf("Failed to generate report: %v", err)
			return &gauge_messages.Empty{}, err
		}

		logger.Info("Enhanced HTML report generated successfully!")

		// Close database connection
		if err := p.reportBuilder.Close(); err != nil {
			logger.Warnf("Failed to close report builder: %v", err)
		}
	}

	return &gauge_messages.Empty{}, nil
}
