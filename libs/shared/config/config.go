package config

import (
	"github.com/caarlos0/env/v11"
)

type AppConfig struct {
	Postgres PostgresConfig `envPrefix:"DB_"`
	Redis    RedisConfig    `envPrefix:"REDIS_"`
	Kafka    KafkaConfig    `envPrefix:"KAFKA_"`
	Service  ServiceConfig  `envPrefix:"SERVICE_"`
}

type PostgresConfig struct {
	Host     string `env:"HOST,required"`
	Port     string `env:"PORT,required"`
	User     string `env:"USER,required"`
	Password string `env:"PASSWORD,required"`
	Name     string `env:"NAME,required"`
	SSLMode  string `env:"SSL_MODE" envDefault:"disable"`
}

type RedisConfig struct {
	Host     string `env:"HOST,required"`
	Port     string `env:"PORT,required"`
	Password string `env:"PASSWORD"`
}

type KafkaConfig struct {
	Brokers []string `env:"BROKERS,required"`
	GroupID string   `env:"GROUP_ID,required"`
}

type ServiceConfig struct {
	Port    string `env:"PORT" envDefault:"8080"`
	LogJSON bool   `env:"LOG_JSON" envDefault:"true"`
}

func Load() (*AppConfig, error) {
	cfg := &AppConfig{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
