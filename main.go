package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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

	defaultLabels = []string{"alertname", "alertstate", "alertstart", "alertjob"}
	customLabels  = strings.Split(os.Getenv("CUSTOM_LABELS"), ",")

	alertsSyncAlerts = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "alertmanager_sync_alerts",
		Help: "alerts with state from alertmanager api",
	},
		append(defaultLabels, customLabels...),
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

		metricLabels := prometheus.Labels{
			"alertname":  alert.Labels["alertname"],
			"alertstate": *alert.Status.State,
			"alertstart": strconv.FormatInt(parsedTime.Unix(), 10),
			"alertjob":   alert.Labels["job"],
		}

		for _, label := range customLabels {
			if val, ok := alert.Labels[label]; ok {
				metricLabels[label] = val
			} else {
				metricLabels[label] = ""
			}
		}

		alertsSyncAlerts.With(metricLabels).Set(1)
	}
	promhttp.Handler().ServeHTTP(w, r)
}

func main() {
	// Register the /metrics endpoint
	http.HandleFunc("/metrics", syncHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// Start the server
	log.Printf("Starting server on port :%s ...", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
