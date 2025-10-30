# Automatic Reconciliation Feature - Implementation Summary

## Overview

Added automatic reconciliation feature that runs the alert reconciliation process on a configurable interval in the background.

## What Changed

### 1. Core Implementation

**File: `cmd/alertmanager-alert-sync/main.go`**

Added:
- Import of `context`, `strconv`, and `time` packages
- `startReconciliationLoop()` function - runs reconciliation in a background goroutine
- `runReconciliation()` function - executes a single reconciliation cycle with error handling
- Configuration logic to parse `RECONCILE_INTERVAL` environment variable
- Automatic startup of reconciliation loop if interval is configured

**Behavior:**
- Reconciliation runs immediately on startup (if enabled)
- Continues running at the specified interval
- Runs in a separate goroutine (non-blocking)
- Errors are logged but don't crash the service
- Invalid interval values are rejected with clear error messages

### 2. Documentation

**Updated files:**
- `README.md` - Added `RECONCILE_INTERVAL` to environment variables table
- `README.md` - Updated reconciliation section with automatic mode
- `README.md` - Added quick reference for the new feature
- `.env.example` - Added `RECONCILE_INTERVAL` configuration with comments
- `Makefile` - Added `RECONCILE_INTERVAL` to docker-run target

**New files:**
- `docs/RECONCILIATION.md` - Comprehensive guide covering:
  - How reconciliation works
  - Automatic vs. manual modes
  - Configuration examples
  - Best practices
  - Troubleshooting
  - Monitoring

### 3. Kubernetes Support

**File: `kubernetes/bundle.yaml`**

Added `RECONCILE_INTERVAL` to ConfigMap with:
- Default value of 300 seconds (5 minutes)
- Inline documentation
- Recommendation for production use

### 4. Testing Support

**New file: `scripts/test-reconciliation.sh`**

Helper script for testing the feature locally:
- Sets up test environment variables
- Shows expected log messages
- Provides clear feedback

## Environment Variable

```bash
RECONCILE_INTERVAL=<seconds>
```

**Values:**
- `> 0`: Enable automatic reconciliation with specified interval (in seconds)
- `0` or unset: Disable automatic reconciliation (use manual `/reconcile` endpoint)

**Recommended intervals:**
- **Production:** 300 (5 minutes)
- **High-volume:** 600 (10 minutes)
- **Development:** 60 (1 minute)
- **Critical systems:** 120 (2 minutes)

## Usage Examples

### Example 1: Local Development

```bash
export ALERTMANAGER_HOST="localhost:9093"
export GRAFANA_IRM_URL="https://your-instance.grafana.net"
export GRAFANA_IRM_TOKEN="your-token"
export RECONCILE_INTERVAL="60"  # Run every minute for testing

./alertmanager-alert-sync
```

Expected logs:
```
Starting Alertmanager Alert Sync...
Grafana IRM integration enabled
Background reconciliation enabled with interval: 60 seconds
Server listening on port :8080
Starting reconciliation loop with interval: 1m0s
Running scheduled reconciliation...
Starting alert reconciliation...
Found X firing alert groups in Grafana IRM
Found Y silenced firing alerts in Alertmanager
Found Z inconsistent alerts
Reconciliation completed successfully
```

### Example 2: Docker

```bash
docker run -p 8080:8080 \
  -e ALERTMANAGER_HOST=alertmanager:9093 \
  -e GRAFANA_IRM_URL=https://your-instance.grafana.net \
  -e GRAFANA_IRM_TOKEN=your-token \
  -e RECONCILE_INTERVAL=300 \
  alertmanager-alert-sync:latest
```

### Example 3: Kubernetes

```yaml
env:
- name: RECONCILE_INTERVAL
  value: "300"
```

The ConfigMap in `kubernetes/bundle.yaml` already includes this.

### Example 4: Disable Automatic Reconciliation

Simply don't set `RECONCILE_INTERVAL` or set it to `0`:

```bash
# Don't set RECONCILE_INTERVAL
./alertmanager-alert-sync

# Or explicitly disable
export RECONCILE_INTERVAL="0"
./alertmanager-alert-sync
```

Then use manual reconciliation:
```bash
curl -X POST http://localhost:8080/reconcile
```

## Technical Details

### Goroutine Management

- Reconciliation runs in a separate goroutine spawned at startup
- Uses `time.NewTicker()` for precise interval timing
- Goroutine lifecycle is tied to application lifecycle (will exit when app exits)
- No explicit shutdown handling needed (ticker cleanup via defer)

### Error Handling

- Invalid intervals (non-numeric, negative) are rejected with clear error messages
- API failures during reconciliation are logged but don't crash the service
- Each reconciliation cycle is independent (failures don't affect subsequent runs)
- Individual alert resolution failures don't stop processing of other alerts

### Concurrency

- HTTP server and reconciliation loop run concurrently
- Manual `/reconcile` endpoint can still be called while automatic reconciliation is running
- Both use the same reconciler instance (no state conflicts)
- Grafana IRM API is idempotent (duplicate resolutions are safe)

## Testing

### Unit Testing

The core reconciliation logic in `internal/sync/reconciler.go` can be tested independently.

### Integration Testing

Use the provided test script:
```bash
./scripts/test-reconciliation.sh
```

### Manual Testing

1. Start the service with a short interval (e.g., 60 seconds)
2. Watch the logs for reconciliation cycles
3. Check `/all-alerts` and `/grafana-alerts` endpoints for current state
4. Verify that inconsistencies are detected and resolved

## Monitoring

### Logs

All reconciliation activity is logged:
- Loop startup
- Each reconciliation cycle
- Number of inconsistencies found
- Resolution attempts and results
- Errors and failures

### Health Check

The `/health` endpoint continues to work normally. The service remains healthy even if reconciliation fails.

### Future Metrics

Planned Prometheus metrics for reconciliation:
- `alertmanager_sync_reconciliation_total` - Total reconciliation attempts
- `alertmanager_sync_reconciliation_failures_total` - Failed reconciliations
- `alertmanager_sync_inconsistencies_found` - Number of inconsistencies detected
- `alertmanager_sync_inconsistencies_resolved` - Number of inconsistencies resolved
- `alertmanager_sync_reconciliation_duration_seconds` - Time taken per reconciliation

## Benefits

1. **Automation**: No need for external cron jobs or manual intervention
2. **Simplicity**: Single configuration variable to enable
3. **Reliability**: Built-in error handling and logging
4. **Flexibility**: Can still use manual reconciliation when needed
5. **Performance**: Non-blocking background operation
6. **Observability**: Comprehensive logging of all operations

## Backward Compatibility

- Existing deployments without `RECONCILE_INTERVAL` continue to work unchanged
- Manual `/reconcile` endpoint remains available
- No breaking changes to existing functionality
- New feature is opt-in via configuration

## Future Enhancements

Potential improvements:
- [ ] Add Prometheus metrics for reconciliation operations
- [ ] Support for leader election in multi-replica deployments
- [ ] Configurable retry logic with exponential backoff
- [ ] Dry-run mode to preview changes without applying them
- [ ] Webhook notifications for reconciliation events
- [ ] Filtering reconciliation by alert severity or namespace

## Deployment Checklist

When deploying with automatic reconciliation:

- [ ] Set `RECONCILE_INTERVAL` to appropriate value for your environment
- [ ] Ensure `GRAFANA_IRM_URL` and `GRAFANA_IRM_TOKEN` are configured
- [ ] Verify network connectivity to both Alertmanager and Grafana IRM
- [ ] Check logs for successful reconciliation loop startup
- [ ] Monitor logs for reconciliation cycles and results
- [ ] Verify inconsistencies are being detected and resolved
- [ ] Adjust interval based on volume and API rate limits

## Support

For issues or questions:
1. Check logs for error messages
2. Verify configuration values
3. Test connectivity to external services
4. Review [docs/RECONCILIATION.md](../docs/RECONCILIATION.md)
5. Open an issue on GitHub
