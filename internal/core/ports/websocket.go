package ports

// WebSocketManager defines the interface for WebSocket management
type WebSocketManager interface {
	BroadcastProgressFromPorts(jobID, status string, progress int, currentStep, eta, message string)
	GetConnectionCount(jobID string) int
}
