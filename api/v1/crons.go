package v1

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Jimdo/wonderland-cron/api"
	"github.com/Jimdo/wonderland-cron/cron"
)

func New(c *Config) *API {
	return &API{
		config: c,
	}
}

type Config struct {
	CronStore cron.CronStore
	Router    *mux.Router
}

type API struct {
	config *Config
}

func (a *API) Register() {
	a.config.Router.HandleFunc("/status", a.StatusHandler).Methods("GET")

	a.config.Router.HandleFunc("/crons", api.HandlerWithDefaultTimeout(a.ListCrons)).Methods("GET")
	a.config.Router.HandleFunc("/crons", api.HandlerWithDefaultTimeout(a.RunCron)).Methods("POST")
	a.config.Router.HandleFunc("/crons/{name}", api.HandlerWithDefaultTimeout(a.CronStatus)).Methods("GET")
	a.config.Router.HandleFunc("/crons/{name}", api.HandlerWithDefaultTimeout(a.StopCron)).Methods("DELETE")
	a.config.Router.HandleFunc("/crons/{name}/allocations", api.HandlerWithDefaultTimeout(a.CronAllocations)).Methods("GET")
	a.config.Router.HandleFunc("/crons/allocations/{id}", api.HandlerWithDefaultTimeout(a.AllocationStatus)).Methods("GET")
	a.config.Router.HandleFunc("/crons/allocations/{id}/logs", api.HandlerWithDefaultTimeout(a.AllocationLogs)).Methods("GET")
}

func (a *API) StatusHandler(w http.ResponseWriter, req *http.Request) {}

func (a *API) ListCrons(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	crons, err := a.config.CronStore.List()
	if err != nil {
		sendError(w, fmt.Errorf("Unable to list crons: %s", err), http.StatusInternalServerError)
		return
	}

	sendJSON(w, crons, http.StatusOK)
}

func (a *API) CronStatus(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cronName := vars["name"]

	status, err := a.config.CronStore.Status(cronName)
	if err != nil {
		if err == cron.ErrCronNotFound {
			sendError(w, err, http.StatusNotFound)
			return
		}

		sendError(w, fmt.Errorf("Unable to get status of cron %q: %s", cronName, err), http.StatusInternalServerError)
		return
	}

	sendJSON(w, status, http.StatusOK)
}

func (a *API) StopCron(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cronName := vars["name"]

	if err := a.config.CronStore.Stop(cronName); err != nil {
		if err == cron.ErrCronNotFound {
			sendError(w, err, http.StatusNotFound)
			return
		}

		sendError(w, fmt.Errorf("Unable to delete cron %q: %s", cronName, err), http.StatusInternalServerError)
		return
	}
}

func (a *API) RunCron(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		sendError(w, fmt.Errorf("Unable to read request body: %s", err), http.StatusInternalServerError)
		return
	}
	desc, err := cron.NewCronDescriptionFromJSON(body)
	if err != nil {
		sendError(w, fmt.Errorf("Unable to parse cron description: %s", err), http.StatusBadRequest)
		return
	}

	if err := a.config.CronStore.Run(desc); err != nil {
		sendError(w, fmt.Errorf("Unable to run cron: %s", err), http.StatusInternalServerError)
		return
	}
}

func (a *API) CronAllocations(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cronName := vars["name"]

	allocs, err := a.config.CronStore.Allocations(cronName)
	if err != nil {
		if err == cron.ErrCronNotFound {
			sendError(w, err, http.StatusNotFound)
			return
		}

		sendError(w, fmt.Errorf("Unable to list cron allocations: %s", err), http.StatusInternalServerError)
		return
	}

	sendJSON(w, allocs, http.StatusOK)
}

func (a *API) AllocationStatus(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	allocID := vars["id"]

	status, err := a.config.CronStore.AllocationStatus(allocID)
	if err != nil {
		if err == cron.ErrInvalidAllocationID {
			sendError(w, err, http.StatusBadRequest)
			return
		} else if err == cron.ErrAllocationNotFound {
			sendError(w, err, http.StatusNotFound)
			return
		}

		sendError(w, fmt.Errorf("Unable to get status of allocation %q: %s", allocID, err), http.StatusInternalServerError)
		return
	}

	sendJSON(w, status, http.StatusOK)
}

func (a *API) AllocationLogs(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	allocID := vars["id"]

	logType := "stdout"
	if req.URL.Query().Get("log-type") != "" {
		logType = req.URL.Query().Get("log-type")
	}

	status, err := a.config.CronStore.AllocationLogs(allocID, logType)
	if err != nil {
		if err == cron.ErrAllocationNotFound {
			sendError(w, err, http.StatusNotFound)
			return
		}

		sendError(w, fmt.Errorf("Unable to get logs of allocation %q: %s", allocID, err), http.StatusInternalServerError)
		return
	}

	sendJSON(w, status, http.StatusOK)
}
