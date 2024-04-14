package forwarder

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/gurkanoluc/trust-wallet-homework/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/zap"
)

type mockHTTPRequestor struct {
	response *http.Response
	err      error
}

func (m *mockHTTPRequestor) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	return m.response, m.err
}

func TestDefaultClient_Forward(t *testing.T) {
	targetURL := "http://example.com"
	log := zap.NewNop()
	metrics := metrics.New()
	req := []byte("test request")

	t.Run("successful request", func(t *testing.T) {
		expectedResp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("test response")),
		}

		client := New(targetURL, log, &mockHTTPRequestor{response: expectedResp}, metrics)
		resp, err := client.Forward(req)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if resp != expectedResp {
			t.Errorf("expected response: %v, got: %v", expectedResp, resp)
		}

		if testutil.ToFloat64(metrics.OutgoingRPCRequests.WithLabelValues("200")) != float64(1) {
			t.Error("metric OutgoingRPCRequests should have been incremented")
		}
	})

	t.Run("request error", func(t *testing.T) {
		expectedErr := errors.New("request error")

		client := New(targetURL, log, &mockHTTPRequestor{err: expectedErr}, metrics)
		resp, err := client.Forward(req)

		if err != expectedErr {
			t.Errorf("expected error: %v, got: %v", expectedErr, err)
		}

		if resp != nil {
			t.Errorf("expected nil response, got: %v", resp)
		}
		if testutil.ToFloat64(metrics.OutgoingRPCRequests.WithLabelValues("500")) != float64(1) {
			t.Error("metric OutgoingRPCRequests should have been incremented")
		}
	})
}
