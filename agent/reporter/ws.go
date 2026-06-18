package reporter

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/glrs/observer/agent/model"
)

// WSReporter manages the WebSocket connection to the Gateway.
type WSReporter struct {
	GatewayURL string
	ServerID   string
	Token      string
	conn       *websocket.Conn
	done       chan struct{}
}

// NewWSReporter creates a WSReporter.
func NewWSReporter(gatewayURL, serverID, token string) *WSReporter {
	return &WSReporter{
		GatewayURL: gatewayURL,
		ServerID:   serverID,
		Token:      token,
		done:       make(chan struct{}),
	}
}

// Connect establishes the WebSocket connection and authenticates.
func (r *WSReporter) Connect() error {
	u, err := url.Parse(r.GatewayURL)
	if err != nil {
		return fmt.Errorf("parse gateway URL: %w", err)
	}

	// Ensure path ends with /api/ws/agent
	u.Path = "/api/ws/agent"
	if u.Scheme == "http" {
		u.Scheme = "ws"
	} else if u.Scheme == "https" {
		u.Scheme = "wss"
	}

	log.Printf("[agent] connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}

	// Send auth message
	authMsg := model.AgentMessage{
		Type:     "auth",
		ServerID: r.ServerID,
		Token:    r.Token,
	}
	if err := conn.WriteJSON(authMsg); err != nil {
		conn.Close()
		return fmt.Errorf("send auth: %w", err)
	}

	// Read auth response
	var resp model.AgentMessage
	if err := conn.ReadJSON(&resp); err != nil {
		conn.Close()
		return fmt.Errorf("read auth response: %w", err)
	}

	if resp.Type == "auth_error" {
		conn.Close()
		return fmt.Errorf("authentication failed: invalid token")
	}

	if resp.Type != "auth_ok" {
		conn.Close()
		return fmt.Errorf("unexpected auth response: %s", resp.Type)
	}

	r.conn = conn
	log.Printf("[agent] authenticated as %s", r.ServerID)
	return nil
}

// SendMetrics sends a metrics snapshot to the Gateway.
func (r *WSReporter) SendMetrics(metrics *model.Metrics) error {
	if r.conn == nil {
		return fmt.Errorf("not connected")
	}

	msg := model.AgentMessage{
		Type:      "metrics",
		ServerID:  r.ServerID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      metrics,
	}

	data, _ := json.Marshal(msg)
	if err := r.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}

	return r.conn.WriteMessage(websocket.TextMessage, data)
}

// Close closes the WebSocket connection.
func (r *WSReporter) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// RunWithReconnect runs the reporting loop with automatic reconnection.
// It collects and sends metrics at the specified interval.
func (r *WSReporter) RunWithReconnect(interval time.Duration, collectFn func() (*model.Metrics, error), stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial connect
	if err := r.Connect(); err != nil {
		log.Printf("[agent] initial connect failed: %v, will retry", err)
	}

	// Start a goroutine to read pings/responses from the server
	go func() {
		for {
			if r.conn == nil {
				time.Sleep(time.Second)
				continue
			}
			if _, _, err := r.conn.ReadMessage(); err != nil {
				log.Printf("[agent] connection read error: %v", err)
				r.conn = nil
				return
			}
		}
	}()

	for {
		select {
		case <-stop:
			log.Println("[agent] stopping reporter")
			r.Close()
			return

		case <-ticker.C:
			// Ensure connected
			if r.conn == nil {
				if err := r.Connect(); err != nil {
					log.Printf("[agent] reconnect failed: %v", err)
					continue
				}
				// Restart reader goroutine
				go func() {
					for {
						if r.conn == nil {
							time.Sleep(time.Second)
							continue
						}
						if _, _, err := r.conn.ReadMessage(); err != nil {
							r.conn = nil
							return
						}
					}
				}()
			}

			// Collect metrics
			metrics, err := collectFn()
			if err != nil {
				log.Printf("[agent] collect metrics failed: %v", err)
				continue
			}

			// Send
			if err := r.SendMetrics(metrics); err != nil {
				log.Printf("[agent] send metrics failed: %v", err)
				r.conn = nil
			}
		}
	}
}
