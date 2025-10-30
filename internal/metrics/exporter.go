package metrics

import (
	"log"
	"time"

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
}

// NewExporter creates and initializes a new metrics exporter for reconciliation
func NewExporter() *Exporter {
	log.Println("Initializing reconciliation metrics...")

	reconciliationTotal := promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alertmanager_reconciliation_total",
			Help: "Total number of reconciliation attempts",
		},
	)

	reconciliationFailuresTotal := promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alertmanager_reconciliation_failures_total",
			Help: "Total number of failed reconciliation attempts",
		},
	)

	reconciliationDuration := promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "alertmanager_reconciliation_duration_seconds",
			Help:    "Duration of reconciliation operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	inconsistenciesFound := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alertmanager_inconsistencies_found",
			Help: "Number of inconsistencies found in last reconciliation",
		},
	)

	inconsistenciesResolved := promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alertmanager_inconsistencies_resolved_total",
			Help: "Total number of inconsistencies successfully resolved",
		},
	)

	inconsistenciesFailedResolve := promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alertmanager_inconsistencies_failed_resolve_total",
			Help: "Total number of inconsistencies that failed to resolve",
		},
	)

	lastReconciliationTime := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alertmanager_last_reconciliation_timestamp_seconds",
			Help: "Timestamp of the last reconciliation attempt (Unix time)",
		},
	)

	lastReconciliationSuccess := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alertmanager_last_reconciliation_success",
			Help: "Whether the last reconciliation was successful (1=success, 0=failure)",
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
	}
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
