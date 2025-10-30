# Alertmanager Alert Sync

A Go service that automatically reconciles alerts between Prometheus Alertmanager and Grafana IRM (Incident Response & Management), ensuring consistency between both systems with comprehensive Prometheus metrics for monitoring.

## Features

- **Automatic Reconciliation**: Continuously reconciles alerts on a configurable interval
- **Alert State Export**: Automatically exports all alert states as Prometheus metrics with suppression status
- **Inconsistency Detection**: Finds alerts silenced in Alertmanager but still firing in Grafana IRM
- **Automatic Resolution**: Resolves inconsistent alerts in Grafana IRM
- **Comprehensive Metrics**: Exposes detailed Prometheus metrics for monitoring reconciliation health and alert states
- **Flexible Label Configuration**: Choose which alert labels and annotations to export as metric labels
- **Manual Trigger**: Optional HTTP endpoint to trigger immediate reconciliation
- **Kubernetes-Ready**: Includes liveness and readiness probes
- **Production-Ready**: Built with error handling, logging, and observability in mind

📖 **Documentation:**
- **[Reconciliation Guide](docs/RECONCILIATION.md)** - Learn about automatic reconciliation, best practices, and configuration
- **[Metrics Documentation](docs/METRICS.md)** - Complete guide to all Prometheus metrics, queries, and alerts

## Architecture

The application is structured following Go best practices:

```
.
├── cmd/
│   └── alertmanager-alert-sync/    # Application entry point
│       └── main.go
├── internal/
│   ├── alertmanager/               # Alertmanager client
│   │   └── client.go
│   ├── grafana/                    # Grafana IRM client
│   │   ├── client.go
│   │   └── models.go
│   ├── metrics/                    # Prometheus metrics exporter
│   │   └── exporter.go
│   ├── server/                     # HTTP server and handlers
│   │   └── handlers.go
│   └── sync/                       # Reconciliation logic
│       └── reconciler.go
├── kubernetes/                     # Kubernetes deployment manifests
│   └── bundle.yaml
├── Dockerfile
├── go.mod
└── README.md
```

## Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ALERTMANAGER_HOST` | Alertmanager host:port | `localhost:9093` | No |
| `GRAFANA_IRM_URL` | Grafana IRM base URL | - | Yes |
| `GRAFANA_IRM_TOKEN` | Grafana IRM API token | - | Yes |
| `RECONCILE_INTERVAL` | Automatic reconciliation interval in seconds (0 or unset = disabled) | - | No |
| `ALERT_EXPORT_INTERVAL` | Alert state export interval in seconds (0 or unset = disabled) | - | No |
| `ALERTMANAGER_ALERTS_LABELS` | Comma-separated list of alert labels to export | - | No |
| `ALERTMANAGER_ALERTS_ANNOTATIONS` | Comma-separated list of alert annotations to export | - | No |
| `PORT` | HTTP server port | `8080` | No |

## Quick Reference

```bash
# Enable automatic reconciliation every 5 minutes
export RECONCILE_INTERVAL=300

# Or trigger manually
curl -X POST http://localhost:8080/reconcile
```

See the [Reconciliation Guide](docs/RECONCILIATION.md) for detailed configuration options.

## Quick Start

### Local Development

1. **Set up environment variables:**

```bash
export ALERTMANAGER_HOST="localhost:9093"
export GRAFANA_IRM_URL="https://your-grafana-irm.com"
export GRAFANA_IRM_TOKEN="your-api-token"
export RECONCILE_INTERVAL="300"  # Run reconciliation every 5 minutes
export ALERT_EXPORT_INTERVAL="30"  # Export alert states every 30 seconds
export ALERTMANAGER_ALERTS_LABELS="severity,cluster,namespace"
export ALERTMANAGER_ALERTS_ANNOTATIONS="summary,description"
```

2. **Run the application:**

```bash
go run cmd/alertmanager-alert-sync/main.go
```

3. **Access the endpoints:**

- Metrics: `http://localhost:8080/metrics`
- Liveness: `http://localhost:8080/healthz`
- Readiness: `http://localhost:8080/readyz`
- Manual Reconcile: `http://localhost:8080/reconcile`

### Docker

1. **Build the image:**

```bash
docker build -t alertmanager-alert-sync .
```

2. **Run the container:**

```bash
docker run -p 8080:8080 \
  -e ALERTMANAGER_HOST="alertmanager:9093" \
  -e GRAFANA_IRM_URL="https://your-grafana-irm.com" \
  -e GRAFANA_IRM_TOKEN="your-api-token" \
  alertmanager-alert-sync
```

### Kubernetes

```bash
kubectl apply -f kubernetes/bundle.yaml
```

Make sure to update the ConfigMap and Secret in `kubernetes/bundle.yaml` with your configuration.

## API Endpoints

### `/metrics`

Returns Prometheus metrics for the reconciliation process.

**Reconciliation Metrics:**

- `alertmanager_sync_reconciliation_total`: Total number of reconciliation attempts
- `alertmanager_sync_reconciliation_failures_total`: Total number of failed reconciliations
- `alertmanager_sync_reconciliation_duration_seconds`: Histogram of reconciliation durations
- `alertmanager_sync_inconsistencies_found`: Number of inconsistencies found in last reconciliation
- `alertmanager_sync_inconsistencies_resolved_total`: Total number of inconsistencies successfully resolved
- `alertmanager_sync_inconsistencies_failed_resolve_total`: Total number of inconsistencies that failed to resolve
- `alertmanager_sync_last_reconciliation_timestamp_seconds`: Timestamp of the last reconciliation (Unix time)
- `alertmanager_sync_last_reconciliation_success`: Whether the last reconciliation was successful (1=success, 0=failure)

**Alert State Metrics:**

- `alertmanager_sync_alert_state`: Current state of each alert (labels: alertname, alertstate, suppressed, plus configured labels/annotations)
- `alertmanager_sync_alert_export_total`: Total number of alert export attempts
- `alertmanager_sync_alert_export_failures_total`: Total number of failed alert exports
- `alertmanager_sync_last_alert_export_timestamp_seconds`: Timestamp of the last alert export (Unix time)

**Example Prometheus queries:**

```promql
# Reconciliation success rate
rate(alertmanager_sync_reconciliation_total[5m]) - rate(alertmanager_sync_reconciliation_failures_total[5m])

# Average reconciliation duration
rate(alertmanager_sync_reconciliation_duration_seconds_sum[5m]) / rate(alertmanager_sync_reconciliation_duration_seconds_count[5m])

# Current inconsistencies
alertmanager_sync_inconsistencies_found

# Time since last successful reconciliation
time() - alertmanager_sync_last_reconciliation_timestamp_seconds

# Count of active alerts by severity
sum(alertmanager_sync_alert_state{alertstate="active"}) by (severity)

# Count of suppressed alerts
sum(alertmanager_sync_alert_state{suppressed="true"})

# Alerts by state
sum(alertmanager_sync_alert_state) by (alertstate, suppressed)
```

### `/reconcile`

Manually triggers a reconciliation between Alertmanager and Grafana IRM. 

**When automatic reconciliation is enabled** via `RECONCILE_INTERVAL`, this endpoint can still be used to trigger immediate reconciliation without waiting for the next scheduled run.

Returns `200 OK` with a success message, or `500` if reconciliation fails.

### `/healthz`

Kubernetes-style liveness probe. Returns `200 OK` if the service process is running.

Use this for:
- Kubernetes `livenessProbe`
- Basic health monitoring
- Process restart decisions

### `/readyz`

Kubernetes-style readiness probe. Returns `200 OK` if the service is ready to handle traffic (reconciler initialized with Grafana client).

Returns `503 Service Unavailable` if:
- Grafana IRM client failed to initialize
- Service is not ready to reconcile alerts

Use this for:
- Kubernetes `readinessProbe`
- Load balancer health checks
- Service mesh readiness

## Use Cases

### 1. Metrics and Monitoring

Scrape the `/metrics` endpoint with Prometheus to:
- Monitor reconciliation success/failure rates
- Track inconsistencies found and resolved
- Alert on reconciliation failures
- Monitor reconciliation performance

**Prometheus configuration:**

```yaml
scrape_configs:
  - job_name: 'alertmanager-sync'
    static_configs:
      - targets: ['alertmanager-sync:8080']
    scrape_interval: 30s
```

**Example alerts:**

```yaml
groups:
  - name: alertmanager-sync
    rules:
      - alert: ReconciliationFailing
        expr: alertmanager_sync_last_reconciliation_success == 0
        for: 10m
        annotations:
          summary: "Alert reconciliation is failing"
          
      - alert: HighInconsistencyRate
        expr: alertmanager_sync_inconsistencies_found > 10
        for: 5m
        annotations:
          summary: "High number of inconsistencies detected"
          
      - alert: ReconciliationStale
        expr: (time() - alertmanager_sync_last_reconciliation_timestamp_seconds) > 600
        for: 5m
        annotations:
          summary: "No reconciliation in the last 10 minutes"
```

### 2. Alert State Monitoring

The service can automatically export all alert states from Alertmanager as Prometheus metrics on a regular interval.

**Automatic alert export:**

Set the `ALERT_EXPORT_INTERVAL` environment variable to enable:

```bash
export ALERT_EXPORT_INTERVAL="30"  # Export every 30 seconds
export ALERTMANAGER_ALERTS_LABELS="severity,cluster,namespace"
export ALERTMANAGER_ALERTS_ANNOTATIONS="summary,description"
```

The alert export will:
- Run immediately on startup
- Continue running at the specified interval
- Export each alert with its state (active, suppressed, etc.)
- Include custom labels and annotations as metric labels
- Provide visibility into suppressed/silenced alerts

**Use cases:**
- Monitor which alerts are currently suppressed
- Track alert counts by severity, cluster, or any custom label
- Create dashboards showing alert distribution
- Alert on excessive suppressions

### 3. Alert Reconciliation

The service can automatically reconcile alerts between Alertmanager and Grafana IRM on a regular interval.

**Automatic reconciliation (recommended):**

Set the `RECONCILE_INTERVAL` environment variable to enable automatic reconciliation:

```bash
export RECONCILE_INTERVAL="300"  # Run every 5 minutes
```

The reconciliation will:
- Run immediately on startup
- Continue running at the specified interval
- Identify alerts that need attention in Grafana IRM
- Auto-resolve alerts that have been silenced
- Maintain consistency between monitoring systems

**Manual reconciliation:**

You can also trigger reconciliation manually via the `/reconcile` endpoint:

```bash
curl -X POST http://localhost:8080/reconcile
```

**Example cron job (alternative to automatic reconciliation):**

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: alert-reconciliation
spec:
  schedule: "*/5 * * * *"  # Every 5 minutes
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: reconcile
            image: curlimages/curl:latest
            command:
            - curl
            - -X
            - POST
            - http://alertmanager-sync:8080/reconcile
          restartPolicy: OnFailure
```

## Development

### Project Structure

- **cmd/**: Application entry points
- **internal/**: Private application code
  - **alertmanager/**: Alertmanager API client
  - **grafana/**: Grafana IRM API client and models
  - **metrics/**: Prometheus metrics handling
  - **server/**: HTTP handlers and routing
  - **sync/**: Reconciliation and synchronization logic

### Adding New Features

1. **New metrics**: Add to `internal/metrics/exporter.go`
2. **New API endpoints**: Add handlers to `internal/server/handlers.go`
3. **Reconciliation logic**: Implement in `internal/sync/reconciler.go`
4. **Client extensions**: Extend clients in `internal/alertmanager/` or `internal/grafana/`

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

## TODO

- [ ] Implement automatic alert resolution in Grafana IRM (see `internal/sync/reconciler.go`)
- [ ] Add retry logic for API calls
- [ ] Add unit tests
- [ ] Add integration tests
- [ ] Add support for alert silencing via API
- [ ] Add support for custom alert grouping
- [ ] Add metrics for reconciliation operations

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes following Go best practices
4. Add tests for new functionality
5. Submit a pull request

## License

See [LICENSE](LICENSE) file for details.

## Troubleshooting

### Grafana IRM connection issues

If you see `Grafana client initialization failed`, check:
- `GRAFANA_IRM_URL` is correct and accessible
- `GRAFANA_IRM_TOKEN` is valid and has proper permissions
- Network connectivity to Grafana IRM

The service will continue to work without Grafana features if the connection fails.

### Metrics not updating

- Check Alertmanager connectivity via `/all-alerts` endpoint
- Verify `ALERTMANAGER_HOST` is correct
- Check logs for sync failures

### Missing labels in metrics

- Verify label names in `ALERT_LABELS` match actual alert labels
- Check that alerts actually have the requested labels
- Labels not present on alerts will be exported as empty strings
