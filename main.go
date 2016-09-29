package main

import (
	"fmt"
	"os"
	"time"

	graceful "gopkg.in/tylerb/graceful.v1"

	"github.com/Luzifer/rconfig"
	"github.com/gorilla/mux"
	"github.com/hashicorp/nomad/api"

	"github.com/Jimdo/wonderland-cron/api/v1"
	"github.com/Jimdo/wonderland-cron/cron"
)

var (
	config struct {
		// Server
		addr            string        `flag:"addr" default:":8000" description:"The address/port combination to listen on"`
		shutdownTimeout time.Duration `flag:"shutdown-timeout" default:"10s" description:"The time to wait for active requests to finish when shutting down"`

		// Nomad
		nomadURI           string `flag:"nomad-api" env:"NOMAD_API" default:"" description:"The address of nomad API"`
		nomadUser          string `flag:"nomad-user" env:"NOMAD_USER" default:"" description:"The username to use for the nomad API"`
		nomadPass          string `flag:"nomad-pass" env:"NOMAD_PASS" default:"" description:"The password to use for the nomad API"`
		nomadCronPrefix    string `flag:"nomad-cron-prefix" env:"NOMAD_CRON_PREFIX" default:"wlc-" description:"The prefix to use for cron jobs in nomad"`
		nomadWLDockerImage string `flag:"nomad-wl-docker-image" env:"NOMAD_WL_DOCKER_IMAGE" default:"" description:"The Docker image to use for running wl commands in Nomad"`
	}
)

func init() {
	rconfig.Parse(&config)
}

func main() {
	router := mux.NewRouter()

	nomadClient, _ := api.NewClient(&api.Config{
		Address: config.nomadURI,
		HttpAuth: &api.HttpBasicAuth{
			Username: config.nomadUser,
			Password: config.nomadPass,
		},
	})

	v1.New(&v1.Config{
		CronStore: cron.NewNomadCronStore(&cron.NomadCronStoreConfig{
			CronPrefix:    config.nomadCronPrefix,
			Datacenters:   []string{os.Getenv("AWS_REGION")},
			Client:        nomadClient,
			WLDockerImage: config.nomadWLDockerImage,
			WLEnvironment: os.Getenv("WONDERLAND_ENV"),
			WLUser:        config.nomadUser,
			WLPass:        config.nomadPass,
		}),
		Router: router.PathPrefix("/v1").Subrouter(),
	}).Register()

	graceful.Run(config.addr, config.shutdownTimeout, router)
}

func abort(err error) {
	fmt.Fprintf(os.Stderr, "%s", err)
	os.Exit(1)
}
