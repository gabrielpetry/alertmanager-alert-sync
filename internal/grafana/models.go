package grafana

import (
	"encoding/json"
	"time"
)

// NullableTime represents a time that can be null or empty in JSON
type NullableTime struct {
	Time  time.Time
	Valid bool
}

// UnmarshalJSON implements custom JSON unmarshaling for NullableTime
func (nt *NullableTime) UnmarshalJSON(data []byte) error {
	// Handle null
	if string(data) == "null" {
		nt.Valid = false
		return nil
	}

	// Handle empty string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s == "" {
			nt.Valid = false
			return nil
		}
	}

	// Try to parse as time
	var t time.Time
	if err := json.Unmarshal(data, &t); err != nil {
		nt.Valid = false
		return nil
	}

	nt.Time = t
	nt.Valid = true
	return nil
}

// MarshalJSON implements custom JSON marshaling for NullableTime
func (nt NullableTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nt.Time)
}

// User represents a Grafana IRM user
type User struct {
	ID        string `json:"id,omitempty"`
	GrafanaID int    `json:"grafana_id,omitempty"`
	Email     string `json:"email,omitempty"`
	Username  string `json:"username,omitempty"`
}

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
	CreatedAt      NullableTime  `json:"created_at,omitempty"`
	ResolvedAt     NullableTime  `json:"resolved_at,omitempty"`
	ResolvedBy     string        `json:"resolved_by,omitempty"`
	AcknowledgedAt NullableTime  `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string        `json:"acknowledged_by,omitempty"`
	Labels         []interface{} `json:"labels,omitempty"`
	Title          string        `json:"title,omitempty"`
	Permalinks     Permalinks    `json:"permalinks,omitempty"`
	SilencedAt     NullableTime  `json:"silenced_at,omitempty"`
	SilencedBy     string        `json:"silenced_by,omitempty"`
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
	ID           string       `json:"id,omitempty"`
	AlertGroupID string       `json:"alert_group_id,omitempty"`
	CreatedAt    NullableTime `json:"created_at,omitempty"`
	Payload      Payload      `json:"payload,omitempty"`
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
	EndsAt       NullableTime `json:"endsAt,omitempty"`
	Labels       Labels       `json:"labels,omitempty"`
	Status       string       `json:"status,omitempty"`
	StartsAt     NullableTime `json:"startsAt,omitempty"`
	Annotations  Annotations  `json:"annotations,omitempty"`
	Fingerprint  string       `json:"fingerprint,omitempty"`
	GeneratorURL string       `json:"generatorURL,omitempty"`
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
