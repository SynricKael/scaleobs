package collector

import (
	"os/exec"
	"runtime"
	"strings"
)

// KnownAgentNames lists process names that identify coding agent servers.
// These are checked by DetectAgents() on each collect cycle.
var KnownAgentNames = []string{
	"opencode",
	"claude",
	"codex",
	"sop-agent", // ourselves (avoid reporting self)
}

// DetectAgents scans the local machine for running coding agent processes.
// Returns a list of agent names found (e.g., ["opencode", "claude code"]).
// This is best-effort — relies on process name matching.
func DetectAgents() []string {
	found := make(map[string]bool)

	switch runtime.GOOS {
	case "windows":
		found = detectWindows(found)
	case "linux":
		found = detectLinux(found)
	case "darwin":
		found = detectDarwin(found)
	}

	result := make([]string, 0, len(found))
	for name := range found {
		// Skip self
		if name == "sop-agent" {
			continue
		}
		// Normalize names
		switch name {
		case "claude":
			result = append(result, "claude code")
		case "opencode":
			result = append(result, "opencode")
		case "codex":
			result = append(result, "codex")
		default:
			result = append(result, name)
		}
	}

	return result
}

func detectWindows(found map[string]bool) map[string]bool {
	for _, name := range KnownAgentNames {
		cmd := exec.Command("tasklist", "/NH", "/FI", "IMAGENAME eq "+name+".exe")
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		// Case-insensitive check (tasklist outputs "OpenCode.exe" but we search for "opencode.exe")
		if strings.Contains(strings.ToLower(string(out)), strings.ToLower(name+".exe")) {
			found[name] = true
		}
	}
	return found
}

func detectLinux(found map[string]bool) map[string]bool {
	for _, name := range KnownAgentNames {
		cmd := exec.Command("pgrep", "-x", name)
		if err := cmd.Run(); err == nil {
			found[name] = true
		}
	}
	return found
}

func detectDarwin(found map[string]bool) map[string]bool {
	for _, name := range KnownAgentNames {
		cmd := exec.Command("pgrep", "-x", name)
		if err := cmd.Run(); err == nil {
			found[name] = true
		}
	}
	return found
}
