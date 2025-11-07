package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gabrielpetry/alertmanager-alert-sync/internal/alertmanager"
	"github.com/gabrielpetry/alertmanager-alert-sync/internal/grafana"
	"github.com/gabrielpetry/alertmanager-alert-sync/internal/metrics"
	"github.com/gabrielpetry/alertmanager-alert-sync/internal/server"
	"github.com/gabrielpetry/alertmanager-alert-sync/internal/sync"
)

func main() {
	log.Println("Starting Alertmanager Alert Sync...")

	// Initialize Alertmanager client
	amClient := alertmanager.NewClient()

	// Initialize Grafana IRM client
	grafanaClient, err := grafana.NewClient()
	if err != nil {
		log.Printf("Warning: Grafana client initialization failed: %v", err)
		log.Println("Reconciliation features will be disabled")
		grafanaClient = nil
	}

	// Initialize metrics exporter
	exporter := metrics.NewExporter()

	// Initialize reconciler (if Grafana client is available)
	var reconciler *sync.Reconciler
	if grafanaClient != nil {
		reconciler = sync.NewReconciler(amClient, grafanaClient, exporter)
	}

	// Initialize server with all dependencies
	srv := server.NewServer(amClient, grafanaClient, exporter, reconciler)

	// Initialize webhook handler if Grafana client is available
	var webhookHandler *server.WebhookHandler
	if grafanaClient != nil {
		webhookHandler = server.NewWebhookHandler(amClient, grafanaClient)
	}

	// Start background reconciliation if enabled
	if reconciler != nil {
		reconcileIntervalStr := os.Getenv("RECONCILE_INTERVAL")
		if reconcileIntervalStr != "" {
			interval, err := strconv.Atoi(reconcileIntervalStr)
			if err != nil || interval <= 0 {
				log.Printf("Invalid RECONCILE_INTERVAL value '%s', must be a positive integer (seconds)", reconcileIntervalStr)
			} else {
				// Use optimized reconciliation that handles both sync and metrics export
				go startOptimizedReconciliationLoop(reconciler, time.Duration(interval)*time.Second)
				log.Printf("Optimized background reconciliation enabled with interval: %d seconds", interval)
				log.Println("This includes both alert metrics export and silence synchronization")
			}
		} else {
			log.Println("Background reconciliation disabled (set RECONCILE_INTERVAL to enable)")
		}
	} else {
		log.Println("Grafana IRM integration disabled - no background processing available")
	}

	// Register HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", srv.MetricsHandler)
	mux.HandleFunc("/healthz", srv.HealthzHandler)
	mux.HandleFunc("/readyz", srv.ReadyzHandler)

	// Only register webhook endpoints if Grafana client is available
	if grafanaClient != nil {
		if webhookHandler != nil {
			webhookHandler.RegisterRoutes(mux)
			log.Println("Webhook endpoint enabled at /webhook (requires basic auth)")
		}
		log.Println("Grafana IRM integration enabled")
	} else {
		log.Println("Grafana IRM integration disabled")
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start the server
	log.Printf("Server listening on port :%s", port)
	log.Printf("Endpoints:")
	log.Printf("  - /metrics: Prometheus metrics for reconciliation")
	log.Printf("  - /healthz: Liveness probe")
	log.Printf("  - /readyz: Readiness probe")
	if grafanaClient != nil {
		if webhookHandler != nil {
			log.Printf("  - /webhook: Grafana IRM webhook endpoint (POST, basic auth required)")
		}
	}

	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), mux); err != nil {
		log.Fatal(err)
	}
}

// startOptimizedReconciliationLoop runs the optimized reconciliation process at regular intervals
// This handles both metrics export and silence synchronization in parallel
func startOptimizedReconciliationLoop(reconciler *sync.Reconciler, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting optimized reconciliation loop with interval: %v", interval)

	// Run immediately on startup
	runOptimizedReconciliation(reconciler)

	// Then run on interval
	for range ticker.C {
		runOptimizedReconciliation(reconciler)
	}
}

// runOptimizedReconciliation performs a single optimized reconciliation cycle with error handling
func runOptimizedReconciliation(reconciler *sync.Reconciler) {
	ctx := context.Background()
	log.Println("Running scheduled optimized reconciliation...")

	if err := reconciler.ReconcileAndResolveOptimized(ctx); err != nil {
		log.Printf("Optimized reconciliation failed: %v", err)
	} else {
		log.Println("Optimized reconciliation completed successfully")
	}
}


