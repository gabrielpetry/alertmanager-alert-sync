package grafana

import "time"

// AlertGroupResponse represents the response from Grafana IRM alert groups endpoint
type AlertGroupResponse struct {
	Count             int          `json:"count,omitempty"`
	Next              interface{}  `json:"next,omitempty"`
	Previous          interface{}  `json:"previous,omitempty"`
	Results           []AlertGroup `json:"results,omitempty"`
	PageSize          int          `json:"page_size,omitempty"`
	PageLength        int          `json:"page_length,omitempty"`
	CurrentPageNumber int          `json:"current_page_number,omitempty"`
	TotalPages        int          `json:"total_pages,omitempty"`
}

// AlertGroup represents a group of related alerts in Grafana IRM
type AlertGroup struct {
	ID             string        `json:"id,omitempty"`
	IntegrationID  string        `json:"integration_id,omitempty"`
	TeamID         string        `json:"team_id,omitempty"`
	RouteID        string        `json:"route_id,omitempty"`
	AlertsCount    int           `json:"alerts_count,omitempty"`
	State          string        `json:"state,omitempty"`
	CreatedAt      time.Time     `json:"created_at,omitempty"`
	ResolvedAt     interface{}   `json:"resolved_at,omitempty"`
	ResolvedBy     interface{}   `json:"resolved_by,omitempty"`
	AcknowledgedAt interface{}   `json:"acknowledged_at,omitempty"`
	AcknowledgedBy interface{}   `json:"acknowledged_by,omitempty"`
	Labels         []interface{} `json:"labels,omitempty"`
	Title          string        `json:"title,omitempty"`
	Permalinks     Permalinks    `json:"permalinks,omitempty"`
	SilencedAt     interface{}   `json:"silenced_at,omitempty"`
	LastAlert      LastAlert     `json:"last_alert,omitempty"`
}

// Permalinks contains various URLs to access the alert group
type Permalinks struct {
	Slack    interface{} `json:"slack,omitempty"`
	SlackApp interface{} `json:"slack_app,omitempty"`
	Telegram interface{} `json:"telegram,omitempty"`
	Web      string      `json:"web,omitempty"`
}

// LastAlert contains details of the most recent alert in the group
type LastAlert struct {
	ID           string    `json:"id,omitempty"`
	AlertGroupID string    `json:"alert_group_id,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	Payload      Payload   `json:"payload,omitempty"`
}

// Payload contains the full alert payload from Alertmanager
type Payload struct {
	Alerts            []Alert           `json:"alerts,omitempty"`
	Status            string            `json:"status,omitempty"`
	Version           string            `json:"version,omitempty"`
	GroupKey          string            `json:"groupKey,omitempty"`
	Receiver          string            `json:"receiver,omitempty"`
	NumFiring         int               `json:"numFiring,omitempty"`
	ExternalURL       string            `json:"externalURL,omitempty"`
	GroupLabels       GroupLabels       `json:"groupLabels,omitempty"`
	NumResolved       int               `json:"numResolved,omitempty"`
	CommonLabels      CommonLabels      `json:"commonLabels,omitempty"`
	TruncatedAlerts   int               `json:"truncatedAlerts,omitempty"`
	CommonAnnotations CommonAnnotations `json:"commonAnnotations,omitempty"`
}

// Alert represents a single alert within a group
type Alert struct {
	EndsAt       time.Time   `json:"endsAt,omitempty"`
	Labels       Labels      `json:"labels,omitempty"`
	Status       string      `json:"status,omitempty"`
	StartsAt     time.Time   `json:"startsAt,omitempty"`
	Annotations  Annotations `json:"annotations,omitempty"`
	Fingerprint  string      `json:"fingerprint,omitempty"`
	GeneratorURL string      `json:"generatorURL,omitempty"`
}

// Labels contains alert labels
type Labels struct {
	Cluster         string `json:"cluster,omitempty"`
	Severity        string `json:"severity,omitempty"`
	Alertname       string `json:"alertname,omitempty"`
	Component       string `json:"component,omitempty"`
	ClusterProvider string `json:"cluster_provider,omitempty"`
}

// Annotations contains alert annotations
type Annotations struct {
	SLO         string `json:"slo,omitempty"`
	Runbook     string `json:"runbook,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Urgency     string `json:"urgency,omitempty"`
	Description string `json:"description,omitempty"`
}

// GroupLabels contains labels that group alerts together
type GroupLabels struct {
	Cluster   string `json:"cluster,omitempty"`
	Alertname string `json:"alertname,omitempty"`
	Component string `json:"component,omitempty"`
}

// CommonLabels contains labels common to all alerts in the group
type CommonLabels struct {
	Cluster         string `json:"cluster,omitempty"`
	Severity        string `json:"severity,omitempty"`
	Alertname       string `json:"alertname,omitempty"`
	Component       string `json:"component,omitempty"`
	ClusterProvider string `json:"cluster_provider,omitempty"`
}

// CommonAnnotations contains annotations common to all alerts in the group
type CommonAnnotations struct {
	SLO         string `json:"slo,omitempty"`
	Runbook     string `json:"runbook,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Urgency     string `json:"urgency,omitempty"`
	Description string `json:"description,omitempty"`
}
