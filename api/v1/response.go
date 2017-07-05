package v1

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

func sendServerError(w http.ResponseWriter, err error) {
	log.WithError(err).Warn("Server Error")
	sendError(w, err, http.StatusInternalServerError)
}

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
		log.WithError(err).Warn("Error encoding HTTP response data")
		http.Error(w, "Error encoding response data", http.StatusInternalServerError)
		return
	}
}
