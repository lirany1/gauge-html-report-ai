package themes

import (
	"os"
	"path/filepath"

	"github.com/getgauge/common"
	"github.com/lirany1/gauge-html-report-ai/pkg/config"
)

// Manager handles theme management
type Manager struct {
	config *config.Config
}

// NewManager creates a new theme manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{config: cfg}
}

// CopyAssets copies theme assets to output directory
func (m *Manager) CopyAssets(themeName, outputDir string) error {
	themePath := m.getThemePath(themeName)
	assetsPath := filepath.Join(themePath, "assets")

	if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
		// Theme doesn't have assets directory, skip
		return nil
	}

	_, err := common.MirrorDir(assetsPath, outputDir)
	return err
}

// getThemePath returns the full path to a theme
func (m *Manager) getThemePath(themeName string) string {
	// Check if it's an absolute path
	if filepath.IsAbs(themeName) {
		return themeName
	}

	// Check in project themes directory
	projectThemes := filepath.Join("themes", themeName)
	if _, err := os.Stat(projectThemes); err == nil {
		return projectThemes
	}

	// Check in builtin themes
	builtinThemes := filepath.Join("web", "themes", themeName)
	return builtinThemes
}
