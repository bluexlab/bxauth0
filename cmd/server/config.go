package main

import "github.com/bluexlab/bxauth0/pkg/helper/configor"

type HostPort struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Config struct {
	HTTP     HostPort `yaml:"http"`
	GRPC     HostPort `yaml:"grpc"`
	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Name     string `yaml:"name"`
		Pool     int    `yaml:"pool"`
		SSLMode  string `yaml:"sslmode"`
	} `yaml:"database"`
	Endpoint     string `yaml:"endpoint"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	ClientEmail  string `yaml:"client_email"`
}

func LoadConfig(path string) (*Config, error) {
	cfg := &Config{}
	err := configor.FromFile(path, cfg)
	return cfg, err
}
