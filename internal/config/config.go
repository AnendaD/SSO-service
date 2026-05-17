package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env             string        `yaml:"env" env:"ENV" env-default:"local"`
	Database        Database      `yaml:"database"`
	TokenTTL        time.Duration `yaml:"token_ttl" env:"TOKEN_TTL" env-default:"24h"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl" env:"REFRESH_TOKEN_TTL" env-default:"720h"`
	GRPC            GRPCConfig    `yaml:"grpc"`
	HTTP            HTTPConfig    `yaml:"http"`
	TimeoutDuration time.Duration `yaml:"timeout_duration" env-default:"1m"`
}

type GRPCConfig struct {
	Port    int           `yaml:"port" env-default:"8080"`
	Timeout time.Duration `yaml:"timeout" env-default:"5s"`
}

type HTTPConfig struct {
	Port int `yaml:"port" env-default:"8081"`
}

type Database struct {
	URL      string `yaml:"storage_path" env:"DATABASE_URL"`
	MaxConns int    `yaml:"max_conns" env-default:"30"`
	MinConns int    `yaml:"min_conns" env-default:"1"`
}

func Load() *Config {
	configPath := fetchConfigPath()
	if configPath == "" {
		panic("config path is empty")
	}

	return MustLoadPath(configPath)
}

func MustLoadPath(configPath string) *Config {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("cannot read config: " + err.Error())
	}
	return &cfg
}

func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()
	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}
	return res
}
