package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/codingconcepts/env"
	"github.com/joho/godotenv"
)

type EnvVars struct {
	Redis        Redis
	Postgres     Postgres
	MsgWorker    MessageWorker
	Notification Notification
	Common       Common
}

type Common struct {
	MessageChannelSize int `env:"MESSAGE_CHANNEL_SIZE" envDefault:"100"`
}

type MessageWorker struct {
	MaxRecordsPerRead         int     `env:"MAX_RECORDS_PER_RECORD" default:"10"`
	MessageChannelSendTimeout int     `env:"MESSAGE_CHANNEL_SEND_TIMEOUT" default:"10"`
	RateLimitPerMinute        int     `env:"RATE_LIMIT_PER_MINUTE" default:"1"`
	RateLimitBurst            int     `env:"RATE_LIMIT_BURST" default:"2"`
	RedisMessageTTLSeconds    int     `env:"REDIS_MESSAGE_TTL_SECONDS" default:"86400"`
	BackpressureThreshold     float64 `env:"BACKPRESSURE_THRESHOLD" default:"0.9"`
	BackpressurePauseSeconds  int     `env:"BACKPRESSURE_PAUSE_SECONDS" default:"5"`
}

type Redis struct {
	Address            string        `env:"REDIS_ADDRESS" required:"true"`
	Username           string        `env:"REDIS_USERNAME"`
	Password           string        `env:"REDIS_PASSWORD"`
	DB                 int           `env:"REDIS_DB"`
	DialTimeout        time.Duration `env:"REDIS_DIAL_TIMEOUT" default:"5s"`
	ReadTimeout        time.Duration `env:"REDIS_READ_TIMEOUT" default:"10s"`
	WriteTimeout       time.Duration `env:"REDIS_WRITE_TIMEOUT" default:"10s"`
	PoolSize           int           `env:"REDIS_POOL_SIZE" default:"10"`
	MinIdleConnections int           `env:"REDIS_MIN_IDLE_CONNECTIONS" default:"5"`
	MaxConnectionAge   time.Duration `env:"REDIS_MAX_CONNECTION_AGE" default:"5m"`
	IdleTimeout        time.Duration `env:"REDIS_IDLE_TIMEOUT" default:"5m"`
}

type Postgres struct {
	Host     string `env:"POSTGRES_HOST" default:"localhost"`
	Port     int    `env:"POSTGRES_PORT" default:"5432"`
	User     string `env:"POSTGRES_USER" required:"true"`
	Password string `env:"POSTGRES_PASSWORD" required:"true"`
	Database string `env:"POSTGRES_DB" required:"true"`
	SSLMode  string `env:"POSTGRES_SSLMODE" default:"disable"`
}

type Notification struct {
	WebhookBaseUrl          string `env:"WEBHOOK_BASE_URL" required:"true"`
	WebhookEndpoint         string `env:"WEBHOOK_ENDPOINT" required:"true"`
	WebhookApiKey           string `env:"WEBHOOK_API_KEY"`
	HttpTimeoutSeconds      int    `env:"HTTP_TIMEOUT_SECONDS" default:"30"`
	HttpMaxIdleConns        int    `env:"HTTP_MAX_IDLE_CONNS" default:"100"`
	HttpMaxIdleConnsPerHost int    `env:"HTTP_MAX_IDLE_CONNS_PER_HOST" default:"10"`
	HttpIdleConnTimeout     int    `env:"HTTP_IDLE_CONN_TIMEOUT" default:"90"`
	HttpDisableCompression  bool   `env:"HTTP_DISABLE_COMPRESSION" default:"false"`
	HttpDisableKeepAlives   bool   `env:"HTTP_DISABLE_KEEP_ALIVES" default:"false"`
	MaxRetries              int    `env:"MAX_RETRIES" default:"3"`
	InitialRetryDelayMs     int    `env:"INITIAL_RETRY_DELAY_MS" default:"100"`
	MaxRetryDelayMs         int    `env:"MAX_RETRY_DELAY_MS" default:"10000"`
	RateLimitPerSecond      int    `env:"WEBHOOK_RATE_LIMIT_PER_SECOND" default:"10"`
	RateLimitBurst          int    `env:"WEBHOOK_RATE_LIMIT_BURST" default:"20"`
}

func LoadEnvVars() (*EnvVars, error) {
	err := godotenv.Load(".env")

	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error loading .env file: %s", err.Error())
	}

	r := Redis{}
	if err := env.Set(&r); err != nil {
		return nil, fmt.Errorf("loading redis environment variables failed, %s", err.Error())
	}

	p := Postgres{}
	if err := env.Set(&p); err != nil {
		return nil, fmt.Errorf("loading postgres environment variables failed, %s", err.Error())
	}
	mspW := MessageWorker{}

	if err := env.Set(&mspW); err != nil {
		return nil, fmt.Errorf("loading message worker environment variables failed, %s", err.Error())
	}

	n := Notification{}
	if err := env.Set(&n); err != nil {
		return nil, fmt.Errorf("loading notification environment variables failed, %s", err.Error())
	}

	c := Common{}
	if err := env.Set(&c); err != nil {
		return nil, fmt.Errorf("loading notification environment variables failed, %s", err.Error())
	}

	envVars := &EnvVars{
		Redis:        r,
		Postgres:     p,
		MsgWorker:    mspW,
		Notification: n,
		Common:       c,
	}

	return envVars, nil

}
