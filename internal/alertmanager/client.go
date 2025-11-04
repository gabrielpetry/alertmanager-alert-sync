package alertmanager

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/go-openapi/strfmt"
	amclient "github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/client/silence"
	"github.com/prometheus/alertmanager/api/v2/models"
)

// Client wraps the Alertmanager API client
type Client struct {
	api          *amclient.AlertmanagerAPI
	silenceCache map[string]*models.GettableSilence
	cacheMutex   sync.RWMutex
}

// NewClient creates a new Alertmanager client
// It reads the ALERTMANAGER_HOST environment variable or defaults to localhost:9093
func NewClient() *Client {
	alertmanagerHost := os.Getenv("ALERTMANAGER_HOST")
	if alertmanagerHost == "" {
		alertmanagerHost = "localhost:9093"
	}

	cfg := amclient.DefaultTransportConfig().WithHost(alertmanagerHost)
	api := amclient.NewHTTPClientWithConfig(strfmt.Default, cfg)
	log.Printf("Alertmanager client initialized for host: %s", alertmanagerHost)

	return &Client{
		api:          api,
		silenceCache: make(map[string]*models.GettableSilence),
	}
}

// GetFiringAlerts fetches only firing alerts from Alertmanager
func (c *Client) GetFiringAlerts(ctx context.Context) ([]*models.GettableAlert, error) {
	active := true
	params := alert.NewGetAlertsParams().
		WithActive(&active).
		WithContext(ctx)

	ok, err := c.api.Alert.GetAlerts(params)
	if err != nil {
		return nil, err
	}

	return ok.Payload, nil
}

// GetAllAlerts fetches all alerts from Alertmanager, including resolved and silenced
func (c *Client) GetAllAlerts(ctx context.Context) ([]*models.GettableAlert, error) {
	params := alert.NewGetAlertsParams().
		WithContext(ctx)

	ok, err := c.api.Alert.GetAlerts(params)
	if err != nil {
		return nil, err
	}

	return ok.Payload, nil
}

// GetSilencedFiringAlerts retrieves alerts that are currently firing but have been silenced
// These alerts exist in Alertmanager but are not actively notifying due to silences
func (c *Client) GetSilencedFiringAlerts(ctx context.Context) ([]*models.GettableAlert, error) {
	allAlerts, err := c.GetAllAlerts(ctx)
	if err != nil {
		return nil, err
	}

	var silencedFiring []*models.GettableAlert
	for _, alert := range allAlerts {
		// Check if alert is firing and has silences
		if alert.Status != nil &&
			*alert.Status.State == "suppressed" &&
			len(alert.Status.SilencedBy) > 0 {
			// Append to silenced firing alerts
			silencedFiring = append(silencedFiring, alert)
		}
	}

	return silencedFiring, nil
}

// GetSilence retrieves silence details by silence ID with caching
func (c *Client) GetSilence(ctx context.Context, silenceID string) (*models.GettableSilence, error) {
	if silenceID == "" {
		return nil, nil
	}

	// Check cache first (read lock)
	c.cacheMutex.RLock()
	if silence, exists := c.silenceCache[silenceID]; exists {
		c.cacheMutex.RUnlock()
		return silence, nil
	}
	c.cacheMutex.RUnlock()

	// Silence not in cache, fetch from API
	params := silence.NewGetSilenceParams().
		WithSilenceID(strfmt.UUID(silenceID)).
		WithContext(ctx)

	ok, err := c.api.Silence.GetSilence(params)
	if err != nil {
		log.Printf("Failed to fetch silence %s: %v", silenceID, err)
		return nil, err
	}

	// Store in cache (write lock)
	c.cacheMutex.Lock()
	c.silenceCache[silenceID] = ok.Payload
	c.cacheMutex.Unlock()

	log.Printf("Cached silence %s (author: %s)", silenceID, *ok.Payload.CreatedBy)
	return ok.Payload, nil
}

// GetSilenceAuthor retrieves the author of a silence by silence ID (with caching)
func (c *Client) GetSilenceAuthor(ctx context.Context, silenceID string) string {
	silence, err := c.GetSilence(ctx, silenceID)
	if err != nil || silence == nil {
		return ""
	}
	if silence.CreatedBy != nil {
		return *silence.CreatedBy
	}
	return ""
}
