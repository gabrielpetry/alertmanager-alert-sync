package metrics

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gabrielpetry/alertmanager-alert-sync/internal/alertmanager"
	"github.com/gabrielpetry/alertmanager-alert-sync/internal/grafana"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Exporter handles Prometheus metrics for alert reconciliation
type Exporter struct {
	// Reconciliation metrics
	reconciliationTotal          prometheus.Counter
	reconciliationFailuresTotal  prometheus.Counter
	reconciliationDuration       prometheus.Histogram
	inconsistenciesFound         prometheus.Gauge
	inconsistenciesResolved      prometheus.Counter
	inconsistenciesFailedResolve prometheus.Counter
	lastReconciliationTime       prometheus.Gauge
	lastReconciliationSuccess    prometheus.Gauge

	// Alert state metrics
	alertStateGauge          *prometheus.GaugeVec
	alertExportTotal         prometheus.Counter
	alertExportFailuresTotal prometheus.Counter
	lastAlertExportTime      prometheus.Gauge

	// Configuration for alert labels
	alertLabels      []string
	alertAnnotations []string
}

// NewExporter creates and initializes a new metrics exporter for reconciliation
func NewExporter() *Exporter {
	log.Println("Initializing reconciliation metrics...")

	reconciliationTotal := promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alertmanager_sync_reconciliation_total",
			Help: "Total number of reconciliation attempts",
		},
	)

	reconciliationFailuresTotal := promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alertmanager_sync_reconciliation_failures_total",
			Help: "Total number of failed reconciliation attempts",
		},
	)

	reconciliationDuration := promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "alertmanager_sync_reconciliation_duration_seconds",
			Help:    "Duration of reconciliation operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	inconsistenciesFound := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alertmanager_sync_inconsistencies_found",
			Help: "Number of inconsistencies found in last reconciliation",
		},
	)

	inconsistenciesResolved := promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alertmanager_sync_inconsistencies_resolved_total",
			Help: "Total number of inconsistencies successfully resolved",
		},
	)

	inconsistenciesFailedResolve := promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alertmanager_sync_inconsistencies_failed_resolve_total",
			Help: "Total number of inconsistencies that failed to resolve",
		},
	)

	lastReconciliationTime := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alertmanager_sync_last_reconciliation_timestamp_seconds",
			Help: "Timestamp of the last reconciliation attempt (Unix time)",
		},
	)

	lastReconciliationSuccess := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alertmanager_sync_last_reconciliation_success",
			Help: "Whether the last reconciliation was successful (1=success, 0=failure)",
		},
	)

	// Parse alert labels and annotations from environment
	alertLabels := parseEnvList("ALERTMANAGER_ALERTS_LABELS")
	alertAnnotations := parseEnvList("ALERTMANAGER_ALERTS_ANNOTATIONS")

	// Default labels that are always included
	defaultLabels := []string{"alertname", "fingerprint", "suppressed", "acknowledged_by", "resolved_by", "silenced_by", "inhibited_by"}

	// Combine all labels for the metric
	allLabels := append(defaultLabels, alertLabels...)
	allLabels = append(allLabels, alertAnnotations...)

	log.Printf("Alert export configuration:")
	log.Printf("  - Alert labels to export: %v", alertLabels)
	log.Printf("  - Alert annotations to export: %v", alertAnnotations)
	log.Printf("  - All metric labels: %v", allLabels)

	// Create alert state gauge
	alertStateGauge := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "alertmanager_sync_alert_state",
			Help: "Current state of alerts from Alertmanager (1=active, value indicates if suppressed)",
		},
		allLabels,
	)

	alertExportTotal := promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alertmanager_sync_alert_export_total",
			Help: "Total number of alert export attempts",
		},
	)

	alertExportFailuresTotal := promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alertmanager_sync_alert_export_failures_total",
			Help: "Total number of failed alert export attempts",
		},
	)

	lastAlertExportTime := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alertmanager_sync_last_alert_export_timestamp_seconds",
			Help: "Timestamp of the last alert export (Unix time)",
		},
	)

	return &Exporter{
		reconciliationTotal:          reconciliationTotal,
		reconciliationFailuresTotal:  reconciliationFailuresTotal,
		reconciliationDuration:       reconciliationDuration,
		inconsistenciesFound:         inconsistenciesFound,
		inconsistenciesResolved:      inconsistenciesResolved,
		inconsistenciesFailedResolve: inconsistenciesFailedResolve,
		lastReconciliationTime:       lastReconciliationTime,
		lastReconciliationSuccess:    lastReconciliationSuccess,
		alertStateGauge:              alertStateGauge,
		alertExportTotal:             alertExportTotal,
		alertExportFailuresTotal:     alertExportFailuresTotal,
		lastAlertExportTime:          lastAlertExportTime,
		alertLabels:                  alertLabels,
		alertAnnotations:             alertAnnotations,
	}
}

// parseEnvList parses a comma-separated environment variable into a list of trimmed strings
func parseEnvList(envVar string) []string {
	value := os.Getenv(envVar)
	if value == "" {
		return []string{}
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// RecordReconciliationStart records the start of a reconciliation cycle
func (e *Exporter) RecordReconciliationStart() func() {
	e.reconciliationTotal.Inc()
	e.lastReconciliationTime.SetToCurrentTime()

	startTime := time.Now()

	// Return a function to be called when reconciliation completes
	return func() {
		duration := time.Since(startTime).Seconds()
		e.reconciliationDuration.Observe(duration)
	}
}

// RecordReconciliationSuccess records a successful reconciliation
func (e *Exporter) RecordReconciliationSuccess(inconsistenciesFound, inconsistenciesResolved int) {
	e.lastReconciliationSuccess.Set(1)
	e.inconsistenciesFound.Set(float64(inconsistenciesFound))
	e.inconsistenciesResolved.Add(float64(inconsistenciesResolved))
}

// RecordReconciliationFailure records a failed reconciliation
func (e *Exporter) RecordReconciliationFailure() {
	e.reconciliationFailuresTotal.Inc()
	e.lastReconciliationSuccess.Set(0)
}

// RecordInconsistencyResolved records a successfully resolved inconsistency
func (e *Exporter) RecordInconsistencyResolved() {
	e.inconsistenciesResolved.Inc()
}

// RecordInconsistencyFailedResolve records a failed inconsistency resolution
func (e *Exporter) RecordInconsistencyFailedResolve() {
	e.inconsistenciesFailedResolve.Inc()
}

// ExportAlerts exports the current state of alerts as Prometheus metrics
func (e *Exporter) ExportAlerts(ctx context.Context, alerts []*models.GettableAlert, amClient *alertmanager.Client) error {
	e.alertExportTotal.Inc()
	e.lastAlertExportTime.SetToCurrentTime()

	// Reset previous metrics to avoid stale data
	e.alertStateGauge.Reset()

	for _, alert := range alerts {
		if err := e.exportAlert(ctx, alert, nil, nil, amClient); err != nil {
			log.Printf("Error exporting alert %s: %v", alert.Labels["alertname"], err)
			// Continue with other alerts even if one fails
		}
	}

	return nil
}

// ExportAlertsWithGrafana exports alerts with additional information from Grafana IRM
func (e *Exporter) ExportAlertsWithGrafana(ctx context.Context, alerts []*models.GettableAlert, grafanaAlertGroups []grafana.AlertGroup, grafanaClient *grafana.Client, amClient *alertmanager.Client) error {
	e.alertExportTotal.Inc()
	e.lastAlertExportTime.SetToCurrentTime()

	// Reset previous metrics to avoid stale data
	e.alertStateGauge.Reset()

	// Build a map of alert fingerprints to Grafana alert groups for quick lookup
	grafanaMap := make(map[string]*grafana.AlertGroup)
	for i := range grafanaAlertGroups {
		group := &grafanaAlertGroups[i]
		for _, alert := range group.LastAlert.Payload.Alerts {
			if alert.Fingerprint != "" {
				grafanaMap[alert.Fingerprint] = group
			}
		}
	}

	for _, alert := range alerts {
		var grafanaGroup *grafana.AlertGroup
		if alert.Fingerprint != nil {
			grafanaGroup = grafanaMap[*alert.Fingerprint]
		}

		if err := e.exportAlert(ctx, alert, grafanaGroup, grafanaClient, amClient); err != nil {
			log.Printf("Error exporting alert %s: %v", alert.Labels["alertname"], err)
			// Continue with other alerts even if one fails
		}
	}

	return nil
}

// exportAlert exports a single alert as a Prometheus metric
func (e *Exporter) exportAlert(ctx context.Context, alert *models.GettableAlert, grafanaGroup *grafana.AlertGroup, grafanaClient *grafana.Client, amClient *alertmanager.Client) error {
	// Extract alert fingerprint
	fingerprint := ""
	if alert.Fingerprint != nil {
		fingerprint = *alert.Fingerprint
	}

	// Determine if alert is suppressed (silenced)
	suppressed := "false"
	silencedBy := ""

	if len(alert.Status.SilencedBy) > 0 {
		suppressed = "true"

		// Get the author of the first silence (with caching)
		if amClient != nil {
			silencedBy = amClient.GetSilenceAuthor(ctx, alert.Status.SilencedBy[0])
		}
	}

	// Extract inhibited_by (fingerprint of inhibiting alert)
	inhibitedBy := ""
	if len(alert.Status.InhibitedBy) > 0 {
		// Use the first inhibiting alert's fingerprint
		inhibitedBy = alert.Status.InhibitedBy[0]
	}

	// Extract acknowledged_by and resolved_by from Grafana (user emails)
	acknowledgedBy := ""
	resolvedBy := ""

	if grafanaGroup != nil && grafanaClient != nil {
		// Fetch user emails from user IDs (with caching)
		if grafanaGroup.AcknowledgedBy != "" {
			acknowledgedBy = grafanaClient.GetUserEmail(grafanaGroup.AcknowledgedBy)
		}
		if grafanaGroup.ResolvedBy != "" {
			resolvedBy = grafanaClient.GetUserEmail(grafanaGroup.ResolvedBy)
		}
	}

	// Build metric labels
	metricLabels := prometheus.Labels{
		"alertname":       alert.Labels["alertname"],
		"fingerprint":     fingerprint,
		"suppressed":      suppressed,
		"acknowledged_by": acknowledgedBy,
		"resolved_by":     resolvedBy,
		"silenced_by":     silencedBy,
		"inhibited_by":    inhibitedBy,
	}

	// Add extra labels from alert labels
	for _, label := range e.alertLabels {
		if val, ok := alert.Labels[label]; ok {
			metricLabels[label] = val
		} else {
			metricLabels[label] = ""
		}
	}

	// Add extra labels from alert annotations
	for _, annotation := range e.alertAnnotations {
		if val, ok := alert.Annotations[annotation]; ok {
			metricLabels[annotation] = val
		} else {
			metricLabels[annotation] = ""
		}
	}
	var alertStateNumber float64
	alertStateNumber = 0.0
	// Set the gauge value to 1 (alert firing)
	if *alert.Status.State == "active" {
		alertStateNumber = 1
	}
	// Set the gauge value to 1 (alert exists)
	e.alertStateGauge.With(metricLabels).Set(alertStateNumber)

	return nil
}

// RecordAlertExportFailure increments the alert export failure counter
func (e *Exporter) RecordAlertExportFailure() {
	e.alertExportFailuresTotal.Inc()
}
