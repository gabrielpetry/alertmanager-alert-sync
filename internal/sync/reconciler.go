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
