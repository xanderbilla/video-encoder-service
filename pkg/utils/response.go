package utils

import (
	"encoding/json"
	"encoder-service/pkg/types"
	"net/http"
)

func SendJSON(w http.ResponseWriter, statusCode int, response *types.APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func SendSuccess(w http.ResponseWriter, statusCode int, requestID string, data interface{}) {
	response := types.NewSuccessResponse(statusCode, requestID, data)
	SendJSON(w, statusCode, response)
}

func SendSuccessWithMessage(w http.ResponseWriter, statusCode int, requestID, message string, data interface{}) {
	response := types.NewSuccessResponseWithMessage(statusCode, requestID, message, data)
	SendJSON(w, statusCode, response)
}

func SendError(w http.ResponseWriter, statusCode int, requestID, code, message string, details map[string]interface{}) {
	response := types.NewErrorResponse(statusCode, requestID, code, message, details)
	SendJSON(w, statusCode, response)
}
