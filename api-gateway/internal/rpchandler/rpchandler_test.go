package rpchandler

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gurkanoluc/trust-wallet-homework/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/zap"
)

type mockForwarder struct {
	response *http.Response
	err      error
}

func (m *mockForwarder) Forward(req []byte) (*http.Response, error) {
	return m.response, m.err
}

func TestNewRPCHandler(t *testing.T) {
	log := zap.NewNop()
	metrics := metrics.New()

	t.Run("successful request", func(t *testing.T) {
		expectedResp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("test response")),
			Header:     make(http.Header),
		}
		expectedResp.Header.Set("foo", "bar")

		forwarder := &mockForwarder{response: expectedResp}
		handler := NewRPCHandler(log, forwarder, metrics)

		// Create a test request
		reqBody := []byte(`{"jsonrpc": "2.0", "method": "eth_blockNumber", "params": [], "id": 1}`)
		req, _ := http.NewRequest("POST", "/rpc", bytes.NewBuffer(reqBody))
		w := httptest.NewRecorder()

		// Call the handler
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		handler(c)

		// Check the response
		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		if resp.Header.Get("foo") != "bar" {
			t.Errorf("Expected header 'foo' to be 'bar', got '%s'", resp.Header.Get("foo"))
		}

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "test response" {
			t.Errorf("expected response body 'test response', got '%s'", string(body))
		}
		if testutil.ToFloat64(metrics.HTTPRequests.WithLabelValues("/rpc", "200")) != float64(1) {
			t.Error("metric HTTPRequests should have been incremented")
		}

	})

	t.Run("failed to forward request", func(t *testing.T) {
		expectedErr := errors.New("forwarding error")

		forwarder := &mockForwarder{err: expectedErr}
		handler := NewRPCHandler(log, forwarder, metrics)

		// Create a test request
		reqBody := []byte(`{"jsonrpc": "2.0", "method": "eth_blockNumber", "params": [], "id": 1}`)
		req, _ := http.NewRequest("POST", "/rpc", bytes.NewBuffer(reqBody))
		w := httptest.NewRecorder()

		// Call the handler
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		handler(c)

		// Check the response
		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		expectedBody := `{"error":"failed to forward request: forwarding error"}`
		if string(body) != expectedBody {
			t.Errorf("expected response body '%s', got '%s'", expectedBody, string(body))
		}
		if testutil.ToFloat64(metrics.HTTPRequests.WithLabelValues("/rpc", "500")) != float64(1) {
			t.Error("metric HTTPRequests should have been incremented")
		}
	})
}
