package api

import (
	"encoding/json"
	"net/http"

	"email-bot/internal/config"
	"email-bot/internal/core"
)

type Server struct {
	cfg    *config.Config
	logger *core.LogManager
}

func NewServer(cfg *config.Config, logger *core.LogManager) *Server {
	return &Server{
		cfg:    cfg,
		logger: logger,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/status", s.handleStatus)

	s.logger.Infof("Starting API server on %s", s.cfg.API.Address)
	return http.ListenAndServe(s.cfg.API.Address, mux)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	logs := s.logger.GetLogs(100) // return last 100 logs
	json.NewEncoder(w).Encode(logs)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// For simplicity, we just return basic config info as status here.
	// You can expand this to include live metrics (e.g., messages processed).
	status := map[string]interface{}{
		"status":        "running",
		"sources_count": len(s.cfg.Sources),
		"targets_count": len(s.cfg.Targets),
		"rules_count":   len(s.cfg.Rules),
	}
	json.NewEncoder(w).Encode(status)
}
