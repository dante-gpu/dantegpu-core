package config

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/pricing"
	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/service"
	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/solana"
)

// Config represents the complete configuration for the billing service
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Solana     solana.Config    `yaml:"solana"`
	Pricing    pricing.Config   `yaml:"pricing"`
	NATS       NATSConfig       `yaml:"nats"`
	Consul     ConsulConfig     `yaml:"consul"`
	Wallet     WalletConfig     `yaml:"wallet"`
	Security   SecurityConfig   `yaml:"security"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	Billing    service.Config   `yaml:"billing"`
	Payouts    PayoutsConfig    `yaml:"payouts"`
	LogLevel   string           `yaml:"log_level"`
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	URL               string        `yaml:"url"`
	MaxConnections    int           `yaml:"max_connections"`
	MinConnections    int           `yaml:"min_connections"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
	MaxLifetime       time.Duration `yaml:"max_lifetime"`
}

// NATSConfig represents NATS configuration
type NATSConfig struct {
	Address   string          `yaml:"address"`
	ClusterID string          `yaml:"cluster_id"`
	ClientID  string          `yaml:"client_id"`
	Subjects  SubjectsConfig  `yaml:"subjects"`
	JetStream JetStreamConfig `yaml:"jetstream"`
}

// SubjectsConfig represents NATS subjects configuration
type SubjectsConfig struct {
	UsageUpdates    string `yaml:"usage_updates"`
	PaymentEvents   string `yaml:"payment_events"`
	SessionEvents   string `yaml:"session_events"`
	ProviderPayouts string `yaml:"provider_payouts"`
}

// JetStreamConfig represents NATS JetStream configuration
type JetStreamConfig struct {
	Enabled    bool          `yaml:"enabled"`
	StreamName string        `yaml:"stream_name"`
	MaxAge     time.Duration `yaml:"max_age"`
	MaxMsgs    int64         `yaml:"max_msgs"`
}

// ConsulConfig represents Consul configuration
type ConsulConfig struct {
	Address                            string        `yaml:"address"`
	ServiceName                        string        `yaml:"service_name"`
	ServiceID                          string        `yaml:"service_id"`
	HealthCheckInterval                time.Duration `yaml:"health_check_interval"`
	HealthCheckTimeout                 time.Duration `yaml:"health_check_timeout"`
	HealthCheckDeregisterCriticalAfter time.Duration `yaml:"health_check_deregister_critical_after"`
}

// WalletConfig represents wallet configuration
type WalletConfig struct {
	MinimumBalance      decimal.Decimal `yaml:"minimum_balance"`
	LowBalanceThreshold decimal.Decimal `yaml:"low_balance_threshold"`
	AutoRefillEnabled   bool            `yaml:"auto_refill_enabled"`
	AutoRefillThreshold decimal.Decimal `yaml:"auto_refill_threshold"`
	AutoRefillAmount    decimal.Decimal `yaml:"auto_refill_amount"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	RateLimit            RateLimitConfig `yaml:"rate_limit"`
	MaxTransactionAmount decimal.Decimal `yaml:"max_transaction_amount"`
	DailyWithdrawalLimit decimal.Decimal `yaml:"daily_withdrawal_limit"`
	EncryptionKeyPath    string          `yaml:"encryption_key_path"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int `yaml:"requests_per_minute"`
	BurstSize         int `yaml:"burst_size"`
}

// MonitoringConfig represents monitoring configuration
type MonitoringConfig struct {
	MetricsEnabled     bool             `yaml:"metrics_enabled"`
	MetricsPort        int              `yaml:"metrics_port"`
	HealthCheckEnabled bool             `yaml:"health_check_enabled"`
	Prometheus         PrometheusConfig `yaml:"prometheus"`
}

// PrometheusConfig represents Prometheus configuration
type PrometheusConfig struct {
	Namespace string `yaml:"namespace"`
	Subsystem string `yaml:"subsystem"`
}

// PayoutsConfig represents payout configuration
type PayoutsConfig struct {
	MinimumPayoutAmount decimal.Decimal `yaml:"minimum_payout_amount"`
	PayoutSchedule      string          `yaml:"payout_schedule"`
	PayoutFeePercent    decimal.Decimal `yaml:"payout_fee_percent"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server configuration
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Validate database configuration
	if c.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}

	// Validate Solana configuration
	if c.Solana.RPCURL == "" {
		return fmt.Errorf("Solana RPC URL is required")
	}
	if c.Solana.TokenAddress == "" {
		return fmt.Errorf("dGPU token address is required")
	}
	if c.Solana.PlatformWallet == "" {
		return fmt.Errorf("platform wallet address is required")
	}

	// Validate pricing configuration
	if len(c.Pricing.BaseRates) == 0 {
		return fmt.Errorf("at least one base rate must be configured")
	}

	// Validate wallet configuration
	if c.Wallet.MinimumBalance.LessThan(decimal.Zero) {
		return fmt.Errorf("minimum balance cannot be negative")
	}

	// Validate security configuration
	if c.Security.MaxTransactionAmount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("max transaction amount must be positive")
	}

	// Validate payout configuration
	if c.Payouts.MinimumPayoutAmount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("minimum payout amount must be positive")
	}
	if c.Payouts.PayoutFeePercent.LessThan(decimal.Zero) || c.Payouts.PayoutFeePercent.GreaterThan(decimal.NewFromInt(100)) {
		return fmt.Errorf("payout fee percent must be between 0 and 100")
	}

	return nil
}

// GetDatabaseConfig returns database configuration for pgxpool
func (c *Config) GetDatabaseConfig() (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(c.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	config.MaxConns = int32(c.Database.MaxConnections)
	config.MinConns = int32(c.Database.MinConnections)
	config.MaxConnLifetime = c.Database.MaxLifetime
	config.MaxConnIdleTime = c.Database.IdleTimeout

	return config, nil
}

// GetSolanaConfig returns Solana client configuration
func (c *Config) GetSolanaConfig() *solana.Config {
	return &c.Solana
}

// GetPricingConfig returns pricing engine configuration
func (c *Config) GetPricingConfig() *pricing.Config {
	return &c.Pricing
}

// GetBillingConfig returns billing service configuration
func (c *Config) GetBillingConfig() *service.Config {
	return &c.Billing
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.LogLevel == "debug"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.LogLevel == "error" || c.LogLevel == "warn"
}
