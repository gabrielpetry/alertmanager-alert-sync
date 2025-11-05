package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gabrielpetry/alertmanager-alert-sync/internal/alertmanager"
	"github.com/gabrielpetry/alertmanager-alert-sync/internal/grafana"
	"github.com/gabrielpetry/alertmanager-alert-sync/internal/metrics"
	"github.com/gabrielpetry/alertmanager-alert-sync/internal/sync"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server holds all dependencies for HTTP handlers
type Server struct {
	amClient      *alertmanager.Client
	grafanaClient *grafana.Client
	exporter      *metrics.Exporter
	reconciler    *sync.Reconciler
}

// NewServer creates a new server with all dependencies
func NewServer(
	amClient *alertmanager.Client,
	grafanaClient *grafana.Client,
	exporter *metrics.Exporter,
	reconciler *sync.Reconciler,
) *Server {
	return &Server{
		amClient:      amClient,
		grafanaClient: grafanaClient,
		exporter:      exporter,
		reconciler:    reconciler,
	}
}

// MetricsHandler serves Prometheus metrics for reconciliation
func (s *Server) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}

// ReconcileHandler triggers a reconciliation between Alertmanager and Grafana IRM
// It identifies and logs inconsistencies between the two systems
func (s *Server) ReconcileHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Reconcile endpoint called...")

	err := s.reconciler.ReconcileAndResolveOptimized(r.Context())
	if err != nil {
		log.Printf("Error during reconciliation: %v", err)
		http.Error(w, "Error during reconciliation", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Reconciliation completed successfully\n")
}

// HealthzHandler provides a Kubernetes-style liveness probe endpoint
// Returns 200 OK if the service is running
func (s *Server) HealthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK\n")
}

// ReadyzHandler provides a Kubernetes-style readiness probe endpoint
// Returns 200 OK if the service is ready to accept traffic
func (s *Server) ReadyzHandler(w http.ResponseWriter, r *http.Request) {
	// Check if reconciler is initialized (requires Grafana client)
	if s.reconciler == nil {
		http.Error(w, "Not ready: reconciler not initialized", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Ready\n")
}
