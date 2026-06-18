package monitor

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/glrs/observer/gateway/model"
	"github.com/glrs/observer/gateway/plugin"
)

// HealthChecker periodically checks the health of registered services.
type HealthChecker struct {
	registry *plugin.Registry
	statuses map[string]*model.ServiceStatus
	mu       sync.RWMutex
	client   *http.Client
}

// NewHealthChecker creates a HealthChecker.
func NewHealthChecker(registry *plugin.Registry) *HealthChecker {
	return &HealthChecker{
		registry: registry,
		statuses: make(map[string]*model.ServiceStatus),
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        20,
				IdleConnTimeout:     30 * time.Second,
				DisableKeepAlives:   false,
			},
		},
	}
}

// Start begins periodic health checks. Blocks until ctx is cancelled.
func (hc *HealthChecker) Start(ctx context.Context) {
	log.Println("[health] health checker started")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Run an initial check immediately
	hc.checkAll()

	for {
		select {
		case <-ticker.C:
			hc.checkAll()
		case <-ctx.Done():
			log.Println("[health] health checker stopped")
			return
		}
	}
}

// checkAll performs a health check on every registered service.
func (hc *HealthChecker) checkAll() {
	for _, svc := range hc.registry.List() {
		status := hc.check(svc)
		hc.mu.Lock()
		hc.statuses[svc.ID] = status
		hc.mu.Unlock()
	}
}

// check performs a single health check on a service.
func (hc *HealthChecker) check(svc *model.Service) *model.ServiceStatus {
	status := &model.ServiceStatus{
		Service:   *svc,
		Status:    "offline",
		LastCheck: time.Now().Unix(),
	}

	if svc.HealthCheck == nil {
		status.Status = "online" // No health check configured, assume online
		return status
	}

	switch svc.HealthCheck.Type {
	case "http":
		hc.checkHTTP(svc, status)
	case "tcp":
		hc.checkTCP(svc, status)
	default:
		status.Status = "online"
	}

	return status
}

// checkHTTP performs an HTTP health check.
func (hc *HealthChecker) checkHTTP(svc *model.Service, status *model.ServiceStatus) {
	checkPath := svc.HealthCheck.Path
	if checkPath == "" {
		checkPath = "/"
	}

	checkURL := svc.Target + checkPath

	resp, err := hc.client.Get(checkURL)
	if err != nil {
		status.Status = "offline"
		status.Error = fmt.Sprintf("HTTP error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		status.Status = "online"
		status.Error = ""
	} else {
		status.Status = "degraded"
		status.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
}

// checkTCP performs a TCP dial health check.
func (hc *HealthChecker) checkTCP(svc *model.Service, status *model.ServiceStatus) {
	targetHost := svc.Target
	port := svc.HealthCheck.Port
	if port == 0 {
		// Try to extract port from target URL
		status.Status = "offline"
		status.Error = "no port specified for TCP health check"
		return
	}

	addr := net.JoinHostPort(targetHost, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		status.Status = "offline"
		status.Error = fmt.Sprintf("TCP error: %v", err)
		return
	}
	conn.Close()
	status.Status = "online"
}

// GetStatus returns the current status of all services.
func (hc *HealthChecker) GetStatus() []*model.ServiceStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result := make([]*model.ServiceStatus, 0, len(hc.statuses))
	for _, s := range hc.statuses {
		result = append(result, s)
	}
	return result
}

// GetServiceStatus returns the status of a single service.
func (hc *HealthChecker) GetServiceStatus(id string) *model.ServiceStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.statuses[id]
}
