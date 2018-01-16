package v2

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
)

func newContextError(err error) *contextError {
	return &contextError{
		fields: log.Fields{},
		err:    err,
	}
}

type contextError struct {
	fields log.Fields
	err    error
}

func (ce *contextError) WithField(k string, v interface{}) *contextError {
	ce.fields[k] = v
	return ce
}

func (ce *contextError) Error() string {
	return ce.err.Error()
}

func sendServerError(r *http.Request, w http.ResponseWriter, err error) {
	entry := log.WithError(err).WithField("route", mux.CurrentRoute(r).GetName())

	if ce, ok := err.(*contextError); ok {
		entry = entry.WithFields(ce.fields)
	}

	entry.Warn("Server Error")
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
