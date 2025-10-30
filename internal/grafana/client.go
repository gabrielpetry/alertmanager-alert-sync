package grafana

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	alertGroupsEndpoint  = "/api/v1/alert_groups"
	resolveAlertEndpoint = "/api/v1/alert_groups/%s/resolve"
)

// Client wraps the Grafana IRM API client
type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

// NewClient creates a new Grafana IRM client
// It reads GRAFANA_IRM_URL and GRAFANA_IRM_TOKEN from environment variables
func NewClient() (*Client, error) {
	baseURL := os.Getenv("GRAFANA_IRM_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("GRAFANA_IRM_URL environment variable not set")
	}

	apiToken := os.Getenv("GRAFANA_IRM_TOKEN")
	if apiToken == "" {
		return nil, fmt.Errorf("GRAFANA_IRM_TOKEN environment variable not set")
	}

	return &Client{
		baseURL:  baseURL,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// GetFiringAlertGroups retrieves all firing alert groups from Grafana IRM
func (c *Client) GetFiringAlertGroups() ([]AlertGroup, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, alertGroupsEndpoint)
	log.Printf("Fetching alert groups from URL: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var response AlertGroupResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	// Filter only firing alerts
	firing := make([]AlertGroup, 0)
	for _, alertGroup := range response.Results {
		if alertGroup.State == "firing" {
			firing = append(firing, alertGroup)
		}
	}

	return firing, nil
}

// GetAllAlertGroups retrieves all alert groups from Grafana IRM (firing, resolved, etc.)
func (c *Client) GetAllAlertGroups() ([]AlertGroup, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, alertGroupsEndpoint)
	log.Printf("Fetching all alert groups from URL: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var response AlertGroupResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return response.Results, nil
}

func (c *Client) ResolveAlertGroup(alertGroupID string) error {
	url := fmt.Sprintf("%s%s", c.baseURL, fmt.Sprintf(resolveAlertEndpoint, alertGroupID))
	log.Printf("Resolving alert group at URL: %s", url)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully resolved alert group: %s", alertGroupID)
	return nil
}
