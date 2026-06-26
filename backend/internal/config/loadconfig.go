package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server              ServerConfig        `yaml:"server"`
	Database            DatabaseConfig      `yaml:"database"`
	Redis               RedisConfig         `yaml:"redis"`
	RabbitMQ            RabbitMQConfig      `yaml:"rabbitmq"`
	MinIO               MinIOConfig         `yaml:"minio"`
	Demo                DemoConfig          `yaml:"demo"`
	ObservabilityConfig ObservabilityConfig `yaml:"observability"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type RabbitMQConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type MinIOConfig struct {
	Endpoint               string `yaml:"endpoint"`
	PublicEndpoint         string `yaml:"public_endpoint"`
	AccessKey              string `yaml:"access_key"`
	SecretKey              string `yaml:"secret_key"`
	Bucket                 string `yaml:"bucket"`
	Region                 string `yaml:"region"`
	UseSSL                 bool   `yaml:"use_ssl"`
	PublicUseSSL           bool   `yaml:"public_use_ssl"`
	SignedURLExpirySeconds int    `yaml:"signed_url_expiry_seconds"`
}

type DemoConfig struct {
	AllowPublicRegistration bool   `yaml:"allow_public_registration"`
	BootstrapUsername       string `yaml:"bootstrap_username"`
	BootstrapPassword       string `yaml:"bootstrap_password"`
}

func (c MinIOConfig) Validate() error {
	if c.Endpoint == "" {
		return errors.New("minio endpoint is required")
	}
	if c.PublicEndpoint == "" {
		return errors.New("minio public endpoint is required")
	}
	if c.AccessKey == "" {
		return errors.New("minio access key is required")
	}
	if c.SecretKey == "" {
		return errors.New("minio secret key is required")
	}
	if c.Bucket == "" {
		return errors.New("minio bucket is required")
	}
	if c.SignedURLExpirySeconds <= 0 {
		return errors.New("minio signed URL expiry must be positive")
	}
	return nil
}

type ObservabilityConfig struct {
	Pprof PprofConfig `yaml:"pprof"`
}
type PprofConfig struct {
	Enabled    bool   `yaml:"enabled"`
	ApiAddr    string `yaml:"api_addr"`
	WorkerAddr string `yaml:"worker_addr"`
}

func Load(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", filename, err)
	}

	ApplyEnvOverrides(&cfg)
	return cfg, nil
}

func ApplyEnvOverrides(cfg *Config) {
	if cfg == nil {
		return
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("MYSQL_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("MYSQL_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = port
		}
	}
	if v := os.Getenv("MYSQL_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("MYSQL_ROOT_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("MYSQL_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("MYSQL_DATABASE"); v != "" {
		cfg.Database.DBName = v
	}
	if v := os.Getenv("REDIS_HOST"); v != "" {
		cfg.Redis.Host = v
	}
	if v := os.Getenv("REDIS_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Redis.Port = port
		}
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if db, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = db
		}
	}
	if v := os.Getenv("RABBITMQ_HOST"); v != "" {
		cfg.RabbitMQ.Host = v
	}
	if v := os.Getenv("RABBITMQ_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.RabbitMQ.Port = port
		}
	}
	if v := os.Getenv("RABBITMQ_USER"); v != "" {
		cfg.RabbitMQ.Username = v
	}
	if v := os.Getenv("RABBITMQ_PASS"); v != "" {
		cfg.RabbitMQ.Password = v
	}
	if v := os.Getenv("MINIO_ENDPOINT"); v != "" {
		cfg.MinIO.Endpoint = v
	}
	if v := os.Getenv("MINIO_PUBLIC_ENDPOINT"); v != "" {
		cfg.MinIO.PublicEndpoint = v
	}
	if v := os.Getenv("MINIO_ACCESS_KEY"); v != "" {
		cfg.MinIO.AccessKey = v
	}
	if v := os.Getenv("MINIO_SECRET_KEY"); v != "" {
		cfg.MinIO.SecretKey = v
	}
	if v := os.Getenv("MINIO_BUCKET"); v != "" {
		cfg.MinIO.Bucket = v
	}
	if v := os.Getenv("MINIO_REGION"); v != "" {
		cfg.MinIO.Region = v
	}
	if v := os.Getenv("MINIO_USE_SSL"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			cfg.MinIO.UseSSL = enabled
		}
	}
	if v := os.Getenv("MINIO_PUBLIC_USE_SSL"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			cfg.MinIO.PublicUseSSL = enabled
		}
	}
	if v := os.Getenv("MINIO_SIGNED_URL_EXPIRY_SECONDS"); v != "" {
		if seconds, err := strconv.Atoi(v); err == nil {
			cfg.MinIO.SignedURLExpirySeconds = seconds
		}
	}
	if v := os.Getenv("ALLOW_PUBLIC_REGISTRATION"); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			cfg.Demo.AllowPublicRegistration = enabled
		}
	}
	if v := os.Getenv("BOOTSTRAP_USERNAME"); v != "" {
		cfg.Demo.BootstrapUsername = v
	}
	if v := os.Getenv("BOOTSTRAP_PASSWORD"); v != "" {
		cfg.Demo.BootstrapPassword = v
	}
}

// bool用来表示是否使用了默认配置，true表示使用了默认配置
func LoadLocalDev(filename string) (Config, bool, error) {
	cfg, err := Load(filename)
	if err == nil {
		return cfg, false, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return DefaultLocalConfig(), true, nil
	}
	return Config{}, false, err
}

func DefaultLocalConfig() Config {
	cfg := Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			User:     "root",
			Password: "123456",
			DBName:   "feedsystem",
		},
		Redis: RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "123456",
			DB:       0,
		},
		RabbitMQ: RabbitMQConfig{
			Host:     "localhost",
			Port:     5672,
			Username: "admin",
			Password: "password123",
		},
		MinIO: MinIOConfig{
			Endpoint:               "localhost:9002",
			PublicEndpoint:         "localhost:9002",
			AccessKey:              "minioadmin",
			SecretKey:              "minioadmin",
			Bucket:                 "feedsystem-media",
			Region:                 "us-east-1",
			UseSSL:                 false,
			PublicUseSSL:           false,
			SignedURLExpirySeconds: 7200,
		},
		Demo: DemoConfig{
			AllowPublicRegistration: false,
		},
		ObservabilityConfig: ObservabilityConfig{
			Pprof: PprofConfig{
				Enabled:    true,
				ApiAddr:    "localhost:6060",
				WorkerAddr: "localhost:6061",
			},
		},
	}
	ApplyEnvOverrides(&cfg)
	return cfg
}
