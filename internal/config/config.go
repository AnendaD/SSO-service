package config

import (
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env             string        `yaml:"env" env-default:"local"`
	StoragePath     string        `yaml:"storage_path" env-required:"true"`
	TokenTTL        time.Duration `yaml:"token_ttl" env-required:"true"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl" env-default:"720h"`
	GRPC            GRPCConfig    `yaml:"grpc"`
}

type GRPCConfig struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

func MustLoad() *Config {
	configPath := fetchConfigPath()
	if configPath == "" {
		panic("config path is empty")
	}

	return MustLoadPath(configPath)
}

func MustLoadPath(configPath string) *Config {
	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		content, _ := os.ReadFile(configPath)
		panic("cannot read config: " + err.Error() + "\nFile content:\n" + string(content))
	}

	// Override with environment variables

	// DATABASE_URL override
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.StoragePath = dbURL
	}

	// PORT override (for Koyeb, Heroku, etc.)
	if portStr := os.Getenv("PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			cfg.GRPC.Port = port
		}
	}

	// GRPC_PORT override (alternative to PORT)
	if portStr := os.Getenv("GRPC_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			cfg.GRPC.Port = port
		}
	}

	// ENV override
	if env := os.Getenv("ENV"); env != "" {
		cfg.Env = env
	}

	// TOKEN_TTL override
	if ttlStr := os.Getenv("TOKEN_TTL"); ttlStr != "" {
		if ttl, err := time.ParseDuration(ttlStr); err == nil {
			cfg.TokenTTL = ttl
		}
	}

	// REFRESH_TOKEN_TTL override
	if ttlStr := os.Getenv("REFRESH_TOKEN_TTL"); ttlStr != "" {
		if ttl, err := time.ParseDuration(ttlStr); err == nil {
			cfg.RefreshTokenTTL = ttl
		}
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
