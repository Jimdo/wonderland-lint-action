package v2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/Jimdo/wonderland-crons/api"
	"github.com/Jimdo/wonderland-crons/aws"
	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/store"
	"github.com/Jimdo/wonderland-crons/validation"
)

func New(c *Config) *API {
	return &API{
		config: c,
	}
}

type Config struct {
	Router  *mux.Router
	Service *aws.Service
	URI     *URIGenerator
}

type API struct {
	config *Config
}

func (a *API) Register() {
	a.config.Router.HandleFunc("/status", a.StatusHandler).Methods("GET")

	a.config.Router.HandleFunc("/aws/sns/execution-trigger", a.ExecutionTriggerHandler).Methods("POST")

	a.config.Router.HandleFunc("/crons/ping", api.HandlerWithDefaultTimeout(a.PingHandler)).Methods("GET")
	a.config.Router.HandleFunc("/crons", api.HandlerWithDefaultTimeout(a.ListCrons)).Methods("GET")
	a.config.Router.HandleFunc("/crons/{name}", api.HandlerWithDefaultTimeout(a.DeleteHandler)).Methods("DELETE")
	a.config.Router.HandleFunc("/crons/{name}", api.HandlerWithDefaultTimeout(a.PutHandler)).Methods("PUT")
	a.config.Router.HandleFunc("/crons/{name}", api.HandlerWithDefaultTimeout(a.CronStatus)).Methods("GET")
	a.config.Router.HandleFunc("/crons/{name}/logs", api.HandlerWithDefaultTimeout(a.CronLogs)).Methods(http.MethodGet)
}

func (a *API) StatusHandler(w http.ResponseWriter, req *http.Request) {}

func (a *API) ExecutionTriggerHandler(w http.ResponseWriter, req *http.Request) {
	msgType := req.Header.Get("x-amz-sns-message-type")
	switch msgType {
	case "SubscriptionConfirmation":
		var opt struct {
			SubscribeURL string
		}
		if err := json.NewDecoder(req.Body).Decode(&opt); err != nil {
			sendServerError(w, err)
			return
		}
		resp, err := http.Get(opt.SubscribeURL)
		if err != nil {
			sendServerError(w, err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			sendServerError(w, errors.New(http.StatusText(resp.StatusCode)))
			return
		}
	case "Notification":
		var notification struct {
			Type    string
			Message string
		}
		if err := json.NewDecoder(req.Body).Decode(&notification); err != nil {
			sendServerError(w, err)
			return
		}

		var cwEvent struct {
			DetailType string   `json:"detail-type"`
			Resources  []string `json:"resources"`
		}
		if err := json.Unmarshal([]byte(notification.Message), &cwEvent); err != nil {
			sendServerError(w, err)
			return
		}

		if cwEvent.DetailType != "Scheduled Event" {
			sendServerError(w, fmt.Errorf("Unhandled event type %q", cwEvent.DetailType))
			return
		}
		if len(cwEvent.Resources) != 1 {
			sendServerError(w, fmt.Errorf("Event contains not exactly one resource, but %d", len(cwEvent.Resources)))
			return
		}

		ruleARN := cwEvent.Resources[0]
		if err := a.config.Service.TriggerExecution(ruleARN); err != nil {
			sendServerError(w, err)
			return
		}
	default:
		sendServerError(w, fmt.Errorf("Unsupported message type %q", msgType))
	}
}

func (a *API) PingHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	sendJSON(w, "pong", http.StatusOK)
}

func (a *API) PutHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cronName := vars["name"]

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

	if err := a.config.Service.Apply(cronName, desc); err != nil {
		statusCode := http.StatusInternalServerError
		if _, ok := err.(validation.Error); ok {
			statusCode = http.StatusBadRequest
		}
		sendError(w, fmt.Errorf("Unable to run cron: %s", err), statusCode)
		return
	}
}

func (a *API) DeleteHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cronName := vars["name"]

	if err := a.config.Service.Delete(cronName); err != nil {
		sendServerError(w, fmt.Errorf("Unable to delete cron %q: %s", cronName, err))
		return
	}
}

func (a *API) ListCrons(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	crons, err := a.config.Service.List()
	if err != nil {
		sendServerError(w, fmt.Errorf("Unable to list crons: %s", err))
		return
	}

	sendJSON(w, crons, http.StatusOK)
}

func (a *API) CronStatus(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cronName := vars["name"]

	params := req.URL.Query()
	configuredExecutions := params.Get("executions")

	var executions int64
	var err error

	if configuredExecutions == "" {
		executions = 10
	} else {
		executions, err = strconv.ParseInt(configuredExecutions, 10, 64)
		if err != nil {
			sendServerError(w, fmt.Errorf("Could not convert executions into int64: %s", err))
			return
		}
	}

	status, err := a.config.Service.Status(cronName, executions)
	if err != nil {
		if err == store.ErrCronNotFound {
			sendError(w, fmt.Errorf("Cron not found"), http.StatusNotFound)
		} else {
			sendServerError(w, fmt.Errorf("Unable to get status of cron %s: %s", cronName, err))
		}
		return
	}

	sendJSON(w, status, http.StatusOK)
}

func (a *API) CronLogs(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cronName := vars["name"]

	exists, err := a.config.Service.Exists(cronName)
	if err != nil {
		sendServerError(w, fmt.Errorf("Unable to check if cron %s exists: %s", cronName, err))
		return
	} else if !exists {
		sendError(w, fmt.Errorf("Cron not found"), http.StatusNotFound)
		return
	}

	cronLogsInformation := CronLogsInformation{
		HTMLLink: a.config.URI.CronLogsHTML(cronName),
	}

	sendJSON(w, cronLogsInformation, http.StatusOK)
}
