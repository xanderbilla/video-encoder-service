package types

import "time"

type APIResponse struct {
	Success   bool        `json:"success"`
	Status    int         `json:"status"`
	Timestamp string      `json:"timestamp"`
	RequestID string      `json:"requestId"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
}

type ErrorInfo struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp string                 `json:"timestamp"`
	Stage     string                 `json:"stage,omitempty"`
	LogsPath  string                 `json:"logsPath,omitempty"`
}

func NewSuccessResponse(statusCode int, requestID string, data interface{}) *APIResponse {
	return &APIResponse{
		Success:   true,
		Status:    statusCode,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: requestID,
		Data:      data,
	}
}

func NewSuccessResponseWithMessage(statusCode int, requestID, message string, data interface{}) *APIResponse {
	return &APIResponse{
		Success:   true,
		Status:    statusCode,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: requestID,
		Message:   message,
		Data:      data,
	}
}

func NewErrorResponse(statusCode int, requestID, code, message string, details map[string]interface{}) *APIResponse {
	return &APIResponse{
		Success:   false,
		Status:    statusCode,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: requestID,
		Error: &ErrorInfo{
			Code:      code,
			Message:   message,
			Details:   details,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
}
