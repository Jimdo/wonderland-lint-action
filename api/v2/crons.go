package v2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/Jimdo/wonderland-crons/api"
	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/store"
	"github.com/Jimdo/wonderland-crons/validation"
)

func New(c *Config) *API {
	return &API{
		config: c,
		hc:     &http.Client{Timeout: time.Second * 30},
	}
}

type CronService interface {
	Apply(cronName string, desc *cron.Description) error
	Delete(cronName string) error
	Exists(cronName string) (bool, error)
	List() ([]string, error)
	Status(cronName string, executionCount int64) (*cron.Status, error)
	TriggerExecutionByRuleARN(ruleARN string) error
	TriggerExecutionByCronName(cronName string) error
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Config struct {
	Router  *mux.Router
	Service CronService
	URI     *URIGenerator
}

type API struct {
	config *Config
	hc     HTTPClient
}

func (a *API) Register() {
	a.config.Router.HandleFunc("/status", a.StatusHandler).Methods("GET").Name("v2_status")

	a.config.Router.HandleFunc("/aws/sns/execution-trigger", a.ExecutionTriggerHandler).Methods("POST").Name("v2_execution_trigger")

	a.config.Router.HandleFunc("/crons", api.HandlerWithDefaultTimeout(a.ListCrons)).Methods("GET").Name("v2_list_crons")
	a.config.Router.HandleFunc("/crons/{name}", api.HandlerWithDefaultTimeout(a.DeleteHandler)).Methods("DELETE").Name("v2_delete_cron")
	a.config.Router.HandleFunc("/crons/{name}", api.HandlerWithDefaultTimeout(a.PutHandler)).Methods("PUT").Name("v2_put_cron")
	a.config.Router.HandleFunc("/crons/{name}/executions", api.HandlerWithDefaultTimeout(a.CronExecutionHandler)).Methods("POST").Name("v2_post_cron_executions")
	a.config.Router.HandleFunc("/crons/{name}", api.HandlerWithDefaultTimeout(a.CronStatus)).Methods("GET").Name("v2_cron_status")
	a.config.Router.HandleFunc("/crons/{name}/logs", api.HandlerWithDefaultTimeout(a.CronLogs)).Methods(http.MethodGet).Name("v2_cron_logs")
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
			sendServerError(req, w, newContextError(err).WithField("msg_type", msgType))
			return
		}

		r, _ := http.NewRequest(http.MethodGet, opt.SubscribeURL, nil)
		resp, err := a.hc.Do(r)
		if err != nil {
			sendServerError(req, w, newContextError(err).WithField("msg_type", msgType))
			return
		}
		if resp.StatusCode != http.StatusOK {
			sendServerError(req, w, newContextError(errors.New("Could not subscribe to SNS topic")).
				WithField("msg_type", msgType).
				WithField("aws_response", resp.StatusCode))
			return
		}
	case "Notification":
		var notification struct {
			Type    string
			Message string
		}
		if err := json.NewDecoder(req.Body).Decode(&notification); err != nil {
			sendServerError(req, w, newContextError(err).WithField("msg_type", msgType))
			return
		}

		var cwEvent struct {
			DetailType string   `json:"detail-type"`
			Resources  []string `json:"resources"`
		}
		if err := json.Unmarshal([]byte(notification.Message), &cwEvent); err != nil {
			sendServerError(req, w, newContextError(err).WithField("msg_type", msgType))
			return
		}

		if cwEvent.DetailType != "Scheduled Event" {
			sendServerError(req, w, newContextError(errors.New("Unhandled event type")).WithField("msg_type", msgType).WithField("event_type", cwEvent.DetailType))
			return
		}
		if len(cwEvent.Resources) != 1 {
			sendServerError(req, w, newContextError(fmt.Errorf("Event contains not exactly one resource, but %d", len(cwEvent.Resources))).WithField("msg_type", msgType))
			return
		}

		ruleARN := cwEvent.Resources[0]
		if err := a.config.Service.TriggerExecutionByRuleARN(ruleARN); err != nil {
			sendServerError(req, w, newContextError(err).WithField("ruleARN", ruleARN).WithField("msg_type", msgType))
			return
		}

		w.WriteHeader(http.StatusCreated)
	default:
		sendServerError(req, w, newContextError(errors.New("Unsupported message type")).WithField("msg_type", msgType))
	}
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
		sendServerError(req, w, newContextError(errors.New("Unable to delete cron")).WithField("cron", cronName))
		return
	}
}

func (a *API) CronExecutionHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cronName := vars["name"]

	if err := a.config.Service.TriggerExecutionByCronName(cronName); err != nil {
		statusCode := http.StatusInternalServerError
		if err == store.ErrCronNotFound {
			statusCode = http.StatusNotFound
		}
		sendError(w, fmt.Errorf("Unable to run cron: %s", err), statusCode)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (a *API) ListCrons(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	crons, err := a.config.Service.List()
	if err != nil {
		sendServerError(req, w, fmt.Errorf("Unable to list crons: %s", err))
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
			sendServerError(req, w, fmt.Errorf("Could not convert executions into int64: %s", err))
			return
		}
	}

	status, err := a.config.Service.Status(cronName, executions)
	if err != nil {
		if err == store.ErrCronNotFound {
			sendError(w, fmt.Errorf("Cron not found"), http.StatusNotFound)
		} else {
			sendServerError(req, w, newContextError(fmt.Errorf("Unable to get status of cron: %s", err)).WithField("cron", cronName))
		}
		return
	}

	sendJSON(w, MapToCronAPICronStatus(status), http.StatusOK)
}

func (a *API) CronLogs(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	cronName := vars["name"]

	exists, err := a.config.Service.Exists(cronName)
	if err != nil {
		sendServerError(req, w, newContextError(fmt.Errorf("Unable to check if cron exists: %s", err)).WithField("cron", cronName))
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
