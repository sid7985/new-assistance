package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"nanoclaw-orchestrator/internal"
)

type Server struct {
	db *internal.Database
}

func StartServer(port string, db *internal.Database) {
	s := &Server{db: db}

	mux := http.NewServeMux()

	// CORS Middleware
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	mux.HandleFunc("/api/budget", s.handleBudget)
	mux.HandleFunc("/api/missions", s.handleMissions)
	mux.HandleFunc("/api/audit", s.handleAudit)

	fmt.Printf("🌐 Starting NanoClaw REST API on port %s...\n", port)
	err := http.ListenAndServe(":"+port, corsMiddleware(mux))
	if err != nil {
		fmt.Printf("⚠️  API Server failed: %v\n", err)
	}
}

func (s *Server) handleBudget(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, "Database unavailable", http.StatusInternalServerError)
		return
	}

	limit, spent, err := s.db.GetBudgetStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"limitCents": limit,
		"spentCents": spent,
	})
}

// Structs to help serialize DB outputs
type Mission struct {
	ID         int64  `json:"id"`
	Goal       string `json:"goal"`
	Status     string `json:"status"`
	TokensUsed int    `json:"tokens_used"`
	ApiCalls   int    `json:"api_calls"`
	CreatedAt  string `json:"created_at"`
}

func (s *Server) handleMissions(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, "Database unavailable", http.StatusInternalServerError)
		return
	}

	rows, err := s.db.Conn.Query(`
		SELECT id, goal, status, tokens_used, api_calls, created_at 
		FROM missions ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var missions []Mission
	for rows.Next() {
		var m Mission
		if err := rows.Scan(&m.ID, &m.Goal, &m.Status, &m.TokensUsed, &m.ApiCalls, &m.CreatedAt); err == nil {
			missions = append(missions, m)
		}
	}

	json.NewEncoder(w).Encode(missions)
}

type AuditLog struct {
	ID           int    `json:"id"`
	MissionID    int    `json:"mission_id"`
	ActionType   string `json:"action_type"`
	ActionDetail string `json:"action_detail"`
	Source       string `json:"source"`
	TokensUsed   int    `json:"tokens_used"`
	CreatedAt    string `json:"created_at"`
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, "Database unavailable", http.StatusInternalServerError)
		return
	}

	rows, err := s.db.Conn.Query(`
		SELECT id, mission_id, action_type, action_detail, source, tokens_used, created_at 
		FROM audit_log ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(&l.ID, &l.MissionID, &l.ActionType, &l.ActionDetail, &l.Source, &l.TokensUsed, &l.CreatedAt); err == nil {
			logs = append(logs, l)
		}
	}

	json.NewEncoder(w).Encode(logs)
}
