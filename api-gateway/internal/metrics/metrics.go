package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	HTTPRequests               *prometheus.CounterVec
	HTTPRequestDuration        prometheus.Summary
	OutgoingRPCRequests        *prometheus.CounterVec
	OutgoingRPCRequestDuration prometheus.Summary
}

func New() *Metrics {
	var (
		httpRequests = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests.",
			},
			[]string{"endpoint", "status"},
		)

		httpDurations = prometheus.NewSummary(
			prometheus.SummaryOpts{
				Name:       "http_request_duration_seconds",
				Help:       "Request latencies in seconds",
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			},
		)

		outgoingRPCRequests = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "outgoing_rpc_requests_total",
				Help: "Total number of requests made to polygon",
			},
			[]string{"status"},
		)

		outgoingRPCRequestDurations = prometheus.NewSummary(
			prometheus.SummaryOpts{
				Name:       "outgoing_rpc_request_duration_seconds",
				Help:       "Polygon RPC Request latencies in seconds",
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			},
		)
	)

	return &Metrics{
		HTTPRequests:               httpRequests,
		HTTPRequestDuration:        httpDurations,
		OutgoingRPCRequests:        outgoingRPCRequests,
		OutgoingRPCRequestDuration: outgoingRPCRequestDurations,
	}
}

func (m Metrics) Register() {
	prometheus.MustRegister(
		m.HTTPRequests,
		m.HTTPRequestDuration,
		m.OutgoingRPCRequests,
		m.OutgoingRPCRequestDuration,
	)
}
