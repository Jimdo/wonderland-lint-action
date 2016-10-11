package v1

import (
	"encoding/json"
	"log"
	"net/http"
)

func sendError(w http.ResponseWriter, err error, code int) {
	apiErr := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}{
		Code:    code,
		Message: err.Error(),
	}

	sendJSON(w, apiErr, code)
}

func sendJSON(w http.ResponseWriter, data interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding HTTP response data: %s", err)
		http.Error(w, "Error encoding response data", http.StatusInternalServerError)
		return
	}
}
