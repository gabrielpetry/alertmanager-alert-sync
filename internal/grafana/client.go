package grafana

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	alertGroupsEndpoint   = "/api/v1/alert_groups"
	resolveAlertEndpoint  = "/api/v1/alert_groups/%s/resolve"
	unsilenceAlertEndpoint = "/api/v1/alert_groups/%s/unsilence"
	userEndpoint          = "/api/v1/users/%s"
)

// Client wraps the Grafana IRM API client
type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
	userCache  map[string]*User
	cacheMutex sync.RWMutex
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
		userCache: make(map[string]*User),
	}, nil
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

// UnsilenceAlertGroup unsilences an alert group in Grafana IRM
func (c *Client) UnsilenceAlertGroup(alertGroupID string) error {
	url := fmt.Sprintf("%s%s", c.baseURL, fmt.Sprintf(unsilenceAlertEndpoint, alertGroupID))
	log.Printf("Unsilencing alert group at URL: %s", url)

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

	log.Printf("Successfully unsilenced alert group: %s", alertGroupID)
	return nil
}

// GetUser retrieves user information by user ID with caching
func (c *Client) GetUser(userID string) (*User, error) {
	if userID == "" {
		return nil, nil
	}

	// Check cache first (read lock)
	c.cacheMutex.RLock()
	if user, exists := c.userCache[userID]; exists {
		c.cacheMutex.RUnlock()
		return user, nil
	}
	c.cacheMutex.RUnlock()

	// User not in cache, fetch from API
	url := fmt.Sprintf("%s%s", c.baseURL, fmt.Sprintf(userEndpoint, userID))
	log.Printf("Fetching user from URL: %s", url)

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

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	// Store in cache (write lock)
	c.cacheMutex.Lock()
	c.userCache[userID] = &user
	c.cacheMutex.Unlock()

	log.Printf("Cached user %s (email: %s)", userID, user.Email)
	return &user, nil
}

// GetUserEmail retrieves only the email for a user ID (with caching)
func (c *Client) GetUserEmail(userID string) string {
	user, err := c.GetUser(userID)
	if err != nil {
		log.Printf("Failed to fetch user %s: %v", userID, err)
		return ""
	}
	if user == nil {
		return ""
	}
	return user.Email
}
