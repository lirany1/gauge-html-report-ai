package server

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lirany1/gauge-html-report-ai/pkg/logger"
)

// Config holds server configuration
type Config struct {
	Host       string
	Port       int
	ReportsDir string
	Watch      bool
}

// Server provides live report viewing
type Server struct {
	config *Config
	router *mux.Router
}

// NewServer creates a new report server
func NewServer(cfg *Config) *Server {
	s := &Server{
		config: cfg,
		router: mux.NewRouter(),
	}
	s.setupRoutes()
	return s
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	logger.Infof("Server running at http://%s", addr)
	logger.Infof("Press Ctrl+C to stop")

	return http.ListenAndServe(addr, s.router)
}

func (s *Server) setupRoutes() {
	// Serve static files
	fs := http.FileServer(http.Dir(s.config.ReportsDir))
	s.router.PathPrefix("/").Handler(fs)

	// API endpoints
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/reports", s.handleListReports).Methods("GET")
	api.HandleFunc("/reports/{id}", s.handleGetReport).Methods("GET")
}

func (s *Server) handleListReports(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement report listing
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"reports": []}`)) // Ignore write errors for TODO endpoint
}

func (s *Server) handleGetReport(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement report retrieval
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{}`)) // Ignore write errors for TODO endpoint
}
