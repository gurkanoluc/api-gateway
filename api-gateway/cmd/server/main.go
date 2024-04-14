package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gurkanoluc/trust-wallet-homework/internal/forwarder"
	"github.com/gurkanoluc/trust-wallet-homework/internal/metrics"
	"github.com/gurkanoluc/trust-wallet-homework/internal/rpchandler"
	"github.com/ulule/limiter/v3"
	rlgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	port     = flag.Int("port", 8080, "HTTP server port")
	logLevel = flag.String("logLevel", "info", "Log level")
)

func main() {
	m := metrics.New()
	m.Register()

	var zapConfig zap.Config
	if *logLevel == "debug" {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}
	zapConfig.Level.SetLevel(parseLogLevel(*logLevel))
	log, _ := zapConfig.Build()

	router := newRouter(log, m)

	runServer(log, router, *port)
}

func newRouter(log *zap.Logger, m *metrics.Metrics) http.Handler {
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  1000,
	}
	store := memory.NewStore()
	middleware := rlgin.NewMiddleware(limiter.New(store, rate))

	router := gin.Default()
	router.Use(middleware)
	router.POST("/rpc",
		rpchandler.NewRPCHandler(
			log,
			forwarder.New("https://polygon-rpc.com", log, newHTTPClient(), m),
			m,
		),
	)
	router.GET("/metrics", metrics.NewHandler())
	router.GET("/health", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "OK")
	})

	return router
}

func runServer(log *zap.Logger, router http.Handler, port int) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("can't listen: %w", zap.Error(err))
			os.Exit(1)
		}
	}()

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	log.Info("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown: ", zap.Error(err))
	}
}

func newHTTPClient() *http.Client {
	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	return &http.Client{
		Timeout:   time.Second * 10,
		Transport: netTransport,
	}

}

func parseLogLevel(logLevel string) zapcore.Level {
	switch logLevel {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
