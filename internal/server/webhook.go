package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gabrielpetry/alertmanager-alert-sync/internal/alertmanager"
	"github.com/gabrielpetry/alertmanager-alert-sync/internal/grafana"
	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/models"
)

// WebhookEvent represents the incoming webhook payload from Grafana IRM
type WebhookEvent struct {
	Event struct {
		Type  string `json:"type"`
		Time  string `json:"time"`
		Until string `json:"until"`
	} `json:"event"`
	User struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"user"`
	AlertGroup struct {
		ID             string                 `json:"id"`
		IntegrationID  string                 `json:"integration_id"`
		TeamID         string                 `json:"team_id"`
		RouteID        string                 `json:"route_id"`
		AlertsCount    int                    `json:"alerts_count"`
		State          string                 `json:"state"`
		CreatedAt      string                 `json:"created_at"`
		ResolvedAt     *string                `json:"resolved_at"`
		ResolvedBy     *string                `json:"resolved_by"`
		AcknowledgedAt *string                `json:"acknowledged_at"`
		AcknowledgedBy *string                `json:"acknowledged_by"`
		Labels         map[string]interface{} `json:"labels"`
		Title          string                 `json:"title"`
		Permalinks     struct {
			Slack    *string `json:"slack"`
			SlackApp *string `json:"slack_app"`
			Telegram *string `json:"telegram"`
			Web      string  `json:"web"`
		} `json:"permalinks"`
		SilencedAt string `json:"silenced_at"`
		LastAlert  struct {
			ID           string `json:"id"`
			AlertGroupID string `json:"alert_group_id"`
			CreatedAt    string `json:"created_at"`
			Payload      struct {
				Alerts []struct {
					EndsAt       string            `json:"endsAt"`
					Labels       map[string]string `json:"labels"`
					Status       string            `json:"status"`
					StartsAt     string            `json:"startsAt"`
					Annotations  map[string]string `json:"annotations"`
					Fingerprint  string            `json:"fingerprint"`
					GeneratorURL string            `json:"generatorURL"`
				} `json:"alerts"`
				Status            string            `json:"status"`
				Version           string            `json:"version"`
				GroupKey          string            `json:"groupKey"`
				Receiver          string            `json:"receiver"`
				NumFiring         int               `json:"numFiring"`
				ExternalURL       string            `json:"externalURL"`
				GroupLabels       map[string]string `json:"groupLabels"`
				NumResolved       int               `json:"numResolved"`
				CommonLabels      map[string]string `json:"commonLabels"`
				TruncatedAlerts   int               `json:"truncatedAlerts"`
				CommonAnnotations map[string]string `json:"commonAnnotations"`
			} `json:"payload"`
		} `json:"last_alert"`
		ResolutionNotes []struct {
			ID        string `json:"id"`
			Author    string `json:"author"`
			CreatedAt string `json:"created_at"`
			Text      string `json:"text"`
		} `json:"resolution_notes"`
	} `json:"alert_group"`
	AlertGroupID string `json:"alert_group_id"`
}

// WebhookHandler handles incoming webhook requests from Grafana IRM
type WebhookHandler struct {
	amClient      *alertmanager.Client
	grafanaClient *grafana.Client
	username      string
	password      string
	allowlist     map[string]bool
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(amClient *alertmanager.Client, grafanaClient *grafana.Client) *WebhookHandler {
	username := os.Getenv("WEBHOOK_USERNAME")
	password := os.Getenv("WEBHOOK_PASSWORD")
	allowlistEnv := os.Getenv("WEBHOOK_EMAIL_ALLOWLIST")

	if username == "" || password == "" {
		log.Fatal("WEBHOOK_USERNAME and WEBHOOK_PASSWORD environment variables must be set")
	}

	allowlist := make(map[string]bool)
	if allowlistEnv != "" {
		emails := strings.Split(allowlistEnv, ",")
		for _, email := range emails {
			allowlist[strings.TrimSpace(email)] = true
		}
	}

	log.Printf("Webhook handler initialized with %d allowed emails", len(allowlist))

	return &WebhookHandler{
		amClient:      amClient,
		grafanaClient: grafanaClient,
		username:      username,
		password:      password,
		allowlist:     allowlist,
	}
}

// basicAuth validates the basic authentication credentials
func (h *WebhookHandler) basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != h.username || password != h.password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// HandleWebhook processes incoming webhook events
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	var event WebhookEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Printf("Failed to decode webhook payload: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Ignore if event.type does not exist or is empty
	if event.Event.Type == "" {
		log.Println("Ignoring webhook event: event.type is empty")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ignored", "reason": "no event type"})
		return
	}

	// Only process silence events
	if event.Event.Type != "silence" {
		log.Printf("Ignoring webhook event: type is %s (not silence)", event.Event.Type)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ignored", "reason": "not a silence event"})
		return
	}

	log.Printf("Processing silence event for alert group %s by user %s", event.AlertGroup.ID, event.User.Email)

	// Check if user email is in allowlist
	isAllowed := h.allowlist[event.User.Email]

	if !isAllowed {
		// User NOT in allowlist - unsilence the alert in Grafana
		log.Printf("User %s not in allowlist, unsilencing alert group %s in Grafana", event.User.Email, event.AlertGroup.ID)
		if err := h.grafanaClient.UnsilenceAlertGroup(event.AlertGroup.ID); err != nil {
			log.Printf("Failed to unsilence alert group %s: %v", event.AlertGroup.ID, err)
			http.Error(w, fmt.Sprintf("Failed to unsilence alert: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("Successfully unsilenced alert group %s", event.AlertGroup.ID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "unsilenced", "alert_group_id": event.AlertGroup.ID})
		return
	}

	// User IS in allowlist and has event.until - create silence in Alertmanager
	if event.Event.Until == "" {
		log.Printf("User %s in allowlist but no until time specified, ignoring", event.User.Email)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ignored", "reason": "no until time"})
		return
	}

	// Parse until time
	untilTime, err := time.Parse(time.RFC3339, event.Event.Until)
	if err != nil {
		log.Printf("Failed to parse until time %s: %v", event.Event.Until, err)
		http.Error(w, fmt.Sprintf("Invalid until time: %v", err), http.StatusBadRequest)
		return
	}

	// Create silence in Alertmanager for each alert in the group
	silencesCreated := 0
	for _, alert := range event.AlertGroup.LastAlert.Payload.Alerts {
		silenceID, err := h.createSilenceForAlert(ctx, alert, event, untilTime)
		if err != nil {
			log.Printf("Failed to create silence for alert %s: %v", alert.Fingerprint, err)
			// Continue with other alerts
			continue
		}
		log.Printf("Created silence %s for alert %s", silenceID, alert.Fingerprint)
		silencesCreated++
	}

	if silencesCreated == 0 {
		http.Error(w, "Failed to create any silences", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully created %d silences in Alertmanager for alert group %s", silencesCreated, event.AlertGroup.ID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":           "silenced",
		"alert_group_id":   event.AlertGroup.ID,
		"silences_created": fmt.Sprintf("%d", silencesCreated),
	})
}

// createSilenceForAlert creates a silence in Alertmanager for a single alert
func (h *WebhookHandler) createSilenceForAlert(ctx context.Context, alert struct {
	EndsAt       string            `json:"endsAt"`
	Labels       map[string]string `json:"labels"`
	Status       string            `json:"status"`
	StartsAt     string            `json:"startsAt"`
	Annotations  map[string]string `json:"annotations"`
	Fingerprint  string            `json:"fingerprint"`
	GeneratorURL string            `json:"generatorURL"`
}, event WebhookEvent, untilTime time.Time) (string, error) {

	// Build matchers from alert labels
	matchers := make(models.Matchers, 0, len(alert.Labels))
	for key, value := range alert.Labels {
		isEqual := true
		isRegex := false
		name := key
		val := value
		matchers = append(matchers, &models.Matcher{
			IsEqual: &isEqual,
			IsRegex: &isRegex,
			Name:    &name,
			Value:   &val,
		})
	}

	// Create comment with alert group details
	comment := fmt.Sprintf("Automated silence for Grafana IRM Alert Group: %s - %s (ID: %s)",
		event.AlertGroup.Title,
		event.AlertGroup.Permalinks.Web,
		event.AlertGroup.ID,
	)

	// Create silence
	startsAt := strfmt.DateTime(time.Now())
	endsAt := strfmt.DateTime(untilTime)
	createdBy := event.User.Email

	silence := &models.PostableSilence{
		Silence: models.Silence{
			Comment:   &comment,
			CreatedBy: &createdBy,
			Matchers:  matchers,
			StartsAt:  &startsAt,
			EndsAt:    &endsAt,
		},
	}

	log.Printf("Creating silence in Alertmanager for alert %s (fingerprint: %s) until %s",
		alert.Labels["alertname"], alert.Fingerprint, untilTime.Format(time.RFC3339))

	return h.amClient.CreateSilence(ctx, silence)
}

// RegisterRoutes registers the webhook routes
func (h *WebhookHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/webhook", h.basicAuth(h.HandleWebhook))
}
