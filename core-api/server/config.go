package server

import (
	"fmt"
	"net/url"
	"path"
	"time"

	"github.com/caarlos0/env/v11"
)

// Configuration for the server, loaded from environment variables
type Config struct {
	Port                  int           `env:"CORE_API_PORT" envDefault:"8000"`
	HTTPReadHeaderTimeout time.Duration `env:"CORE_API_READ_HEADER_TIMEOUT" envDefault:"5s"`
	HTTPReadTimeout       time.Duration `env:"CORE_API_READ_TIMEOUT" envDefault:"15s"`
	HTTPWriteTimeout      time.Duration `env:"CORE_API_WRITE_TIMEOUT" envDefault:"15s"`
	HTTPIdleTimeout       time.Duration `env:"CORE_API_IDLE_TIMEOUT" envDefault:"60s"`
	UploadMaxBytes        int64         `env:"CORE_API_UPLOAD_MAX_BYTES" envDefault:"52428800"`
	DownloadURLExpires    time.Duration `env:"CORE_API_DOWNLOAD_URL_EXPIRES" envDefault:"15m"`
	TranscodeJobPriority  int           `env:"CORE_API_TRANSCODE_JOB_PRIORITY" envDefault:"1"`
	S3Endpoint            string        `env:"S3_ENDPOINT,required"`
	S3Region              string        `env:"S3_REGION" envDefault:"ru-1"`
	S3Bucket              string        `env:"S3_BUCKET,required"`
	S3AccessKey           string        `env:"S3_ACCESS_KEY,required"`
	S3SecretKey           string        `env:"S3_SECRET_KEY,required"`
	S3PublicBaseURL       string        `env:"S3_PUBLIC_BASE_URL,required"`
	KafkaBrokers          string        `env:"KAFKA_BROKERS" envDefault:"localhost:9094"`
	KafkaTranscodeTopic   string        `env:"KAFKA_TRANSCODE_TOPIC" envDefault:"transcode-jobs"`
	KafkaWriteTimeout     time.Duration `env:"KAFKA_WRITE_TIMEOUT" envDefault:"10s"`
	KafkaReadTimeout      time.Duration `env:"KAFKA_READ_TIMEOUT" envDefault:"10s"`
	PostgresHost          string        `env:"POSTGRES_HOST" envDefault:"localhost"`
	PostgresPort          int           `env:"POSTGRES_PORT" envDefault:"5432"`
	PostgresUser          string        `env:"POSTGRES_USER,required"`
	PostgresPassword      string        `env:"POSTGRES_PASSWORD,required"`
	PostgresDB            string        `env:"POSTGRES_DB,required"`
	PostgresSSLMode       string        `env:"POSTGRES_SSLMODE" envDefault:"disable"`
}

// Build a public URL for an object in S3 based on the configured base URL and the object key
func (c *Config) BuildPublicObjectURL(objectKey string) (string, error) {
	if c.S3PublicBaseURL == "" {
		return "", nil
	}

	base, err := url.Parse(c.S3PublicBaseURL)
	if err != nil {
		return "", err
	}

	base.Path = path.Join(base.Path, objectKey)
	return base.String(), nil
}

// Build a database connection URL for PostgreSQL
func (c *Config) DatabaseURL() string {
	dbURL := &url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%d", c.PostgresHost, c.PostgresPort),
		Path:   c.PostgresDB,
		User:   url.UserPassword(c.PostgresUser, c.PostgresPassword),
	}

	query := url.Values{}
	query.Set("sslmode", c.PostgresSSLMode)
	dbURL.RawQuery = query.Encode()

	return dbURL.String()
}

// DI constructor for Config
func NewConfig() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg)

	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
