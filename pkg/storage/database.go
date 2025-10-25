package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/lirany1/gauge-html-report-ai/pkg/logger"
)

// Database handles historical test execution data
type Database struct {
	db   *sql.DB
	path string
}

// ExecutionRecord represents a single test execution run
type ExecutionRecord struct {
	ID               string                 `json:"id"`
	Timestamp        time.Time              `json:"timestamp"`
	Duration         int64                  `json:"duration"`
	TotalScenarios   int                    `json:"totalScenarios"`
	PassedScenarios  int                    `json:"passedScenarios"`
	FailedScenarios  int                    `json:"failedScenarios"`
	SkippedScenarios int                    `json:"skippedScenarios"`
	SuccessRate      float64                `json:"successRate"`
	Environment      string                 `json:"environment"`
	Tags             []string               `json:"tags"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// ScenarioRecord represents a single scenario execution
type ScenarioRecord struct {
	ExecutionID  string `json:"executionId"`
	ScenarioName string `json:"scenarioName"`
	SpecName     string `json:"specName"`
	Status       string `json:"status"`
	Duration     int64  `json:"duration"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	StackTrace   string `json:"stackTrace,omitempty"`
}

// NewDatabase creates or opens the historical database
func NewDatabase(reportsDir string) (*Database, error) {
	historyDir := filepath.Join(reportsDir, ".gauge-history")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create history directory: %w", err)
	}

	dbPath := filepath.Join(historyDir, "test-history.db")
	logger.Infof("Opening database at: %s", dbPath)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		db:   db,
		path: dbPath,
	}

	// Run migrations
	if err := database.migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	logger.Infof("Database initialized successfully")
	return database, nil
}

// migrate creates or updates the database schema
func (d *Database) migrate() error {
	migrations := []string{
		// Executions table
		`CREATE TABLE IF NOT EXISTS executions (
			id TEXT PRIMARY KEY,
			timestamp DATETIME NOT NULL,
			duration INTEGER NOT NULL,
			total_scenarios INTEGER,
			passed_scenarios INTEGER,
			failed_scenarios INTEGER,
			skipped_scenarios INTEGER,
			success_rate REAL,
			environment TEXT,
			tags TEXT,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Index for timestamp queries
		`CREATE INDEX IF NOT EXISTS idx_execution_timestamp 
		 ON executions(timestamp DESC)`,

		// Scenario history table
		`CREATE TABLE IF NOT EXISTS scenario_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			execution_id TEXT NOT NULL,
			scenario_name TEXT NOT NULL,
			spec_name TEXT NOT NULL,
			status TEXT NOT NULL,
			duration INTEGER,
			error_message TEXT,
			stack_trace TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (execution_id) REFERENCES executions(id)
		)`,

		// Indexes for scenario queries
		`CREATE INDEX IF NOT EXISTS idx_scenario_name 
		 ON scenario_history(scenario_name)`,

		`CREATE INDEX IF NOT EXISTS idx_scenario_execution 
		 ON scenario_history(execution_id)`,

		// Failure patterns table for grouping
		`CREATE TABLE IF NOT EXISTS failure_patterns (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			error_signature TEXT NOT NULL UNIQUE,
			first_seen DATETIME,
			last_seen DATETIME,
			occurrence_count INTEGER DEFAULT 1,
			classification TEXT,
			ai_analysis TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE INDEX IF NOT EXISTS idx_error_signature 
		 ON failure_patterns(error_signature)`,

		// Performance metrics table
		`CREATE TABLE IF NOT EXISTS performance_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			execution_id TEXT NOT NULL,
			step_text TEXT NOT NULL,
			duration INTEGER NOT NULL,
			scenario_name TEXT,
			spec_name TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (execution_id) REFERENCES executions(id)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_step_performance 
		 ON performance_metrics(step_text, duration)`,
	}

	for i, migration := range migrations {
		if _, err := d.db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}

	logger.Infof("Database migrations completed")
	return nil
}

// SaveExecution saves an execution record to the database
func (d *Database) SaveExecution(exec *ExecutionRecord) error {
	query := `
		INSERT INTO executions (
			id, timestamp, duration, total_scenarios,
			passed_scenarios, failed_scenarios, skipped_scenarios,
			success_rate, environment, tags, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	tagsJSON, _ := json.Marshal(exec.Tags)
	metadataJSON, _ := json.Marshal(exec.Metadata)

	_, err := d.db.Exec(query,
		exec.ID,
		exec.Timestamp.Format(time.RFC3339),
		exec.Duration,
		exec.TotalScenarios,
		exec.PassedScenarios,
		exec.FailedScenarios,
		exec.SkippedScenarios,
		exec.SuccessRate,
		exec.Environment,
		string(tagsJSON),
		string(metadataJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to save execution: %w", err)
	}

	logger.Infof("Saved execution record: %s", exec.ID)
	return nil
}

// SaveScenario saves a scenario result
func (d *Database) SaveScenario(scenario *ScenarioRecord) error {
	query := `
		INSERT INTO scenario_history (
			execution_id, scenario_name, spec_name, status,
			duration, error_message, stack_trace
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query,
		scenario.ExecutionID,
		scenario.ScenarioName,
		scenario.SpecName,
		scenario.Status,
		scenario.Duration,
		scenario.ErrorMessage,
		scenario.StackTrace,
	)

	return err
}

// GetRecentExecutions retrieves the last N executions
func (d *Database) GetRecentExecutions(limit int) ([]ExecutionRecord, error) {
	query := `
		SELECT 
			id, timestamp, duration, total_scenarios,
			passed_scenarios, failed_scenarios, skipped_scenarios,
			success_rate, environment, tags, metadata
		FROM executions
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []ExecutionRecord
	for rows.Next() {
		var exec ExecutionRecord
		var timestamp string
		var tagsJSON, metadataJSON string

		err := rows.Scan(
			&exec.ID,
			&timestamp,
			&exec.Duration,
			&exec.TotalScenarios,
			&exec.PassedScenarios,
			&exec.FailedScenarios,
			&exec.SkippedScenarios,
			&exec.SuccessRate,
			&exec.Environment,
			&tagsJSON,
			&metadataJSON,
		)
		if err != nil {
			continue
		}

		exec.Timestamp, _ = time.Parse(time.RFC3339, timestamp)
		json.Unmarshal([]byte(tagsJSON), &exec.Tags)
		json.Unmarshal([]byte(metadataJSON), &exec.Metadata)

		executions = append(executions, exec)
	}

	return executions, nil
}

// GetScenarioHistory retrieves historical data for a specific scenario
func (d *Database) GetScenarioHistory(scenarioName string, days int) ([]ScenarioRecord, error) {
	query := `
		SELECT 
			sh.execution_id, sh.scenario_name, sh.spec_name,
			sh.status, sh.duration, sh.error_message, sh.stack_trace
		FROM scenario_history sh
		JOIN executions e ON sh.execution_id = e.id
		WHERE sh.scenario_name = ?
		AND e.timestamp >= datetime('now', '-' || ? || ' days')
		ORDER BY e.timestamp DESC
	`

	rows, err := d.db.Query(query, scenarioName, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scenarios []ScenarioRecord
	for rows.Next() {
		var scenario ScenarioRecord
		err := rows.Scan(
			&scenario.ExecutionID,
			&scenario.ScenarioName,
			&scenario.SpecName,
			&scenario.Status,
			&scenario.Duration,
			&scenario.ErrorMessage,
			&scenario.StackTrace,
		)
		if err != nil {
			continue
		}
		scenarios = append(scenarios, scenario)
	}

	return scenarios, nil
}

// CalculateFlakyScore calculates how flaky a scenario is (0.0 = stable, 1.0 = very flaky)
func (d *Database) CalculateFlakyScore(scenarioName string, days int) (float64, error) {
	query := `
		SELECT 
			COUNT(*) as total_runs,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_runs
		FROM scenario_history sh
		JOIN executions e ON sh.execution_id = e.id
		WHERE sh.scenario_name = ?
		AND e.timestamp >= datetime('now', '-' || ? || ' days')
	`

	var totalRuns, failedRuns int
	err := d.db.QueryRow(query, scenarioName, days).Scan(&totalRuns, &failedRuns)
	if err != nil {
		return 0.0, err
	}

	if totalRuns < 3 {
		return 0.0, nil // Not enough data
	}

	failureRate := float64(failedRuns) / float64(totalRuns)

	// Flaky score is highest when failure rate is around 50%
	// Scale: 0% or 100% failure = 0.0 (stable), 50% failure = 1.0 (very flaky)
	flakyScore := 1.0 - (2.0 * abs(failureRate-0.5))

	return flakyScore, nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// GetTrendData retrieves trend data for the last N days
func (d *Database) GetTrendData(days int) ([]TrendPoint, error) {
	query := `
		SELECT 
			timestamp,
			success_rate,
			duration,
			total_scenarios,
			passed_scenarios,
			failed_scenarios
		FROM executions
		WHERE timestamp >= datetime('now', '-' || ? || ' days')
		ORDER BY timestamp ASC
	`

	rows, err := d.db.Query(query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []TrendPoint
	for rows.Next() {
		var tp TrendPoint
		var timestamp string

		err := rows.Scan(
			&timestamp,
			&tp.SuccessRate,
			&tp.Duration,
			&tp.Total,
			&tp.Passed,
			&tp.Failed,
		)
		if err != nil {
			continue
		}

		tp.Timestamp, _ = time.Parse(time.RFC3339, timestamp)
		trends = append(trends, tp)
	}

	return trends, nil
}

// TrendPoint represents a single point in trend data
type TrendPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	SuccessRate float64   `json:"successRate"`
	Duration    int64     `json:"duration"`
	Total       int       `json:"total"`
	Passed      int       `json:"passed"`
	Failed      int       `json:"failed"`
}

// CleanupOldData removes data older than specified days
func (d *Database) CleanupOldData(retentionDays int) error {
	tables := []string{"scenario_history", "performance_metrics", "executions"}

	for _, table := range tables {
		query := fmt.Sprintf(`
			DELETE FROM %s 
			WHERE created_at < datetime('now', '-' || ? || ' days')
		`, table)

		result, err := d.db.Exec(query, retentionDays)
		if err != nil {
			logger.Warnf("Failed to cleanup %s: %v", table, err)
			continue
		}

		rows, _ := result.RowsAffected()
		logger.Infof("Cleaned up %d old records from %s", rows, table)
	}

	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}
