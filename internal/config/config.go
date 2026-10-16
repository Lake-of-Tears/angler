package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigDir  = ".angler"
	DefaultConfigFile = "config.yaml"
	DefaultEndpoint   = "http://localhost:8000"
	DefaultContext    = "default"
)

type Context struct {
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
	Token    string `yaml:"token" mapstructure:"token"`
	Email    string `yaml:"email" mapstructure:"email"`
	Timeout  int    `yaml:"timeout" mapstructure:"timeout"`
}

type Config struct {
	CurrentContext string             `yaml:"current_context" mapstructure:"current_context"`
	Contexts       map[string]Context `yaml:"contexts" mapstructure:"contexts"`
}

func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, DefaultConfigDir)
	return dir, nil
}

func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultConfigFile), nil
}

func LoadConfig() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		CurrentContext: DefaultContext,
		Contexts:       make(map[string]Context),
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Default initial context
		cfg.Contexts[DefaultContext] = Context{
			Endpoint: DefaultEndpoint,
			Timeout:  30,
		}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.CurrentContext == "" {
		cfg.CurrentContext = DefaultContext
	}
	if cfg.Contexts == nil {
		cfg.Contexts = make(map[string]Context)
	}

	return cfg, nil
}

func SaveConfig(cfg *Config) error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	path := filepath.Join(dir, DefaultConfigFile)
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", path, err)
	}

	return nil
}

// GetActiveContext returns the active context after applying environment overrides.
func (c *Config) GetActiveContext(overrideContext string) Context {
	ctxName := c.CurrentContext
	if overrideContext != "" {
		ctxName = overrideContext
	}
	if envCtx := os.Getenv("ANGLER_CONTEXT"); envCtx != "" && overrideContext == "" {
		ctxName = envCtx
	}

	ctx, exists := c.Contexts[ctxName]
	if !exists {
		ctx = Context{
			Endpoint: DefaultEndpoint,
			Timeout:  30,
		}
	}

	if envEndpoint := os.Getenv("ANGLER_ENDPOINT"); envEndpoint != "" {
		ctx.Endpoint = envEndpoint
	}
	if envToken := os.Getenv("ANGLER_TOKEN"); envToken != "" {
		ctx.Token = envToken
	}

	return ctx
}

func SetupViper() {
	viper.SetEnvPrefix("ANGLER")
	viper.AutomaticEnv()
}
