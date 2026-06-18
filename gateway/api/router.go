package api

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/glrs/observer/gateway/auth"
	"github.com/glrs/observer/gateway/config"
	"github.com/glrs/observer/gateway/monitor"
	"github.com/glrs/observer/gateway/plugin"
	"github.com/glrs/observer/gateway/proxy"
)

// Router sets up all HTTP routes for the Gateway.
type Router struct {
	mux            *http.ServeMux
	proxyRouter    *proxy.ProxyRouter
	healthChecker  *monitor.HealthChecker
	agentHub       *monitor.AgentHub
	cfg            *config.GatewayConfig
}

// NewRouter creates a Router with all routes registered.
func NewRouter(cfg *config.GatewayConfig, registry *plugin.Registry, hc *monitor.HealthChecker, hub *monitor.AgentHub, configPath string, agentSrvHandler *AgentServerHandler) *Router {
	r := &Router{
		mux:           http.NewServeMux(),
		proxyRouter:   proxy.NewRouter(),
		healthChecker: hc,
		agentHub:      hub,
		cfg:           cfg,
	}

	r.registerRoutes(cfg, registry, configPath, agentSrvHandler)
	return r
}

// Handler returns the http.Handler (with auth middleware applied).
func (r *Router) Handler() http.Handler {
	// Only these API endpoints require JWT authentication.
	// Everything else (SPA, assets, proxy, WebSocket, health) is public.
	protectPaths := []string{
		"/api/services",
		"/api/servers",
		"/api/docker/",
	}

	if !r.cfg.Gateway.Auth.Enabled {
		return r.mux
	}

	return auth.Middleware(r.cfg.Gateway.Auth.JWTSecret, protectPaths)(r.mux)
}

func (r *Router) registerRoutes(cfg *config.GatewayConfig, registry *plugin.Registry, configPath string, agentSrvHandler *AgentServerHandler) {
	// Register proxy routes for each service
	for _, svc := range cfg.Services {
		if err := r.proxyRouter.Register(&svc); err != nil {
			log.Printf("[router] failed to register proxy for %s: %v", svc.ID, err)
			continue
		}
		log.Printf("[router] registered proxy: /p/%s/ -> %s", svc.ID, svc.Target)
	}

	// Auth
	authHandler := NewAuthHandler(&cfg.Gateway.Auth)
	r.mux.HandleFunc("/api/auth/login", authHandler.Login)

	// Services
	svcHandler := NewServiceHandler(registry, r.healthChecker)
	r.mux.HandleFunc("/api/services", svcHandler.ListServices)

	// Servers
	srvHandler := NewServerHandler(r.agentHub)
	r.mux.HandleFunc("/api/servers", srvHandler.ListServers)

	// Config (read/write services.yml)
	cfgHandler := NewConfigHandler(configPath)
	r.mux.HandleFunc("/api/config", cfgHandler.HandleConfig)

	// Agent download
	r.mux.HandleFunc("/api/agent/platforms", AgentPlatformsHandler)
	r.mux.HandleFunc("/api/agent/download/", AgentDownloadHandler)

	// Agent Servers (coding agent UIs)
	r.mux.HandleFunc("/api/agent-servers", agentSrvHandler.ListAgentServers)

	// Proxy catch-all: /p/{serviceId}/...
	r.mux.HandleFunc("/p/", r.proxyRouter.ServeHTTP)

	// Health check endpoint (no auth)
	r.mux.HandleFunc("/api/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","version":"0.1.0"}`))
	})

	// Docker container management (requires auth)
	dockerHandler := NewDockerHandler()
	r.mux.HandleFunc("/api/docker/", dockerHandler.HandleContainerAction)

	// Server settings (per-server SSH, group, agent-server link)
	settingsHandler := NewServerSettingsHandler(configPath, r.agentHub)
	r.mux.HandleFunc("/api/servers/", settingsHandler.HandleSettingsUpdate)

	// Dashboard static files — SPA-compatible FileServer at root
	dashboardCandidates := []string{
		"/app/dashboard",
		"dashboard/dist",
		"../dashboard/dist",
	}
	var dashboardFound bool
	for _, dir := range dashboardCandidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			r.mux.Handle("/", &spaFileServer{root: dir})
			log.Printf("[router] serving dashboard (SPA) from %s", dir)
			dashboardFound = true
			break
		}
	}
	if !dashboardFound {
		log.Println("[router] WARNING: dashboard dist not found, static files won't be served")
		// Fallback root handler
		r.mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			w.Write([]byte("SOP Gateway running — dashboard not built yet"))
		})
	}
}

// spaFileServer serves static files with SPA fallback:
// existing files are served directly, otherwise index.html is served
// (client-side routing fallback).
type spaFileServer struct {
	root string
}

func (s *spaFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Never cache the SPA — every load gets fresh JS/CSS
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")

	// SPA fallback: if the requested file doesn't exist, serve index.html
	// so that client-side routing works (e.g. /login, /dashboard, etc.)
	path := filepath.Join(s.root, r.URL.Path)
	if fi, err := os.Stat(path); err != nil || fi.IsDir() {
		r.URL.Path = "/"
	}
	http.FileServer(http.Dir(s.root)).ServeHTTP(w, r)
}

// CORSMiddleware adds CORS headers for development.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}


