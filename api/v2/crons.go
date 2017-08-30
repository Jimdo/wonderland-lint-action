package v2

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Jimdo/wonderland-crons/api"
	"github.com/Jimdo/wonderland-crons/service"
)

func New(c *Config) *API {
	return &API{
		config: c,
	}
}

type Config struct {
	Service *service.CronService
	Router  *mux.Router
}

type API struct {
	config *Config
}

func (a *API) Register() {
	a.config.Router.HandleFunc("/status", a.StatusHandler).Methods("GET")

	a.config.Router.HandleFunc("/crons/ping", api.HandlerWithDefaultTimeout(a.PingHandler)).Methods("GET")
}

func (a *API) StatusHandler(w http.ResponseWriter, req *http.Request) {}

func (a *API) PingHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	sendJSON(w, "pong", http.StatusOK)
}
