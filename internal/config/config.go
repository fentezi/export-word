package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	Env    string `yaml:"env"`
	Server Server `yaml:"server"`
	Kafka  Kafka  `yaml:"kafka"`
	Mongo  Mongo  `yaml:"mongo"`
	Gmail  Gmail
}

type Kafka struct {
	Address string `yaml:"address"`
	Port    string `yaml:"port"`
	Topic   string `yaml:"topic"`
}

type Mongo struct {
	Database string `yaml:"database"`
	Address  string `env:"MONGO_URL" env-required:"true"`
}

type Server struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type Gmail struct {
	Email    string `env:"GMAIL_EMAIL" env-required:"true"`
	Password string `env:"GMAIL_PASSWORD" env-required:"true"`
}

func MustConfig() Config {
	var cfg Config

	if err := godotenv.Load(); err != nil {
		panic(fmt.Errorf("failed to load .env file: %w", err))
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config.yml"
	}

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic(fmt.Errorf("failed to read config file from %s: %w", configPath, err))
	}

	return cfg
}
