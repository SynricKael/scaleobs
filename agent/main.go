package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/glrs/observer/agent/collector"
	"github.com/glrs/observer/agent/model"
	"github.com/glrs/observer/agent/reporter"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("[agent] starting Server Ops Portal Agent...")

	// Parse flags and env vars
	gatewayURL := getEnv("GATEWAY_URL", "ws://localhost:8080")
	serverID := getEnv("SERVER_ID", "main-server")
	token := getEnv("AGENT_TOKEN", "agent-secret")
	intervalStr := getEnv("COLLECT_INTERVAL", "10s")

	// Allow CLI flags to override
	flag.StringVar(&gatewayURL, "gateway", gatewayURL, "Gateway WebSocket URL")
	flag.StringVar(&serverID, "server-id", serverID, "Server ID")
	flag.StringVar(&token, "token", token, "Authentication token")
	flag.StringVar(&intervalStr, "interval", intervalStr, "Collection interval (e.g. 10s, 30s)")
	flag.Parse()

	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		log.Fatalf("[agent] invalid interval %q: %v", intervalStr, err)
	}

	log.Printf("[agent] config: gateway=%s server=%s interval=%s",
		gatewayURL, serverID, interval)

	// Initialize collectors
	cpuCollector := collector.NewCPUCollector()
	netCollector := collector.NewNetworkCollector()

	// Build collect function
	collectFn := func() (*model.Metrics, error) {
		// CPU
		cpuPercent, err := cpuCollector.Percent()
		if err != nil {
			log.Printf("[agent] cpu collect error: %v", err)
			cpuPercent = 0
		}

		// Memory
		memInfo, err := collector.CollectMemory()
		if err != nil {
			log.Printf("[agent] memory collect error: %v", err)
			memInfo = &collector.MemoryInfo{}
		}

		// Disk
		disks, err := collector.CollectDisks()
		if err != nil {
			log.Printf("[agent] disk collect error: %v", err)
			disks = nil
		}

		// Network I/O
		sentPerSec, recvPerSec := netCollector.Rates()

		// Docker containers
		containers, dockerStats := collector.CollectDockerContainers()

		// Coding agent detection
		agents := collector.DetectAgents()

		// Build model
		diskInfos := make([]model.DiskInfo, 0, len(disks))
		for _, d := range disks {
			diskInfos = append(diskInfos, model.DiskInfo{
				Mount:   d.Mount,
				TotalGB: d.TotalGB,
				UsedGB:  d.UsedGB,
				Percent: d.Percent,
			})
		}

		metrics := &model.Metrics{
			CPUPercent: cpuPercent,
			Memory: model.MemoryInfo{
				TotalMB: memInfo.TotalMB,
				UsedMB:  memInfo.UsedMB,
				Percent: memInfo.Percent,
			},
			Disks:            diskInfos,
			Network: &model.NetworkInfo{
				BytesSentPerSec: sentPerSec,
				BytesRecvPerSec: recvPerSec,
			},
			DockerContainers: containers,
			DockerStats:      dockerStats,
			Agents:           agents,
			Timestamp:        time.Now().Unix(),
		}

		return metrics, nil
	}

	// Create reporter and run
	r := reporter.NewWSReporter(gatewayURL, serverID, token)

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	stop := make(chan struct{})

	go func() {
		<-sigCh
		log.Println("[agent] shutting down...")
		close(stop)
	}()

	// Check if we should run once (for testing) or continuously
	if os.Getenv("RUN_ONCE") == "true" {
		metrics, err := collectFn()
		if err != nil {
			log.Fatalf("[agent] collect failed: %v", err)
		}
		log.Printf("[agent] metrics: cpu=%.1f%% mem=%.1f%% disks=%d agents=%v",
			metrics.CPUPercent, metrics.Memory.Percent, len(metrics.Disks), metrics.Agents)
		return
	}

	// Run continuous reporting with reconnection
	r.RunWithReconnect(interval, collectFn, stop)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
