package model

// Service represents a third-party web management service to proxy.
type Service struct {
	ID          string       `yaml:"id" json:"id"`
	Name        string       `yaml:"name" json:"name"`
	Icon        string       `yaml:"icon" json:"icon"`
	URL         string       `yaml:"url" json:"url"`
	Target      string       `yaml:"target" json:"-"`
	Category    string       `yaml:"category" json:"category"`
	OpenMethod  string       `yaml:"open_method" json:"open_method"` // iframe | browser
	PanelURL    string       `yaml:"panel_url" json:"panel_url,omitempty"`     // Direct URL for Tauri WebView (bypass proxy)
	HealthCheck *HealthCheck  `yaml:"health_check" json:"-"`
	Auth        *ServiceAuth  `yaml:"auth" json:"-"`
}

// ServiceAuth describes how to authenticate with the backend service.
type ServiceAuth struct {
	Type     string `yaml:"type"`     // "basic" for Basic Auth
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// HealthCheck describes how to check if a service is alive.
type HealthCheck struct {
	Type     string `yaml:"type"`     // http | tcp
	Path     string `yaml:"path,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	Interval string `yaml:"interval"` // e.g. "30s"
}

// ServiceStatus is the live runtime status of a proxied service.
type ServiceStatus struct {
	Service
	Status    string `json:"status"` // online | degraded | offline
	LastCheck int64  `json:"last_check"`
	Error     string `json:"error,omitempty"`
}
