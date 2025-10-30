# Metrics Documentation

This document describes all Prometheus metrics exported by the alertmanager-alert-sync service.

## Overview

The service exposes reconciliation metrics at the `/metrics` endpoint. These metrics help you:
- Monitor reconciliation health and success rate
- Track inconsistencies between Alertmanager and Grafana IRM
- Alert on reconciliation failures
- Measure reconciliation performance

## Metrics List

### alertmanager_reconciliation_total

**Type:** Counter

**Description:** Total number of reconciliation attempts since service startup.

**Use cases:**
- Calculate reconciliation rate
- Track total reconciliation activity

**Example queries:**
```promql
# Reconciliation rate per minute
rate(alertmanager_reconciliation_total[5m])

# Total reconciliations in the last hour
increase(alertmanager_reconciliation_total[1h])
```

---

### alertmanager_reconciliation_failures_total

**Type:** Counter

**Description:** Total number of failed reconciliation attempts.

**Use cases:**
- Monitor reconciliation reliability
- Calculate failure rate
- Alert on increasing failures

**Example queries:**
```promql
# Failure rate
rate(alertmanager_reconciliation_failures_total[5m])

# Failure percentage
100 * rate(alertmanager_reconciliation_failures_total[5m]) / rate(alertmanager_reconciliation_total[5m])
```

---

### alertmanager_reconciliation_duration_seconds

**Type:** Histogram

**Description:** Duration of reconciliation operations in seconds.

**Buckets:** Default Prometheus buckets (0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10)

**Use cases:**
- Monitor reconciliation performance
- Identify slow reconciliations
- Calculate percentiles

**Example queries:**
```promql
# Average duration
rate(alertmanager_reconciliation_duration_seconds_sum[5m]) / rate(alertmanager_reconciliation_duration_seconds_count[5m])

# 95th percentile
histogram_quantile(0.95, rate(alertmanager_reconciliation_duration_seconds_bucket[5m]))

# 99th percentile
histogram_quantile(0.99, rate(alertmanager_reconciliation_duration_seconds_bucket[5m]))
```

---

### alertmanager_inconsistencies_found

**Type:** Gauge

**Description:** Number of inconsistencies found in the last reconciliation cycle.

**Use cases:**
- Monitor current inconsistency count
- Alert on high inconsistency levels
- Track inconsistency trends

**Example queries:**
```promql
# Current inconsistencies
alertmanager_inconsistencies_found

# Average over time
avg_over_time(alertmanager_inconsistencies_found[1h])
```

---

### alertmanager_inconsistencies_resolved_total

**Type:** Counter

**Description:** Total number of inconsistencies successfully resolved since service startup.

**Use cases:**
- Track resolution activity
- Calculate resolution rate
- Measure service effectiveness

**Example queries:**
```promql
# Resolution rate per minute
rate(alertmanager_inconsistencies_resolved_total[5m])

# Total resolutions today
increase(alertmanager_inconsistencies_resolved_total[1d])
```

---

### alertmanager_inconsistencies_failed_resolve_total

**Type:** Counter

**Description:** Total number of inconsistencies that failed to resolve.

**Use cases:**
- Monitor resolution failures
- Calculate resolution success rate
- Alert on resolution issues

**Example queries:**
```promql
# Failed resolution rate
rate(alertmanager_inconsistencies_failed_resolve_total[5m])

# Resolution success rate
100 * rate(alertmanager_inconsistencies_resolved_total[5m]) / (rate(alertmanager_inconsistencies_resolved_total[5m]) + rate(alertmanager_inconsistencies_failed_resolve_total[5m]))
```

---

### alertmanager_last_reconciliation_timestamp_seconds

**Type:** Gauge

**Description:** Unix timestamp of the last reconciliation attempt.

**Use cases:**
- Monitor reconciliation recency
- Alert when reconciliation hasn't run
- Verify reconciliation schedule

**Example queries:**
```promql
# Time since last reconciliation (seconds)
time() - alertmanager_last_reconciliation_timestamp_seconds

# Alert if no reconciliation in 10 minutes
(time() - alertmanager_last_reconciliation_timestamp_seconds) > 600
```

---

### alertmanager_last_reconciliation_success

**Type:** Gauge

**Description:** Whether the last reconciliation was successful (1=success, 0=failure).

**Use cases:**
- Monitor current reconciliation health
- Alert on failures
- Dashboard indicators

**Example queries:**
```promql
# Current success status
alertmanager_last_reconciliation_success

# Alert if last reconciliation failed
alertmanager_last_reconciliation_success == 0
```

---

## Grafana Dashboard Examples

### Reconciliation Overview Panel

```json
{
  "targets": [
    {
      "expr": "rate(alertmanager_reconciliation_total[5m])",
      "legendFormat": "Reconciliation Rate"
    },
    {
      "expr": "rate(alertmanager_reconciliation_failures_total[5m])",
      "legendFormat": "Failure Rate"
    }
  ]
}
```

### Current Inconsistencies Gauge

```json
{
  "targets": [
    {
      "expr": "alertmanager_inconsistencies_found",
      "legendFormat": "Inconsistencies"
    }
  ],
  "type": "gauge"
}
```

### Reconciliation Duration Histogram

```json
{
  "targets": [
    {
      "expr": "histogram_quantile(0.50, rate(alertmanager_reconciliation_duration_seconds_bucket[5m]))",
      "legendFormat": "p50"
    },
    {
      "expr": "histogram_quantile(0.95, rate(alertmanager_reconciliation_duration_seconds_bucket[5m]))",
      "legendFormat": "p95"
    },
    {
      "expr": "histogram_quantile(0.99, rate(alertmanager_reconciliation_duration_seconds_bucket[5m]))",
      "legendFormat": "p99"
    }
  ]
}
```

---

## Alert Rules

### Critical Alerts

```yaml
groups:
  - name: alertmanager-sync-critical
    rules:
      - alert: ReconciliationFailing
        expr: alertmanager_last_reconciliation_success == 0
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "Alertmanager reconciliation is failing"
          description: "The last reconciliation failed. Check service logs for details."
          
      - alert: ReconciliationStale
        expr: (time() - alertmanager_last_reconciliation_timestamp_seconds) > 600
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "No reconciliation in the last 10 minutes"
          description: "Reconciliation hasn't run. Check if RECONCILE_INTERVAL is set and service is running."
```

### Warning Alerts

```yaml
groups:
  - name: alertmanager-sync-warnings
    rules:
      - alert: HighInconsistencyRate
        expr: alertmanager_inconsistencies_found > 10
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "High number of inconsistencies detected"
          description: "{{ $value }} inconsistencies found. Investigate alert management practices."
          
      - alert: HighResolutionFailureRate
        expr: |
          rate(alertmanager_inconsistencies_failed_resolve_total[5m]) /
          (rate(alertmanager_inconsistencies_resolved_total[5m]) + rate(alertmanager_inconsistencies_failed_resolve_total[5m])) > 0.1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High resolution failure rate"
          description: "More than 10% of inconsistency resolutions are failing."
          
      - alert: SlowReconciliation
        expr: |
          histogram_quantile(0.95, rate(alertmanager_reconciliation_duration_seconds_bucket[5m])) > 30
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Reconciliation is slow"
          description: "95th percentile reconciliation duration is {{ $value }}s (threshold: 30s)."
```

---

## Monitoring Best Practices

### 1. Set Up Core Alerts

At minimum, configure these alerts:
- **ReconciliationFailing**: Alerts when reconciliation fails
- **ReconciliationStale**: Alerts when reconciliation stops running
- **HighInconsistencyRate**: Alerts on excessive inconsistencies

### 2. Create a Dashboard

Include these panels:
- Reconciliation success rate (last 24h)
- Current inconsistencies gauge
- Reconciliation duration percentiles
- Failed resolutions over time
- Time since last reconciliation

### 3. Monitor Trends

Track these over time:
- Inconsistency count trends
- Resolution success rate
- Reconciliation performance
- Failure patterns

### 4. Baseline Your Environment

Establish normal values for:
- Typical inconsistency count
- Normal reconciliation duration
- Expected failure rate
- Reconciliation interval compliance

### 5. Regular Reviews

Periodically review:
- Alert thresholds
- Metric trends
- Failure patterns
- Performance degradation

---

## Troubleshooting with Metrics

### Reconciliation Not Running

**Symptoms:**
```promql
(time() - alertmanager_last_reconciliation_timestamp_seconds) > 600
```

**Possible causes:**
- `RECONCILE_INTERVAL` not set
- Service crashed
- Configuration error

**Investigation:**
1. Check service logs
2. Verify `RECONCILE_INTERVAL` is set
3. Check `/readyz` endpoint

---

### High Failure Rate

**Symptoms:**
```promql
rate(alertmanager_reconciliation_failures_total[5m]) > 0
```

**Possible causes:**
- Grafana IRM API issues
- Alertmanager connectivity problems
- Authentication failures

**Investigation:**
1. Check service logs for errors
2. Verify Grafana IRM credentials
3. Test connectivity to both services
4. Check API rate limits

---

### Increasing Inconsistencies

**Symptoms:**
```promql
alertmanager_inconsistencies_found > 10
```

**Possible causes:**
- Alerts being silenced frequently
- Grafana IRM not auto-resolving
- Misconfigured alert routing

**Investigation:**
1. Review silencing practices
2. Check Grafana IRM configuration
3. Verify alert fingerprints match
4. Review resolution logs

---

### Slow Reconciliation

**Symptoms:**
```promql
histogram_quantile(0.95, rate(alertmanager_reconciliation_duration_seconds_bucket[5m])) > 30
```

**Possible causes:**
- Large number of alerts
- API rate limiting
- Network latency
- Resource constraints

**Investigation:**
1. Check reconciliation logs
2. Monitor API response times
3. Verify resource allocation
4. Consider increasing interval

---

## Metric Retention

Recommendations for metric retention:
- **Real-time monitoring**: 15 days minimum
- **Historical analysis**: 90+ days recommended
- **Long-term trends**: 1 year for capacity planning

Configure Prometheus retention accordingly:
```yaml
storage:
  tsdb:
    retention.time: 90d
```

---

## Example Prometheus Configuration

Complete scrape configuration:

```yaml
global:
  scrape_interval: 30s
  evaluation_interval: 30s

scrape_configs:
  - job_name: 'alertmanager-sync'
    static_configs:
      - targets: ['alertmanager-sync:8080']
    scrape_interval: 30s
    scrape_timeout: 10s
    
rule_files:
  - 'alertmanager-sync-alerts.yml'
```

---

## Support

For issues with metrics:
1. Verify `/metrics` endpoint is accessible
2. Check Prometheus scrape status
3. Validate metric names in queries
4. Review service logs for export errors
