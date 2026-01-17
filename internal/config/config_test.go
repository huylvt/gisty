package config

import (
	"os"
	"testing"
)

func TestLoad_WithEnvVariables(t *testing.T) {
	// Set required environment variables
	envVars := map[string]string{
		"PORT":                 "9090",
		"ENV":                  "production",
		"MONGO_URI":            "mongodb://testhost:27017",
		"MONGO_DB":             "testdb",
		"REDIS_URI":            "redis://testhost:6379",
		"S3_BUCKET_NAME":       "test-bucket",
		"S3_REGION":            "us-west-2",
		"S3_ACCESS_KEY_ID":     "test-key",
		"S3_SECRET_ACCESS_KEY": "test-secret",
		"S3_ENDPOINT":          "http://localhost:9000",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	// Verify server config
	if cfg.Server.Port != "9090" {
		t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "9090")
	}
	if cfg.Server.Env != "production" {
		t.Errorf("Server.Env = %q, want %q", cfg.Server.Env, "production")
	}

	// Verify MongoDB config
	if cfg.MongoDB.URI != "mongodb://testhost:27017" {
		t.Errorf("MongoDB.URI = %q, want %q", cfg.MongoDB.URI, "mongodb://testhost:27017")
	}
	if cfg.MongoDB.Database != "testdb" {
		t.Errorf("MongoDB.Database = %q, want %q", cfg.MongoDB.Database, "testdb")
	}

	// Verify Redis config
	if cfg.Redis.URI != "redis://testhost:6379" {
		t.Errorf("Redis.URI = %q, want %q", cfg.Redis.URI, "redis://testhost:6379")
	}

	// Verify S3 config
	if cfg.S3.BucketName != "test-bucket" {
		t.Errorf("S3.BucketName = %q, want %q", cfg.S3.BucketName, "test-bucket")
	}
	if cfg.S3.Region != "us-west-2" {
		t.Errorf("S3.Region = %q, want %q", cfg.S3.Region, "us-west-2")
	}
	if cfg.S3.AccessKeyID != "test-key" {
		t.Errorf("S3.AccessKeyID = %q, want %q", cfg.S3.AccessKeyID, "test-key")
	}
	if cfg.S3.SecretAccessKey != "test-secret" {
		t.Errorf("S3.SecretAccessKey = %q, want %q", cfg.S3.SecretAccessKey, "test-secret")
	}
	if cfg.S3.Endpoint != "http://localhost:9000" {
		t.Errorf("S3.Endpoint = %q, want %q", cfg.S3.Endpoint, "http://localhost:9000")
	}
}

func TestLoad_MissingRequiredFields(t *testing.T) {
	// Clear all relevant env vars
	envVars := []string{
		"MONGO_URI", "MONGO_DB", "REDIS_URI",
		"S3_BUCKET_NAME", "S3_REGION", "S3_ACCESS_KEY_ID", "S3_SECRET_ACCESS_KEY",
	}
	for _, k := range envVars {
		os.Unsetenv(k)
	}

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error when required fields are missing")
	}
}

func TestLoad_PartialMissingFields(t *testing.T) {
	// Set only some required fields
	os.Setenv("MONGO_URI", "mongodb://localhost:27017")
	os.Setenv("REDIS_URI", "redis://localhost:6379")
	// S3 fields are missing

	defer func() {
		os.Unsetenv("MONGO_URI")
		os.Unsetenv("REDIS_URI")
	}()

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error when S3 fields are missing")
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	// Set only required fields
	envVars := map[string]string{
		"MONGO_URI":            "mongodb://localhost:27017",
		"REDIS_URI":            "redis://localhost:6379",
		"S3_BUCKET_NAME":       "test-bucket",
		"S3_REGION":            "us-west-2",
		"S3_ACCESS_KEY_ID":     "test-key",
		"S3_SECRET_ACCESS_KEY": "test-secret",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	// Check defaults
	if cfg.Server.Port != "8080" {
		t.Errorf("Server.Port default = %q, want %q", cfg.Server.Port, "8080")
	}
	if cfg.Server.Env != "development" {
		t.Errorf("Server.Env default = %q, want %q", cfg.Server.Env, "development")
	}
	if cfg.MongoDB.Database != "gisty" {
		t.Errorf("MongoDB.Database default = %q, want %q", cfg.MongoDB.Database, "gisty")
	}
}

func TestValidate_AllFieldsSet(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: "8080",
			Env:  "development",
		},
		MongoDB: MongoDBConfig{
			URI:      "mongodb://localhost:27017",
			Database: "gisty",
		},
		Redis: RedisConfig{
			URI: "redis://localhost:6379",
		},
		S3: S3Config{
			BucketName:      "test-bucket",
			Region:          "us-west-2",
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() returned error for valid config: %v", err)
	}
}

func TestValidate_MissingMongoURI(t *testing.T) {
	cfg := &Config{
		MongoDB: MongoDBConfig{URI: ""},
		Redis:   RedisConfig{URI: "redis://localhost:6379"},
		S3: S3Config{
			BucketName:      "test",
			Region:          "us-west-2",
			AccessKeyID:     "key",
			SecretAccessKey: "secret",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error when MongoDB.URI is empty")
	}
}

func TestValidate_MissingRedisURI(t *testing.T) {
	cfg := &Config{
		MongoDB: MongoDBConfig{URI: "mongodb://localhost:27017"},
		Redis:   RedisConfig{URI: ""},
		S3: S3Config{
			BucketName:      "test",
			Region:          "us-west-2",
			AccessKeyID:     "key",
			SecretAccessKey: "secret",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error when Redis.URI is empty")
	}
}

func TestValidate_MissingS3Fields(t *testing.T) {
	testCases := []struct {
		name string
		s3   S3Config
	}{
		{
			name: "missing bucket_name",
			s3:   S3Config{BucketName: "", Region: "us-west-2", AccessKeyID: "key", SecretAccessKey: "secret"},
		},
		{
			name: "missing region",
			s3:   S3Config{BucketName: "test", Region: "", AccessKeyID: "key", SecretAccessKey: "secret"},
		},
		{
			name: "missing access_key_id",
			s3:   S3Config{BucketName: "test", Region: "us-west-2", AccessKeyID: "", SecretAccessKey: "secret"},
		},
		{
			name: "missing secret_access_key",
			s3:   S3Config{BucketName: "test", Region: "us-west-2", AccessKeyID: "key", SecretAccessKey: ""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				MongoDB: MongoDBConfig{URI: "mongodb://localhost:27017"},
				Redis:   RedisConfig{URI: "redis://localhost:6379"},
				S3:      tc.s3,
			}

			err := cfg.Validate()
			if err == nil {
				t.Errorf("Validate() should return error for %s", tc.name)
			}
		})
	}
}