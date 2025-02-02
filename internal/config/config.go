package config

import (
    "fmt"
    "strings"
    "time"

    "github.com/spf13/viper"
)

type Config struct {
    // Server settings
    ServerAddress      string        `mapstructure:"SERVER_ADDRESS"`
    GracefulTimeout   time.Duration `mapstructure:"GRACEFUL_TIMEOUT"`
    ReadTimeout       time.Duration `mapstructure:"READ_TIMEOUT"`
    WriteTimeout      time.Duration `mapstructure:"WRITE_TIMEOUT"`
    IdleTimeout       time.Duration `mapstructure:"IDLE_TIMEOUT"`
    
    // Database settings
    DatabaseURL       string        `mapstructure:"DATABASE_URL"`
    MaxDBConnections  int           `mapstructure:"MAX_DB_CONNECTIONS"`
    MaxIdleConns      int           `mapstructure:"MAX_IDLE_CONNECTIONS"`
    ConnMaxLifetime   time.Duration `mapstructure:"CONN_MAX_LIFETIME"`
    
    // Authentication
    JWTSecret        string        `mapstructure:"JWT_SECRET"`
    JWTExpiration    time.Duration `mapstructure:"JWT_EXPIRATION"`
    RefreshTokenExp  time.Duration `mapstructure:"REFRESH_TOKEN_EXPIRATION"`
    
    // WebSocket settings
    WSReadBufferSize     int           `mapstructure:"WS_READ_BUFFER_SIZE"`
    WSWriteBufferSize    int           `mapstructure:"WS_WRITE_BUFFER_SIZE"`
    WSWriteWait          time.Duration `mapstructure:"WS_WRITE_WAIT"`
    WSPongWait           time.Duration `mapstructure:"WS_PONG_WAIT"`
    WSPingPeriod         time.Duration `mapstructure:"WS_PING_PERIOD"`
    WSMaxMessageSize     int64         `mapstructure:"WS_MAX_MESSAGE_SIZE"`
    
    // Rate limiting
    RateLimitWindow      time.Duration `mapstructure:"RATE_LIMIT_WINDOW"`
    RateLimitRequests    int           `mapstructure:"RATE_LIMIT_REQUESTS"`
    
    // CORS settings
    CORSAllowedOrigins   []string      `mapstructure:"CORS_ALLOWED_ORIGINS"`
    
    // Sports API settings
    SportsAPIKey         string        `mapstructure:"SPORTS_API_KEY"`
    SportsAPIURL         string        `mapstructure:"SPORTS_API_URL"`
    
    // Feature flags
    EnableMatchUpdates   bool          `mapstructure:"ENABLE_MATCH_UPDATES"`
    EnableHighlights     bool          `mapstructure:"ENABLE_HIGHLIGHTS"`
    EnablePredictions    bool          `mapstructure:"ENABLE_PREDICTIONS"`
    
    // Environment
    Environment         string        `mapstructure:"ENVIRONMENT"`
    LogLevel           string        `mapstructure:"LOG_LEVEL"`
}

func Load() (*Config, error) {
    v := viper.New()

    // Set defaults
    setDefaults(v)

    // Read from environment variables
    v.AutomaticEnv()
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

    // Read from config file if present
    v.SetConfigName("config")
    v.AddConfigPath(".")
    v.AddConfigPath("/etc/sports-chat/")
    
    if err := v.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("failed to read config file: %w", err)
        }
    }

    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    // Validate configuration
    if err := validateConfig(&cfg); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }

    return &cfg, nil
}

func setDefaults(v *viper.Viper) {
    // Server defaults
    // Server defaults
    v.SetDefault("SERVER_ADDRESS", ":8080")
    v.SetDefault("GRACEFUL_TIMEOUT", "30s")
    v.SetDefault("READ_TIMEOUT", "15s")
    v.SetDefault("WRITE_TIMEOUT", "15s")
    v.SetDefault("IDLE_TIMEOUT", "60s")

    // Database defaults
    v.SetDefault("MAX_DB_CONNECTIONS", 20)
    v.SetDefault("MAX_IDLE_CONNECTIONS", 5)
    v.SetDefault("CONN_MAX_LIFETIME", "1h")

    // Authentication defaults
    v.SetDefault("JWT_EXPIRATION", "24h")
    v.SetDefault("REFRESH_TOKEN_EXPIRATION", "720h") // 30 days

    // WebSocket defaults
    v.SetDefault("WS_READ_BUFFER_SIZE", 1024)
    v.SetDefault("WS_WRITE_BUFFER_SIZE", 1024)
    v.SetDefault("WS_WRITE_WAIT", "10s")
    v.SetDefault("WS_PONG_WAIT", "60s")
    v.SetDefault("WS_PING_PERIOD", "54s")
    v.SetDefault("WS_MAX_MESSAGE_SIZE", 4096)

    // Rate limiting defaults
    v.SetDefault("RATE_LIMIT_WINDOW", "1m")
    v.SetDefault("RATE_LIMIT_REQUESTS", 60)

    // CORS defaults
    v.SetDefault("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"})

    // Sports API defaults
    v.SetDefault("SPORTS_API_URL", "https://api.sports-data.io/v1")

    // Feature flags
    v.SetDefault("ENABLE_MATCH_UPDATES", true)
    v.SetDefault("ENABLE_HIGHLIGHTS", true)
    v.SetDefault("ENABLE_PREDICTIONS", true)

    // Environment defaults
    v.SetDefault("ENVIRONMENT", "development")
    v.SetDefault("LOG_LEVEL", "info")
}

func validateConfig(cfg *Config) error {
    // Required fields
    if cfg.JWTSecret == "" {
        return fmt.Errorf("JWT_SECRET is required")
    }
    if cfg.DatabaseURL == "" {
        return fmt.Errorf("DATABASE_URL is required")
    }

    // Validate timeouts
    if cfg.WSPingPeriod >= cfg.WSPongWait {
        return fmt.Errorf("ping period must be less than pong wait")
    }

    // Validate rate limiting
    if cfg.RateLimitRequests <= 0 {
        return fmt.Errorf("rate limit requests must be positive")
    }

    // Validate sports API settings
    if cfg.EnableMatchUpdates && cfg.SportsAPIKey == "" {
        return fmt.Errorf("SPORTS_API_KEY is required when match updates are enabled")
    }

    return nil
}