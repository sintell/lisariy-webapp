package config

import (
	"sync"
	"time"

	"github.com/micro/go-config"
	"github.com/micro/go-config/source/file"
)

// Config represents base config file for application
type Config struct {
	WebApp struct {
		Port            string        `json:"port,omitempty"`
		ShutdownTimeout time.Duration `json:"shutdownTimeout,omitempty"`
		AdminLogin      string        `json:"adminLogin,omitempty"`
		AdminPassword   string        `json:"adminPassword,omitempty"`
	} `json:"webApp,omitempty"`
	DB struct {
		Host     string `json:"host,omitempty"`
		Port     string `json:"port,omitempty"`
		User     string `json:"user,omitempty"`
		DBName   string `json:"dbName,omitempty"`
		Password string `json:"password,omitempty"`
		Debug    bool   `json:"debug,omitempty"`
	} `json:"db,omitempty"`
	Log struct {
		Level           string `json:"level,omitempty"`
		LogsPath        string `json:"logsPath,omitempty"`
		AlsoLogToStdOut bool   `json:"alsoLogToStdOut,omitempty"`
	} `json:"log,omitempty"`
}

// AppConfig is syncronizing entity
type AppConfig struct {
	mx   sync.RWMutex
	conf *Config
}

// New creates new AppConfig instance
func New(configPath string) (*AppConfig, error) {
	cfg := &AppConfig{mx: sync.RWMutex{}, conf: &Config{}}
	if err := cfg.Read(configPath); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Read loads config into memmory
func (ac *AppConfig) Read(configPath string) error {
	ac.mx.Lock()
	defer ac.mx.Unlock()
	config := config.NewConfig()
	err := config.Load(
		file.NewSource(file.WithPath(configPath)),
	)
	if err != nil {
		return err
	}
	err = config.Scan(ac.conf)
	if err != nil {
		return err
	}

	return nil
}

// Access is a synchronisation point for config access
func (ac *AppConfig) Access() *Config {
	ac.mx.RLock()
	defer ac.mx.RUnlock()

	conf := ac.conf
	return conf
}
