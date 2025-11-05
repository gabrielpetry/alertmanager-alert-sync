package sync

import (
	"context"
	"log"

	"github.com/gabrielpetry/alertmanager-alert-sync/internal/alertmanager"
	"github.com/gabrielpetry/alertmanager-alert-sync/internal/grafana"
	"github.com/gabrielpetry/alertmanager-alert-sync/internal/metrics"
	"github.com/prometheus/alertmanager/api/v2/models"
)

// Reconciler handles the synchronization between Alertmanager and Grafana IRM
type Reconciler struct {
	amClient      *alertmanager.Client
	grafanaClient *grafana.Client
	metrics       *metrics.Exporter
}

// NewReconciler creates a new Reconciler instance
func NewReconciler(amClient *alertmanager.Client, grafanaClient *grafana.Client, metricsExporter *metrics.Exporter) *Reconciler {
	return &Reconciler{
		amClient:      amClient,
		grafanaClient: grafanaClient,
		metrics:       metricsExporter,
	}
}

// InconsistentAlert represents an alert that exists in Alertmanager but not in Grafana IRM
type InconsistentAlert struct {
	Alert               *models.GettableAlert
	GrafanaAlertGroupID string
	Reason              string
	Fingerprint         string
	Alertname           string
}

// ReconcileAlerts compares alerts between Alertmanager and Grafana IRM
// and identifies inconsistencies that need to be resolved
func (r *Reconciler) ReconcileAlerts(ctx context.Context) ([]InconsistentAlert, error) {
	log.Println("Starting alert reconciliation...")

	// Get firing alert groups from Grafana IRM
	grafanaAlertGroups, err := r.grafanaClient.GetFiringAlertGroups()
	if err != nil {
		return nil, err
	}

	// Get silenced firing alerts from Alertmanager
	silencedAlerts, err := r.amClient.GetSilencedFiringAlerts(ctx)
	if err != nil {
		return nil, err
	}

	log.Printf("Found %d firing alert groups in Grafana IRM", len(grafanaAlertGroups))
	log.Printf("Found %d silenced firing alerts in Alertmanager", len(silencedAlerts))

	// Build a map of alert fingerprints from Grafana IRM for quick lookup
	grafanaFingerprints := make(map[string]string)
	for _, group := range grafanaAlertGroups {
		for _, alert := range group.LastAlert.Payload.Alerts {
			if alert.Fingerprint != "" {
				grafanaFingerprints[alert.Fingerprint] = group.ID
			}
		}
	}

	// Find inconsistencies: alerts silenced in Alertmanager but still firing in Grafana
	var inconsistencies []InconsistentAlert
	for _, alert := range silencedAlerts {
		fingerprint := alert.Fingerprint
		alertname := alert.Labels["alertname"]

		// If alert is silenced in Alertmanager but firing in Grafana IRM, it's inconsistent
		if _, exists := grafanaFingerprints[*fingerprint]; exists {
			inconsistencies = append(inconsistencies, InconsistentAlert{
				Alert:               alert,
				Reason:              "Alert is silenced in Alertmanager but still firing in Grafana IRM",
				Fingerprint:         *fingerprint,
				Alertname:           alertname,
				GrafanaAlertGroupID: grafanaFingerprints[*fingerprint],
			})
		}
	}

	log.Printf("Found %d inconsistent alerts", len(inconsistencies))
	return inconsistencies, nil
}

// ResolveInconsistency handles the resolution of an inconsistent alert
// This function should be called for each alert that needs to be resolved in IRM
func (r *Reconciler) ResolveInconsistency(ctx context.Context, alert InconsistentAlert) error {
	log.Printf("Resolving inconsistency for alert: %s (fingerprint: %s)",
		alert.Alertname, alert.Fingerprint)
	log.Printf("Reason: %s", alert.Reason)

	// Call Grafana API to resolve the alert
	err := r.grafanaClient.ResolveAlertGroup(alert.GrafanaAlertGroupID)
	if err != nil {
		return err
	}

	log.Printf("Successfully resolved alert %s in Grafana IRM", alert.Alertname)

	return nil
}

// ReconcileAndResolve performs a full reconciliation cycle
// It finds inconsistencies and attempts to resolve them
func (r *Reconciler) ReconcileAndResolve(ctx context.Context) error {
	// Record reconciliation start and get completion function
	done := r.metrics.RecordReconciliationStart()
	defer done()

	inconsistencies, err := r.ReconcileAlerts(ctx)
	if err != nil {
		r.metrics.RecordReconciliationFailure()
		return err
	}

	resolvedCount := 0
	for _, inconsistency := range inconsistencies {
		if err := r.ResolveInconsistency(ctx, inconsistency); err != nil {
			log.Printf("Failed to resolve inconsistency for alert %s: %v",
				inconsistency.Alertname, err)
			r.metrics.RecordInconsistencyFailedResolve()
			// Continue with other alerts even if one fails
		} else {
			r.metrics.RecordInconsistencyResolved()
			resolvedCount++
		}
	}

	// Record success with counts
	r.metrics.RecordReconciliationSuccess(len(inconsistencies), resolvedCount)

	return nil
}

// ReconcileAndResolveOptimized performs a full reconciliation cycle with optimized data fetching
// It fetches data from Alertmanager and Grafana once, then processes it in parallel goroutines
func (r *Reconciler) ReconcileAndResolveOptimized(ctx context.Context) error {
	// Record reconciliation start and get completion function
	done := r.metrics.RecordReconciliationStart()
	defer done()

	log.Println("Starting optimized reconciliation with parallel operations...")

	// Fetch data from both sources once
	type fetchResult struct {
		alerts             []*models.GettableAlert
		grafanaAlertGroups []grafana.AlertGroup
		err                error
	}

	alertsChan := make(chan fetchResult, 1)
	grafanaChan := make(chan fetchResult, 1)

	// Fetch Alertmanager alerts in parallel
	go func() {
		alerts, err := r.amClient.GetAllAlerts(ctx)
		alertsChan <- fetchResult{alerts: alerts, err: err}
	}()

	// Fetch Grafana alert groups in parallel
	go func() {
		groups, err := r.grafanaClient.GetAllAlertGroups()
		grafanaChan <- fetchResult{grafanaAlertGroups: groups, err: err}
	}()

	// Wait for both fetches to complete
	alertsResult := <-alertsChan
	grafanaResult := <-grafanaChan

	if alertsResult.err != nil {
		r.metrics.RecordReconciliationFailure()
		return alertsResult.err
	}
	if grafanaResult.err != nil {
		r.metrics.RecordReconciliationFailure()
		return grafanaResult.err
	}

	log.Printf("Fetched %d alerts from Alertmanager", len(alertsResult.alerts))
	log.Printf("Fetched %d alert groups from Grafana", len(grafanaResult.grafanaAlertGroups))

	// Now perform two operations in parallel using the same data
	type operationResult struct {
		name  string
		err   error
		stats map[string]int
	}

	resultsChan := make(chan operationResult, 2)

	// Goroutine 1: Export metrics with Grafana data
	go func() {
		log.Println("Starting metrics export with Grafana data...")
		err := r.metrics.ExportAlertsWithGrafana(ctx, alertsResult.alerts, grafanaResult.grafanaAlertGroups, r.grafanaClient, r.amClient)
		if err != nil {
			log.Printf("Metrics export failed: %v", err)
			r.metrics.RecordAlertExportFailure()
		} else {
			log.Println("Metrics export completed successfully")
		}
		resultsChan <- operationResult{name: "metrics_export", err: err}
	}()

	// Goroutine 2: Reconcile and resolve inconsistencies
	go func() {
		log.Println("Starting silence reconciliation...")
		
		// Filter for silenced firing alerts
		silencedAlerts := make([]*models.GettableAlert, 0)
		for _, alert := range alertsResult.alerts {
			if alert.Status != nil &&
				*alert.Status.State == "suppressed" &&
				len(alert.Status.SilencedBy) > 0 {
				silencedAlerts = append(silencedAlerts, alert)
			}
		}

		log.Printf("Found %d silenced firing alerts", len(silencedAlerts))

		// Build a map of alert fingerprints from Grafana IRM for quick lookup
		grafanaFingerprints := make(map[string]string)
		for _, group := range grafanaResult.grafanaAlertGroups {
			if group.State != "resolved" {
				for _, alert := range group.LastAlert.Payload.Alerts {
					if alert.Fingerprint != "" {
						grafanaFingerprints[alert.Fingerprint] = group.ID
					}
				}
			}
		}

		// Find inconsistencies
		var inconsistencies []InconsistentAlert
		for _, alert := range silencedAlerts {
			fingerprint := alert.Fingerprint
			alertname := alert.Labels["alertname"]

			if _, exists := grafanaFingerprints[*fingerprint]; exists {
				inconsistencies = append(inconsistencies, InconsistentAlert{
					Alert:               alert,
					Reason:              "Alert is silenced in Alertmanager but still firing in Grafana IRM",
					Fingerprint:         *fingerprint,
					Alertname:           alertname,
					GrafanaAlertGroupID: grafanaFingerprints[*fingerprint],
				})
			}
		}

		log.Printf("Found %d inconsistent alerts", len(inconsistencies))

		// Resolve inconsistencies
		resolvedCount := 0
		for _, inconsistency := range inconsistencies {
			if err := r.ResolveInconsistency(ctx, inconsistency); err != nil {
				log.Printf("Failed to resolve inconsistency for alert %s: %v",
					inconsistency.Alertname, err)
				r.metrics.RecordInconsistencyFailedResolve()
			} else {
				r.metrics.RecordInconsistencyResolved()
				resolvedCount++
			}
		}

		stats := map[string]int{
			"inconsistencies": len(inconsistencies),
			"resolved":        resolvedCount,
		}

		resultsChan <- operationResult{name: "silence_reconciliation", stats: stats}
	}()

	// Wait for both operations to complete
	var metricsErr error
	var reconcileStats map[string]int

	for i := 0; i < 2; i++ {
		result := <-resultsChan
		if result.name == "metrics_export" {
			metricsErr = result.err
		} else if result.name == "silence_reconciliation" {
			reconcileStats = result.stats
		}
	}

	// Record reconciliation success
	if metricsErr == nil && reconcileStats != nil {
		r.metrics.RecordReconciliationSuccess(
			reconcileStats["inconsistencies"],
			reconcileStats["resolved"],
		)
		log.Println("Optimized reconciliation completed successfully")
		return nil
	}

	if metricsErr != nil {
		r.metrics.RecordReconciliationFailure()
		return metricsErr
	}

	return nil
}
