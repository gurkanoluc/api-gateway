package forwarder

import (
	"bytes"
	"io"
	"net/http"
	"strconv"

	"github.com/avast/retry-go/v4"
	"github.com/gurkanoluc/trust-wallet-homework/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Client interface {
	Forward(req []byte) (*http.Response, error)
}

type DefaultClient struct {
	targetURL  string
	log        *zap.Logger
	httpClient HTTPRequestor
	metrics    *metrics.Metrics
}

type HTTPRequestor interface {
	Post(url, contentType string, body io.Reader) (*http.Response, error)
}

func New(targetURL string, log *zap.Logger, httpClient HTTPRequestor, metrics *metrics.Metrics) *DefaultClient {
	return &DefaultClient{targetURL, log, httpClient, metrics}
}

func (f *DefaultClient) Forward(req []byte) (*http.Response, error) {
	timer := prometheus.NewTimer(f.metrics.OutgoingRPCRequestDuration)
	defer timer.ObserveDuration()

	resp, err := retry.DoWithData(func() (*http.Response, error) {
		resp, err := f.httpClient.Post(f.targetURL, "application/json", bytes.NewBuffer(req))

		if err != nil {
			return nil, err
		}

		return resp, nil
	}, retry.Attempts(3))

	if err != nil {
		f.metrics.OutgoingRPCRequests.WithLabelValues("500").Inc()
		return nil, err
	}

	f.metrics.OutgoingRPCRequests.WithLabelValues(strconv.Itoa(resp.StatusCode)).Inc()
	return resp, nil
}
