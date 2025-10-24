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
	Redis    Redis
	Postgres Postgres
}

type Common struct {
	KeyValProvider string `env:"KEY_VAL_PROVIDER" default:"redis"`
	BaseURL        string `env:"BASE_URL" default:"http://127.0.0.1:8080"`
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

	envVars := &EnvVars{
		Redis:    r,
		Postgres: p,
	}

	return envVars, nil

}
