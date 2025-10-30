# Reconciliation Flow Diagram

## Automatic Reconciliation Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Application Startup                          │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ├──► Parse RECONCILE_INTERVAL env var
                             │
                             ▼
                    ┌────────────────┐
                    │ Is Interval Set │
                    │  and Valid?     │
                    └────────┬────────┘
                             │
               ┌─────────────┴─────────────┐
               │                           │
          YES  │                           │  NO
               ▼                           ▼
    ┌──────────────────┐          ┌──────────────────┐
    │ Start Background │          │   Skip Auto      │
    │ Reconciliation   │          │  Reconciliation  │
    │    Goroutine     │          └──────────────────┘
    └────────┬─────────┘                  │
             │                            │
             ▼                            ▼
    ┌──────────────────┐          ┌──────────────────┐
    │ Run Immediately  │          │  Manual Mode     │
    │   on Startup     │          │ (via /reconcile) │
    └────────┬─────────┘          └──────────────────┘
             │
             ▼
    ┌──────────────────┐
    │  Start Ticker    │
    │ (Interval-based) │
    └────────┬─────────┘
             │
             │ Every RECONCILE_INTERVAL seconds
             ▼
    ┌──────────────────────────────────────────────┐
    │         Reconciliation Cycle                 │
    │                                              │
    │  1. Fetch Grafana IRM firing alerts         │
    │  2. Fetch Alertmanager silenced alerts      │
    │  3. Compare fingerprints                    │
    │  4. Identify inconsistencies                │
    │  5. Resolve each inconsistent alert         │
    │  6. Log results                             │
    └──────────────────┬───────────────────────────┘
                       │
                       ├──► Success: Log completion
                       │
                       └──► Failure: Log error, continue
                       │
                       └──► Wait for next interval
```

## Reconciliation Process Detail

```
┌──────────────────────────────────────────────────────────────┐
│                  Reconciliation Cycle                         │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
            ┌──────────────────────┐
            │ Get Grafana IRM      │
            │ Firing Alert Groups  │
            └──────────┬───────────┘
                       │
                       ▼
            ┌──────────────────────┐
            │ Get Alertmanager     │
            │ Silenced Firing      │
            │ Alerts               │
            └──────────┬───────────┘
                       │
                       ▼
            ┌──────────────────────┐
            │ Build Fingerprint    │
            │ Map from Grafana     │
            └──────────┬───────────┘
                       │
                       ▼
            ┌──────────────────────┐
            │ For Each Silenced    │
            │ Alert                │
            └──────────┬───────────┘
                       │
                       ▼
         ┌─────────────────────────┐
         │ Exists in Grafana IRM?  │
         └─────────┬───────────────┘
                   │
         ┌─────────┴─────────┐
         │                   │
    YES  │                   │  NO
         ▼                   ▼
┌─────────────────┐   ┌─────────────┐
│ Add to          │   │ Skip        │
│ Inconsistencies │   │ (Consistent)│
└────────┬────────┘   └─────────────┘
         │
         ▼
┌─────────────────┐
│ For Each        │
│ Inconsistency   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Call Grafana    │
│ IRM Resolve API │
└────────┬────────┘
         │
         ├──► Success: Log resolution
         │
         └──► Failure: Log error, continue to next
```

## Component Interaction

```
┌─────────────────┐       ┌──────────────────┐       ┌─────────────────┐
│                 │       │                  │       │                 │
│  Alertmanager   │◄──────│   Reconciler     │──────►│  Grafana IRM    │
│                 │       │                  │       │                 │
└─────────────────┘       └────────┬─────────┘       └─────────────────┘
                                   │
                                   │
                                   ▼
                          ┌────────────────┐
                          │                │
                          │  Background    │
                          │  Goroutine     │
                          │                │
                          └────────────────┘
                                   │
                                   │
                          ┌────────▼────────┐
                          │                 │
                          │  Main Process   │
                          │  (HTTP Server)  │
                          │                 │
                          └─────────────────┘
```

## State Flow

```
┌──────────────────────────────────────────────────────────┐
│                     Alert States                          │
└──────────────────────────────────────────────────────────┘

Alert Fires
    │
    ├──► Creates Incident in Grafana IRM
    │
    ├──► Alert Shown in Alertmanager
    │
    └──► Alert Gets Silenced in Alertmanager
         │
         ├──► Incident Still Active in Grafana IRM
         │    (INCONSISTENT STATE)
         │
         └──► Reconciler Detects Inconsistency
              │
              └──► Resolves Incident in Grafana IRM
                   │
                   └──► CONSISTENT STATE
```

## Timing Diagram

```
Time    Main Process    Reconciliation Loop    External Systems
─────   ────────────   ───────────────────    ───────────────

T+0s    Start App      
        │              
        └─────────────► Start Loop
                        │
                        └──► Run Immediately
                             │
                             ├──► Query Alertmanager
                             │
                             ├──► Query Grafana IRM
                             │
                             ├──► Detect Inconsistencies
                             │
                             └──► Resolve Alerts
                                  │
T+Xs                              Wait for Interval...
                                  │
T+Xs    Handle /metrics  
        │              
T+Ys                              Run Again
                                  │
                                  ├──► Query Alertmanager
                                  │
                                  └──► Query Grafana IRM
                                       │
                                       Wait for Interval...

(X = RECONCILE_INTERVAL in seconds)
(Y = Time until next cycle)
```

## Configuration Flow

```
Environment Variables
        │
        ├──► RECONCILE_INTERVAL (required for auto mode)
        │
        ├──► GRAFANA_IRM_URL (required)
        │
        └──► GRAFANA_IRM_TOKEN (required)
             │
             ▼
    ┌────────────────┐
    │  Application   │
    │  Startup       │
    └────────┬───────┘
             │
             ├──► Parse & Validate
             │
             ▼
    ┌─────────────────┐
    │ Valid Config?   │
    └────────┬────────┘
             │
    ┌────────┴────────┐
    │                 │
YES │                 │ NO
    ▼                 ▼
Enable Auto    Disable Auto
Reconciliation Reconciliation
    │                 │
    └────────┬────────┘
             │
             ▼
    Start Application
```
