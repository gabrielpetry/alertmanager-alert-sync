# Alert Reconciliation Guide

## Overview

Alert reconciliation is a key feature that ensures consistency between Prometheus Alertmanager and Grafana IRM (Incident Response & Management). The service identifies and resolves discrepancies between these two systems.

## How It Works

### The Problem

When alerts are silenced in Alertmanager but remain active (firing) in Grafana IRM, it creates an inconsistency. This typically happens when:

1. An alert fires and creates an incident in Grafana IRM
2. The alert is silenced in Alertmanager (e.g., during maintenance)
3. The silence expires or is removed in Alertmanager
4. The incident in Grafana IRM is never resolved

### The Solution

The reconciliation process:

1. **Fetches firing alert groups** from Grafana IRM
2. **Fetches silenced firing alerts** from Alertmanager
3. **Compares alert fingerprints** between both systems
4. **Identifies inconsistencies** - alerts that are:
   - Silenced in Alertmanager
   - Still firing in Grafana IRM
5. **Automatically resolves** the alerts in Grafana IRM

## Reconciliation Modes

### 1. Automatic Reconciliation (Recommended)

The service runs reconciliation automatically at a configured interval.

**Configuration:**

```bash
export RECONCILE_INTERVAL=300  # Run every 5 minutes
```

**Behavior:**
- Runs immediately when the service starts
- Continues running at the specified interval
- Runs in the background without blocking the HTTP server
- Logs all reconciliation attempts and results

**When to use:**
- Production environments with active alerting
- When you want hands-off alert management
- When you need consistent state between systems

**Recommended intervals:**
- **Production:** 300 seconds (5 minutes)
- **High-volume environments:** 600 seconds (10 minutes)
- **Development/Testing:** 60 seconds (1 minute)
- **Critical systems:** 120 seconds (2 minutes)

### 2. Manual Reconciliation

Trigger reconciliation on-demand via the HTTP endpoint.

**Usage:**

```bash
curl -X POST http://localhost:8080/reconcile
```

**When to use:**
- When you want full control over when reconciliation runs
- For debugging or testing reconciliation logic
- When using external schedulers (cron, Kubernetes CronJob)
- In low-volume environments

## Configuration Examples

### Example 1: Production Setup with Automatic Reconciliation

```bash
# .env or environment variables
ALERTMANAGER_HOST=alertmanager.monitoring.svc.cluster.local:9093
GRAFANA_IRM_URL=https://company.grafana.net
GRAFANA_IRM_TOKEN=glsa_xxxxxxxxxxxxx
RECONCILE_INTERVAL=300
PORT=8080
```

### Example 2: Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alertmanager-alert-sync
spec:
  replicas: 1
  selector:
    matchLabels:
      app: alertmanager-alert-sync
  template:
    metadata:
      labels:
        app: alertmanager-alert-sync
    spec:
      containers:
      - name: sync
        image: alertmanager-alert-sync:latest
        env:
        - name: ALERTMANAGER_HOST
          value: "alertmanager:9093"
        - name: GRAFANA_IRM_URL
          valueFrom:
            secretKeyRef:
              name: grafana-irm
              key: url
        - name: GRAFANA_IRM_TOKEN
          valueFrom:
            secretKeyRef:
              name: grafana-irm
              key: token
        - name: RECONCILE_INTERVAL
          value: "300"
```

### Example 3: Manual with Kubernetes CronJob

If you prefer manual control, disable automatic reconciliation and use a CronJob:

```yaml
# Deployment without RECONCILE_INTERVAL
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alertmanager-alert-sync
spec:
  template:
    spec:
      containers:
      - name: sync
        image: alertmanager-alert-sync:latest
        env:
        - name: ALERTMANAGER_HOST
          value: "alertmanager:9093"
        # RECONCILE_INTERVAL not set - automatic reconciliation disabled

---
# CronJob to trigger reconciliation
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
            - http://alertmanager-alert-sync:8080/reconcile
          restartPolicy: OnFailure
```

## Monitoring Reconciliation

### Logs

The service logs all reconciliation activity:

```
2025-10-30T10:00:00Z Starting reconciliation loop with interval: 5m0s
2025-10-30T10:00:00Z Running scheduled reconciliation...
2025-10-30T10:00:00Z Starting alert reconciliation...
2025-10-30T10:00:01Z Found 5 firing alert groups in Grafana IRM
2025-10-30T10:00:01Z Found 3 silenced firing alerts in Alertmanager
2025-10-30T10:00:01Z Found 2 inconsistent alerts
2025-10-30T10:00:01Z Resolving inconsistency for alert: HighCPUUsage (fingerprint: abc123)
2025-10-30T10:00:02Z Successfully resolved alert HighCPUUsage in Grafana IRM
2025-10-30T10:00:02Z Reconciliation completed successfully
```

### Health Check

The `/health` endpoint always returns the service status. The service remains healthy even if reconciliation fails.

### Debugging

Use the debug endpoints to inspect the current state:

```bash
# View all Alertmanager alerts
curl http://localhost:8080/all-alerts

# View Grafana IRM alerts
curl http://localhost:8080/grafana-alerts

# Trigger manual reconciliation (see logs for results)
curl -X POST http://localhost:8080/reconcile
```

## Best Practices

### Interval Selection

1. **Start conservative:** Begin with longer intervals (5-10 minutes) and adjust based on needs
2. **Consider volume:** High alert volumes may need longer intervals to avoid API rate limits
3. **Monitor API usage:** Check Grafana IRM and Alertmanager API rate limits
4. **Balance timeliness vs. load:** Shorter intervals provide faster resolution but increase load

### Error Handling

The reconciliation process is designed to be resilient:

- If one alert fails to resolve, others continue to process
- API failures are logged but don't crash the service
- The next reconciliation cycle will retry failed operations

### Scaling

For high-availability setups:

- Run multiple replicas with the same `RECONCILE_INTERVAL`
- Each replica will run reconciliation independently
- Grafana IRM API is idempotent, so duplicate resolutions are safe
- Consider using leader election for single-reconciler scenarios (future enhancement)

## Troubleshooting

### Reconciliation not running

**Check the logs on startup:**
```
Background reconciliation enabled with interval: 300 seconds
```

If you see:
```
Background reconciliation disabled (set RECONCILE_INTERVAL to enable)
```

Then `RECONCILE_INTERVAL` is not set or is invalid.

### Reconciliation failing

**Common issues:**

1. **Invalid Grafana IRM credentials**
   - Check `GRAFANA_IRM_URL` and `GRAFANA_IRM_TOKEN`
   - Verify the token has proper permissions

2. **Network connectivity**
   - Ensure the service can reach both Alertmanager and Grafana IRM
   - Check firewall rules and DNS resolution

3. **API rate limits**
   - Reduce `RECONCILE_INTERVAL` (increase the time between runs)
   - Check Grafana IRM API usage

### No inconsistencies found

This is normal! It means:
- Alerts are properly synchronized
- No silenced alerts are lingering in Grafana IRM
- Both systems are in a consistent state

## Future Enhancements

Planned improvements for reconciliation:

- [ ] Metrics for reconciliation operations (success/failure counts)
- [ ] Configurable retry logic with exponential backoff
- [ ] Support for partial reconciliation (by alert severity, namespace, etc.)
- [ ] Leader election for high-availability deployments
- [ ] Webhook notifications for reconciliation events
- [ ] Dry-run mode to preview changes without applying them

## API Reference

### POST /reconcile

Manually trigger a reconciliation cycle.

**Request:**
```bash
curl -X POST http://localhost:8080/reconcile
```

**Response:**
```json
{
  "status": "success",
  "message": "Reconciliation completed",
  "inconsistencies_found": 2,
  "inconsistencies_resolved": 2
}
```

**Status Codes:**
- `200 OK`: Reconciliation completed successfully
- `500 Internal Server Error`: Reconciliation failed

## Support

For issues or questions:

1. Check the logs for detailed error messages
2. Verify your configuration in `.env` or environment variables
3. Test connectivity to Alertmanager and Grafana IRM
4. Open an issue on GitHub with logs and configuration (redact sensitive data)
