package config

import "testing"

func TestMinIOConfigValidate(t *testing.T) {
	valid := MinIOConfig{
		Endpoint:               "minio:9000",
		PublicEndpoint:         "media.example.com",
		AccessKey:              "access",
		SecretKey:              "secret",
		Bucket:                 "media",
		SignedURLExpirySeconds: 7200,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*MinIOConfig)
	}{
		{name: "endpoint", mutate: func(c *MinIOConfig) { c.Endpoint = "" }},
		{name: "public endpoint", mutate: func(c *MinIOConfig) { c.PublicEndpoint = "" }},
		{name: "access key", mutate: func(c *MinIOConfig) { c.AccessKey = "" }},
		{name: "secret key", mutate: func(c *MinIOConfig) { c.SecretKey = "" }},
		{name: "bucket", mutate: func(c *MinIOConfig) { c.Bucket = "" }},
		{name: "expiry", mutate: func(c *MinIOConfig) { c.SignedURLExpirySeconds = 0 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid
			tt.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
