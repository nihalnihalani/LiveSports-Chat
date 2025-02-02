package main

import (
    "context"
    "expvar"
    "fmt"
    "log"
    "net/http"
    "net/http/pprof"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/rs/cors"
    "go.uber.org/zap"

    "github.com/yourusername/sports-chat/internal/api"
    "github.com/yourusername/sports-chat/internal/auth"
    "github.com/yourusername/sports-chat/internal/config"
    "github.com/yourusername/sports-chat/internal/metrics"
    "github.com/yourusername/sports-chat/internal/store/postgres"
    "github.com/yourusername/sports-chat/internal/websocket"
)

var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

func main() {
    // Initialize logger
    logger, err := zap.NewProduction()
    if err != nil {
        log.Fatalf("Failed to initialize logger: %v", err)
    }
    defer logger.Sync()

    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        logger.Fatal("Failed to load config", zap.Error(err))
    }

    // Initialize metrics
    metricsRegistry := prometheus.NewRegistry()
    metrics := metrics.NewMetrics(metricsRegistry)

    // Initialize stores
    db, err := postgres.New(cfg.DatabaseURL, logger)
    if err != nil {
        logger.Fatal("Failed to initialize postgres", zap.Error(err))
    }
    defer db.Close()

    // Initialize auth service
    authService := auth.NewService(cfg.JWTSecret, logger)

    // Initialize websocket hub
    hub := websocket.NewHub(db, metrics, logger)
    go hub.Run()

    // Initialize API handlers
    apiHandler := api.NewHandler(db, authService, metrics, logger)

    // Setup middleware chain
    mw := cors.New(cors.Options{
        AllowedOrigins:   cfg.CORSAllowedOrigins,
        AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"Authorization", "Content-Type"},
        AllowCredentials: true,
    })

    // Setup routes
    mux := http.NewServeMux()

    // API routes
    mux.Handle("/api/", http.StripPrefix("/api", apiHandler))
    mux.Handle("/ws", websocket.NewHandler(hub, authService, metrics, logger))

    // Metrics and debugging
    if cfg.Environment == "development" {
        mux.Handle("/debug/vars", expvar.Handler())
        mux.HandleFunc("/debug/pprof/", pprof.Index)
        mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
        mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
        mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
        mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
    }

    mux.Handle("/metrics", promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{
        Registry: metricsRegistry,
    }))

    // Health check
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "OK")
    })

    // Version info
    mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Version: %s\nCommit: %s\nBuild Date: %s\n", version, commit, date)
    })

    // Create server
    srv := &http.Server{
        Addr:         cfg.ServerAddress,
        Handler:      mw.Handler(mux),
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Start server
    go func() {
        logger.Info("Starting server",
            zap.String("address", cfg.ServerAddress),
            zap.String("version", version),
            zap.String("commit", commit),
            zap.String("build_date", date))

        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("Server failed", zap.Error(err))
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    // Graceful shutdown
    logger.Info("Server is shutting down...")
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        logger.Fatal("Server forced to shutdown", zap.Error(err))
    }

    logger.Info("Server stopped gracefully")
}