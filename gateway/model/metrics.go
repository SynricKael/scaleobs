package model

// AgentMessage is a message exchanged between Agent and Gateway via WebSocket.
type AgentMessage struct {
	Type      string         `json:"type"`      // "auth" | "metrics" | "exec" | "exec_result"
	ServerID  string         `json:"server_id,omitempty"`
	Token     string         `json:"token,omitempty"`
	Timestamp string         `json:"timestamp,omitempty"`
	Data      *ServerMetrics `json:"data,omitempty"`

	// For exec commands
	Command  string `json:"command,omitempty"`
	Timeout  int    `json:"timeout,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
}
