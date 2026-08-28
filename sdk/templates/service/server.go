package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/unhield/limoxel/sdk"
)

type Server struct {
	client *sdk.Client
	mux    *http.ServeMux
}

func NewServer(repoPath string) (*Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open workspace at %q: %w", repoPath, err)
	}

	s := &Server{
		client: client,
		mux:    http.NewServeMux(),
	}
	s.routes()
	return s, nil
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/v1/info", s.handleInfo)
	s.mux.HandleFunc("/api/v1/health", s.handleHealth)
	s.mux.HandleFunc("/api/v1/search", s.handleSearch)
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	info, err := s.client.Repository().Info(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	health, err := s.client.Analysis().RepositoryHealth(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, health)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	res, err := s.client.Search().SearchSymbols(ctx, q, sdk.PaginationOptions{Limit: 20})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func main() {
	repoPath := "."
	port := ":8080"

	srv, err := NewServer(repoPath)
	if err != nil {
		log.Fatalf("Server startup failure: %v", err)
	}
	defer srv.client.Close()

	httpServer := &http.Server{
		Addr:    port,
		Handler: srv.mux,
	}

	go func() {
		fmt.Printf("Limoxel Intelligence Microservice running on %s\n", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
	fmt.Println("Server gracefully stopped.")
}
