package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Manager handles WebSocket connections for real-time updates
type Manager struct {
	connections map[string]map[*websocket.Conn]bool // jobID -> connections
	mu          sync.RWMutex
	upgrader    websocket.Upgrader
}

// ProgressUpdate represents a real-time progress update
type ProgressUpdate struct {
	JobID       string    `json:"jobId"`
	Status      string    `json:"status"`
	Progress    int       `json:"progress"`
	CurrentStep string    `json:"currentStep"`
	ETA         string    `json:"eta,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Message     string    `json:"message,omitempty"`
}

// NewManager creates a new WebSocket manager
func NewManager() *Manager {
	return &Manager{
		connections: make(map[string]map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins in development
				// In production, implement proper origin checking
				return true
			},
		},
	}
}

// HandleConnection handles new WebSocket connections for job progress
func (m *Manager) HandleConnection(w http.ResponseWriter, r *http.Request, jobID string) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Register connection
	m.addConnection(jobID, conn)
	defer m.removeConnection(jobID, conn)

	log.Printf("WebSocket connection established for job %s", jobID)

	// Send initial status
	m.sendWelcomeMessage(conn, jobID)

	// Keep connection alive and handle client messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for job %s: %v", jobID, err)
			}
			break
		}
		// Echo heartbeat or handle client messages if needed
	}
}

// BroadcastProgress sends progress update to all connections for a job
func (m *Manager) BroadcastProgress(update ProgressUpdate) {
	m.mu.RLock()
	connections, exists := m.connections[update.JobID]
	m.mu.RUnlock()

	if !exists || len(connections) == 0 {
		return
	}

	update.Timestamp = time.Now()
	message, err := json.Marshal(update)
	if err != nil {
		log.Printf("Failed to marshal progress update: %v", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for conn := range connections {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Failed to send progress update: %v", err)
			// Connection will be cleaned up by the connection handler
		}
	}

	log.Printf("Broadcasted progress update for job %s to %d connections", update.JobID, len(connections))
}

// GetConnectionCount returns the number of active connections for a job
func (m *Manager) GetConnectionCount(jobID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if connections, exists := m.connections[jobID]; exists {
		return len(connections)
	}
	return 0
}

// GetTotalConnections returns total active connections across all jobs
func (m *Manager) GetTotalConnections() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := 0
	for _, connections := range m.connections {
		total += len(connections)
	}
	return total
}

func (m *Manager) addConnection(jobID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connections[jobID] == nil {
		m.connections[jobID] = make(map[*websocket.Conn]bool)
	}
	m.connections[jobID][conn] = true
}

func (m *Manager) removeConnection(jobID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if connections, exists := m.connections[jobID]; exists {
		delete(connections, conn)
		if len(connections) == 0 {
			delete(m.connections, jobID)
		}
	}
}

func (m *Manager) sendWelcomeMessage(conn *websocket.Conn, jobID string) {
	welcome := ProgressUpdate{
		JobID:     jobID,
		Status:    "CONNECTED",
		Progress:  0,
		Message:   "WebSocket connection established",
		Timestamp: time.Now(),
	}

	message, _ := json.Marshal(welcome)
	conn.WriteMessage(websocket.TextMessage, message)
}
