package proxy

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/glrs/observer/gateway/model"
)

// ServiceProxy handles reverse proxying for a single service.
// This is a simple pass-through proxy: path rewriting + Basic Auth +
// TLS skip-verify. No HTML/JS/CSS rewriting.
// For panel viewing, the Tauri frontend opens panel_url in the system
// browser directly — the proxy is only needed for API/health aggregation.
type ServiceProxy struct {
	Service    *model.Service
	proxy      *httputil.ReverseProxy
	basePath   string
	targetPath string
}

// New creates a ServiceProxy for the given service.
func New(svc *model.Service) (*ServiceProxy, error) {
	target, err := url.Parse(svc.Target)
	if err != nil {
		return nil, fmt.Errorf("parse target %q: %w", svc.Target, err)
	}

	basePath := svc.URL
	if !strings.HasSuffix(basePath, "/") {
		basePath += "/"
	}

	tp := strings.TrimRight(target.Path, "/")

	sp := &ServiceProxy{
		Service:    svc,
		basePath:   strings.TrimSuffix(basePath, "/"),
		targetPath: tp,
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	proxy := &httputil.ReverseProxy{
		Director:     sp.director(target),
		ErrorHandler: sp.errorHandler,
		Transport:    transport,
	}
	sp.proxy = proxy

	return sp, nil
}

// ServeHTTP implements http.Handler.
func (sp *ServiceProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sp.proxy.ServeHTTP(w, r)
}

// director creates the request director function.
func (sp *ServiceProxy) director(target *url.URL) func(*http.Request) {
	return func(r *http.Request) {
		originalPath := r.URL.Path
		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host

		// Rewrite path: /p/frps/some/path -> /some/path
		relativePath := strings.TrimPrefix(originalPath, sp.basePath)
		if relativePath == "" || !strings.HasPrefix(relativePath, "/") {
			relativePath = "/"
		}
		// Prepend target's own base path (e.g. /web/) if it has one
		if target.Path != "" && target.Path != "/" {
			tp := strings.TrimRight(target.Path, "/")
			if !strings.HasPrefix(relativePath, tp) {
				relativePath = tp + "/" + strings.TrimLeft(relativePath, "/")
			}
		}
		r.URL.Path = relativePath
		r.URL.RawPath = ""

		// Inject Basic Auth if configured
		if sp.Service.Auth != nil && sp.Service.Auth.Type == "basic" {
			authVal := sp.Service.Auth.Username + ":" + sp.Service.Auth.Password
			encoded := base64.StdEncoding.EncodeToString([]byte(authVal))
			r.Header.Set("Authorization", "Basic "+encoded)
		}

		r.Header.Set("X-Forwarded-For", r.RemoteAddr)
		r.Header.Set("X-Forwarded-Proto", "http")
		r.Host = target.Host
	}
}

// errorHandler handles proxy errors gracefully.
func (sp *ServiceProxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("[proxy] %s error: %v", sp.Service.ID, err)
	http.Error(w, fmt.Sprintf("Proxy error for %s: service unreachable", sp.Service.Name),
		http.StatusBadGateway)
}

// ProxyRouter manages all service proxies.
type ProxyRouter struct {
	proxies map[string]*ServiceProxy
}

// NewRouter creates a ProxyRouter.
func NewRouter() *ProxyRouter {
	return &ProxyRouter{
		proxies: make(map[string]*ServiceProxy),
	}
}

// Register adds a proxy for a service.
func (pr *ProxyRouter) Register(svc *model.Service) error {
	if _, exists := pr.proxies[svc.ID]; exists {
		return fmt.Errorf("proxy for service %s already registered", svc.ID)
	}
	sp, err := New(svc)
	if err != nil {
		return err
	}
	pr.proxies[svc.ID] = sp
	return nil
}

// Get returns the ServiceProxy for the given service ID.
func (pr *ProxyRouter) Get(id string) *ServiceProxy {
	return pr.proxies[id]
}

// ServeHTTP routes to the appropriate service proxy based on URL path.
func (pr *ProxyRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 3)
	if len(parts) < 2 || parts[0] != "p" {
		http.NotFound(w, r)
		return
	}
	serviceID := parts[1]

	proxy, ok := pr.proxies[serviceID]
	if !ok {
		http.Error(w, fmt.Sprintf("Service %q not found", serviceID), http.StatusNotFound)
		return
	}

	proxy.ServeHTTP(w, r)
}
