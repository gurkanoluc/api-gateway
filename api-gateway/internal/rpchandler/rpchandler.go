package rpchandler

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gurkanoluc/trust-wallet-homework/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// RPCRequest represents a JSON RPC request
type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params,omitempty"`
	ID      int           `json:"id"`
}

type RPCErrorResponse struct {
	Error string `json:"error,omitempty"`
}

type Forwarder interface {
	Forward(req []byte) (*http.Response, error)
}

func NewRPCHandler(log *zap.Logger, forwarder Forwarder, m *metrics.Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		timer := prometheus.NewTimer(m.HTTPRequestDuration)
		defer timer.ObserveDuration()

		// Parse and validate the request
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			m.HTTPRequests.WithLabelValues(c.Request.URL.Path, "400").Inc()
			c.JSON(http.StatusBadRequest, RPCErrorResponse{Error: "failed to read request body"})
			return
		}

		var rpcReq RPCRequest
		err = json.Unmarshal(body, &rpcReq)

		if err != nil {
			m.HTTPRequests.WithLabelValues(c.Request.URL.Path, "400").Inc()
			c.JSON(http.StatusBadRequest, RPCErrorResponse{Error: "failed to unmarshal request body"})
			return
		}

		if err = validateRequest(rpcReq); err != nil {
			m.HTTPRequests.WithLabelValues(c.Request.URL.Path, "400").Inc()
			c.JSON(http.StatusBadRequest, RPCErrorResponse{Error: fmt.Errorf("invalid request: %w", err).Error()})
			return
		}

		// Forward incoming request to Polygon RPC
		resp, err := forwarder.Forward(body)
		if err != nil {
			m.HTTPRequests.WithLabelValues(c.Request.URL.Path, "500").Inc()
			c.JSON(http.StatusInternalServerError, RPCErrorResponse{Error: fmt.Errorf("failed to forward request: %w", err).Error()})
			return
		}

		defer resp.Body.Close()
		// Forward response back to client
		for k, vv := range resp.Header {
			for _, v := range vv {
				c.Header(k, v)
			}
		}
		c.Status(resp.StatusCode)
		bufio.NewReader(resp.Body).WriteTo(c.Writer)
		m.HTTPRequests.WithLabelValues(c.Request.URL.Path, strconv.Itoa(resp.StatusCode)).Inc()
	}
}

func validateRequest(req RPCRequest) error {
	if req.JSONRPC != "2.0" {
		return errors.New("invalid JSON RPC version")
	}
	if req.Method == "" {
		return errors.New("method is required")
	}
	if req.ID == 0 {
		return errors.New("id is required")
	}

	allowedMethods := map[string]struct{}{
		"eth_blockNumber":      {},
		"eth_getBlockByNumber": {},
	}

	if _, ok := allowedMethods[req.Method]; !ok {
		return errors.New("method not allowed")
	}
	return nil
}
