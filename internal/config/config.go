package config

import (
	"errors"
	"strings"

	"github.com/spf13/viper"
)

// ServerConfig holds server configuration
type ServerConfig struct {
	Port    string `mapstructure:"port"`
	Env     string `mapstructure:"env"`
	BaseURL string `mapstructure:"base_url"`
}

// MongoDBConfig holds MongoDB configuration
type MongoDBConfig struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URI string `mapstructure:"uri"`
}

// S3Config holds S3/MinIO configuration
type S3Config struct {
	BucketName      string `mapstructure:"bucket_name"`
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Endpoint        string `mapstructure:"endpoint"`
}

// CleanupConfig holds cleanup worker configuration
type CleanupConfig struct {
	Interval  string `mapstructure:"interval"`   // e.g., "5m", "1h"
	BatchSize int64  `mapstructure:"batch_size"` // number of pastes to process per batch
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int  `mapstructure:"requests_per_minute"` // max requests per minute per IP
	Enabled           bool `mapstructure:"enabled"`             // whether rate limiting is enabled
}

// Config holds all configuration for the application
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	MongoDB   MongoDBConfig   `mapstructure:"mongodb"`
	Redis     RedisConfig     `mapstructure:"redis"`
	S3        S3Config        `mapstructure:"s3"`
	Cleanup   CleanupConfig   `mapstructure:"cleanup"`
	RateLimit RateLimitConfig `mapstructure:"ratelimit"`
}

// Load reads configuration from environment variables and config files
func Load() (*Config, error) {
	v := viper.New()

	// Set default values
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.env", "development")
	v.SetDefault("mongodb.database", "gisty")
	v.SetDefault("cleanup.interval", "5m")
	v.SetDefault("cleanup.batch_size", 100)
	v.SetDefault("ratelimit.requests_per_minute", 5)
	v.SetDefault("ratelimit.enabled", true)

	// Config file settings
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/gisty")

	// Read config file (optional - won't error if not found)
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, err
		}
	}

	// Environment variable settings
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Bind environment variables explicitly
	bindEnvVars(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// bindEnvVars binds environment variables to config keys
func bindEnvVars(v *viper.Viper) {
	// Server
	_ = v.BindEnv("server.port", "PORT")
	_ = v.BindEnv("server.env", "ENV")
	_ = v.BindEnv("server.base_url", "BASE_URL")

	// MongoDB
	_ = v.BindEnv("mongodb.uri", "MONGO_URI")
	_ = v.BindEnv("mongodb.database", "MONGO_DB")

	// Redis
	_ = v.BindEnv("redis.uri", "REDIS_URI")

	// S3
	_ = v.BindEnv("s3.bucket_name", "S3_BUCKET_NAME")
	_ = v.BindEnv("s3.region", "S3_REGION")
	_ = v.BindEnv("s3.access_key_id", "S3_ACCESS_KEY_ID")
	_ = v.BindEnv("s3.secret_access_key", "S3_SECRET_ACCESS_KEY")
	_ = v.BindEnv("s3.endpoint", "S3_ENDPOINT")

	// Cleanup
	_ = v.BindEnv("cleanup.interval", "CLEANUP_INTERVAL")
	_ = v.BindEnv("cleanup.batch_size", "CLEANUP_BATCH_SIZE")

	// Rate Limit
	_ = v.BindEnv("ratelimit.requests_per_minute", "RATE_LIMIT_REQUESTS_PER_MINUTE")
	_ = v.BindEnv("ratelimit.enabled", "RATE_LIMIT_ENABLED")
}

// Validate checks if required configuration fields are set
func (c *Config) Validate() error {
	var missingFields []string

	if c.MongoDB.URI == "" {
		missingFields = append(missingFields, "mongodb.uri (MONGO_URI)")
	}

	if c.Redis.URI == "" {
		missingFields = append(missingFields, "redis.uri (REDIS_URI)")
	}

	if c.S3.BucketName == "" {
		missingFields = append(missingFields, "s3.bucket_name (S3_BUCKET_NAME)")
	}

	if c.S3.Region == "" {
		missingFields = append(missingFields, "s3.region (S3_REGION)")
	}

	if c.S3.AccessKeyID == "" {
		missingFields = append(missingFields, "s3.access_key_id (S3_ACCESS_KEY_ID)")
	}

	if c.S3.SecretAccessKey == "" {
		missingFields = append(missingFields, "s3.secret_access_key (S3_SECRET_ACCESS_KEY)")
	}

	if len(missingFields) > 0 {
		return errors.New("missing required configuration: " + strings.Join(missingFields, ", "))
	}

	return nil
}
