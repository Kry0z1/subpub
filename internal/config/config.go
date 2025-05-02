package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	// one of "local", "prod"
	Env         string        `yaml:"env" env-default:"local"`
	StopTimeout time.Duration `yaml:"stop_timeout" env-default:"10s"`
	GRPC        GRPCConfig    `yaml:"grpc" env-required:"true"`
}

type GRPCConfig struct {
	Port    int           `yaml:"port" env-required:"true"`
	Timeout time.Duration `yaml:"timeout" env-default:"5s"`
}

func MustLoad() *Config {
	path := getConfigPath()
	return MustLoadPath(path)
}

func MustLoadPath(path string) *Config {
	if path == "" {
		panic("empty config path")
	}

	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic("couldn't read config: " + err.Error())
	}

	return &cfg
}

// Gets config path in this priority:
// param > env > default
//
// Environment variable is CONFIG_PATH.
// Default is empty string.
func getConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
