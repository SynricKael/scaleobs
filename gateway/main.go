package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/glrs/observer/gateway/api"
	"github.com/glrs/observer/gateway/config"
	"github.com/glrs/observer/gateway/monitor"
	"github.com/glrs/observer/gateway/plugin"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("[gateway] starting Server Ops Portal Gateway...")

	// Determine config path
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "/app/config.yml"
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("[gateway] failed to load config: %v", err)
	}
	log.Printf("[gateway] loaded config: title=%q, services=%d, servers=%d",
		cfg.Gateway.Title, len(cfg.Services), len(cfg.Servers))

	// Initialize plugin registry
	registry := plugin.NewRegistry(cfg.Services)
	log.Printf("[gateway] registered %d services in %d categories",
		len(cfg.Services), len(registry.Categories()))

	// Build agent token map
	agentTokens := make(map[string]string)
	for _, s := range cfg.Servers {
		agentTokens[s.ID] = s.AgentToken
	}

	// Add shared agent token as fallback for any server ID
	if sharedToken := os.Getenv("AGENT_TOKEN"); sharedToken != "" {
		agentTokens["*"] = sharedToken
		log.Println("[main] shared AGENT_TOKEN enabled — any server can connect with this token")
	}

	// Initialize agent hub (pre-populated with configured servers)
	agentHub := monitor.NewAgentHub(cfg.Servers, agentTokens)

	// Start Headscale sync (discovers nodes from all configured Headscale networks)
	hsSyncer := monitor.NewHeadscaleSyncer(agentHub, cfg.HeadscaleNetworks)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hsSyncer.Start(30*time.Second, ctx.Done())

	// Apply coding agent annotations
	if len(cfg.HostAgents) > 0 {
		agentHub.SetHostAgents(cfg.HostAgents)
		log.Printf("[main] applied coding agent annotations for %d host(s)", len(cfg.HostAgents))
	}

	// Apply server overrides (group, SSH settings)
	if len(cfg.ServerOverrides) > 0 {
		agentHub.SetOverrides(cfg.ServerOverrides)
		log.Printf("[main] loaded %d server override(s)", len(cfg.ServerOverrides))
	}

	// Start Docker host monitoring (polls remote Docker daemons for containers)
	if len(cfg.DockerHosts) > 0 {
		dockerHub := monitor.NewDockerHub(cfg.DockerHosts)
		agentHub.SetDockerHub(dockerHub)
		go dockerHub.Start(15*time.Second, ctx.Done())
		log.Printf("[main] monitoring %d remote Docker host(s)", len(cfg.DockerHosts))
	}

	// Initialize health checker
	healthChecker := monitor.NewHealthChecker(registry)
	go healthChecker.Start(ctx)

	// Create and start HTTP server
	agentSrvHandler := api.NewAgentServerHandler(cfg.AgentServers)
	agentSrvHandler.SetConfigPath(configPath)
	router := api.NewRouter(cfg, registry, healthChecker, agentHub, configPath, agentSrvHandler)

	handler := api.CORSMiddleware(router.Handler())

	// Register WebSocket endpoint for agents
	// Wrap the handler to add the WebSocket route
	mainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/ws/agent" {
			agentHub.HandleWS(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:         cfg.PortStr(),
		Handler:      mainHandler,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("[gateway] shutting down...")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		server.Shutdown(shutdownCtx)
	}()

	log.Printf("[gateway] listening on %s", cfg.PortStr())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[gateway] server error: %v", err)
	}
	log.Println("[gateway] stopped")
}
