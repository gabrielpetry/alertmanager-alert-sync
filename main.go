package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	// We'll need this to convert numbers to strings
	// Prometheus metrics client
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	// New Alertmanager client imports
	"github.com/go-openapi/strfmt"
	amclient "github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
)

var (
	// This is the client we'll use to talk to Alertmanager
	amAPI *amclient.AlertmanagerAPI

	alertsSyncAlerts = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "alertmanager_sync_alerts",
		Help: "alerts with state from alertmanager api",
	},
		[]string{"alertname", "alertstate", "alertstart", "cluster", "job", "severity"},
	)

	alertSyncTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "alertmanager_sync_total",
		Help: "The total number of sync attempts to Alertmanager",
	})

	alertsSyncFailuresTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "alertmanager_sync_failures_total",
		Help: "The total number of failed sync attempts to Alertmanager",
	})
)

// init() runs once before main() to set up global variables.
// This is a great place to create our API client.
func init() {
	alertmanagerHost := os.Getenv("ALERTMANAGER_HOST")
	if alertmanagerHost == "" {
		alertmanagerHost = "localhost:9093" // Default Alertmanager host
	}

	cfg := amclient.DefaultTransportConfig().WithHost(alertmanagerHost)
	amAPI = amclient.NewHTTPClientWithConfig(strfmt.Default, cfg)
	log.Printf("Alertmanager client initialized for host: %s", alertmanagerHost)
}

// This is our main application logic.
func syncHandler(w http.ResponseWriter, r *http.Request) {
	active := true

	params := alert.NewGetAlertsParams().
		WithActive(&active).
		WithContext(r.Context())

	alertSyncTotal.Inc()
	// reset previous metrics
	alertsSyncAlerts.Reset()

	ok, err := amAPI.Alert.GetAlerts(params)
	if err != nil {
		log.Printf("Error querying Alertmanager: %v", err)
		alertsSyncFailuresTotal.Inc() // Increment failure counter
		http.Error(w, "Error querying Alertmanager", http.StatusInternalServerError)
		return
	}

	for _, alert := range ok.Payload {
		layout := time.RFC3339

		parsedTime, err := time.Parse(layout, alert.StartsAt.String())
		if err != nil {
			log.Printf("Error parsing time: %v", err)
			panic(err)
		}

		alertsSyncAlerts.With(prometheus.Labels{
			"alertname":  alert.Labels["alertname"],
			"alertstate": *alert.Status.State,
			"alertstart": strconv.FormatInt(parsedTime.Unix(), 10),
			"cluster":    alert.Labels["cluster"],
			"job":        alert.Labels["job"],
			"severity":   alert.Labels["severity"],
		}).Set(1)
	}
	promhttp.Handler().ServeHTTP(w, r)
}

func main() {
	// Register the /metrics endpoint
	http.HandleFunc("/metrics", syncHandler)

	// Start the server
	log.Println("Starting server on port :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
