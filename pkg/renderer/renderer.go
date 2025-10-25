package renderer

import (
	"github.com/lirany1/gauge-html-report-ai/pkg/config"
	"github.com/lirany1/gauge-html-report-ai/pkg/models"
)

// Renderer handles HTML template rendering
type Renderer struct {
	config *config.Config
}

// NewRenderer creates a new renderer
func NewRenderer(cfg *config.Config) *Renderer {
	return &Renderer{config: cfg}
}

// RenderIndex renders the main index.html page
func (r *Renderer) RenderIndex(suite *models.EnhancedSuiteResult, outputPath string) error {
	// TODO: Implement index rendering
	return nil
}

// RenderSpec renders an individual specification page
func (r *Renderer) RenderSpec(suite *models.EnhancedSuiteResult, spec *models.SpecResult, outputPath string) error {
	// TODO: Implement spec rendering
	return nil
}
