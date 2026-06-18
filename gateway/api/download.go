package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// AgentDistDir is the directory containing pre-built agent binaries.
// Set at init from env or defaults to ../dist relative to the Gateway.
var AgentDistDir = ""

func init() {
	if d := os.Getenv("AGENT_DIST_DIR"); d != "" {
		AgentDistDir = d
	} else {
		// Default: look for ../dist relative to the working directory
		AgentDistDir = "dist"
	}
}

// platformInfo maps os/arch to filename in the dist directory.
var platformFiles = map[string]string{
	"linux/amd64":   "agent-linux-amd64",
	"linux/arm64":   "agent-linux-arm64",
	"darwin/amd64":  "agent-darwin-amd64",
	"darwin/arm64":  "agent-darwin-arm64",
	"windows/amd64": "agent-windows-amd64.exe",
}

var platformNames = map[string]string{
	"linux/amd64":   "Linux x86_64",
	"linux/arm64":   "Linux ARM64 (树莓派等)",
	"darwin/amd64":  "macOS Intel",
	"darwin/arm64":  "macOS Apple Silicon (M1/M2/M3/M4)",
	"windows/amd64": "Windows x86_64",
}

// AgentDownloadHandler handles agent binary downloads.
func AgentDownloadHandler(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/agent/download/{os}/{arch}
	path := strings.TrimPrefix(r.URL.Path, "/api/agent/download/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		http.Error(w, `{"error":"usage: /api/agent/download/{os}/{arch}"}`, http.StatusBadRequest)
		return
	}
	osName, arch := parts[0], parts[1]
	key := osName + "/" + arch

	filename, ok := platformFiles[key]
	if !ok {
		http.Error(w, fmt.Sprintf(`{"error":"unsupported platform: %s/%s"}`, osName, arch), http.StatusNotFound)
		return
	}

	filePath := filepath.Join(AgentDistDir, filename)
	info, err := os.Stat(filePath)
	if err != nil {
		log.Printf("[download] file not found: %s (%v)", filePath, err)
		http.Error(w, fmt.Sprintf(`{"error":"binary not built for %s/%s"}`, osName, arch), http.StatusNotFound)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="sop-agent-%s-%s"`, osName, arch))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	w.Header().Set("Cache-Control", "no-cache")

	http.ServeFile(w, r, filePath)
}

// AgentPlatformsHandler returns the list of available platforms as JSON.
func AgentPlatformsHandler(w http.ResponseWriter, r *http.Request) {
	platforms := make([]map[string]string, 0)
	for key, filename := range platformFiles {
		filePath := filepath.Join(AgentDistDir, filename)
		if _, err := os.Stat(filePath); err == nil {
			parts := strings.SplitN(key, "/", 2)
			platforms = append(platforms, map[string]string{
				"os":   parts[0],
				"arch": parts[1],
				"name": platformNames[key],
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"platforms": platforms})
}
