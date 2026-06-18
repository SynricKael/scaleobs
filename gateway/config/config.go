package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/glrs/observer/gateway/model"
	"gopkg.in/yaml.v3"
)

// GatewayConfig is the top-level configuration structure.
type GatewayConfig struct {
	Gateway           GatewaySection            `yaml:"gateway"`
	Servers           []model.ServerConfig            `yaml:"servers"`
	Services          []model.Service           `yaml:"services"`
	HeadscaleNetworks []HeadscaleNetworkConfig  `yaml:"headscale_networks,omitempty"`
	DockerHosts       []DockerHostConfig        `yaml:"docker_hosts,omitempty"`
	HostAgents        map[string][]string       `yaml:"host_agents,omitempty"`
	AgentServers      []model.AgentServerConfig `yaml:"agent_servers,omitempty"`
	ServerOverrides   []model.ServerOverride          `yaml:"server_overrides,omitempty"`
}

// DockerHostConfig defines a remote Docker daemon to monitor via its TCP API.
type DockerHostConfig struct {
	Name string `yaml:"name"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// HeadscaleNetworkConfig defines a single Headscale source for node discovery.
type HeadscaleNetworkConfig struct {
	Name   string `yaml:"name"`
	URL    string `yaml:"url"`
	APIKey string `yaml:"api_key"`
}

// GatewaySection holds gateway-level settings.
type GatewaySection struct {
	Title string      `yaml:"title"`
	Port  int         `yaml:"port"`
	Auth  AuthSection `yaml:"auth"`
}

// AuthSection holds authentication settings.
type AuthSection struct {
	Enabled bool       `yaml:"enabled"`
	JWTSecret string  `yaml:"jwt_secret"`
	Users    []UserDef `yaml:"users"`
}

// UserDef defines a dashboard user.
type UserDef struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Load reads and parses the YAML config file, resolving env vars.
func Load(path string) (*GatewayConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Resolve ${VAR} environment variables
	resolved := os.Expand(string(data), func(key string) string {
		if val, ok := os.LookupEnv(key); ok {
			return val
		}
		return "${" + key + "}"
	})

	var cfg GatewayConfig
	decoder := yaml.NewDecoder(strings.NewReader(resolved))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Apply defaults
	if cfg.Gateway.Port == 0 {
		cfg.Gateway.Port = 8080
	}
	if cfg.Gateway.Title == "" {
		cfg.Gateway.Title = "Server Ops Portal"
	}

	// Validate required fields
	if cfg.Gateway.Auth.Enabled && cfg.Gateway.Auth.JWTSecret == "" {
		return nil, fmt.Errorf("config: auth enabled but jwt_secret is empty")
	}
	if cfg.Gateway.Auth.Enabled && len(cfg.Gateway.Auth.Users) == 0 {
		return nil, fmt.Errorf("config: auth enabled but no users defined")
	}
	for i, s := range cfg.Services {
		if s.ID == "" {
			return nil, fmt.Errorf("services[%d]: id is required", i)
		}
		if s.Target == "" {
			return nil, fmt.Errorf("services[%s]: target is required", s.ID)
		}
	}

	return &cfg, nil
}

// PortStr returns the listen address string.
func (c *GatewayConfig) PortStr() string {
	return fmt.Sprintf(":%d", c.Gateway.Port)
}
