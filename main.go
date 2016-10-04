package main

import (
	"fmt"
	"os"
	"time"

	graceful "gopkg.in/tylerb/graceful.v1"

	"github.com/Luzifer/rconfig"
	"github.com/gorilla/mux"
	"github.com/hashicorp/nomad/api"

	"github.com/Jimdo/wonderland-crons/api/v1"
	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/service"
	"github.com/Jimdo/wonderland-crons/validation"
)

var (
	config struct {
		// Server
		Addr            string        `flag:"addr" default:":8000" description:"The address/port combination to listen on"`
		ShutdownTimeout time.Duration `flag:"shutdown-timeout" default:"10s" description:"The time to wait for active requests to finish when shutting down"`

		// Nomad
		NomadURI           string `flag:"nomad-api" env:"NOMAD_API" default:"" description:"The address of nomad API"`
		NomadUser          string `flag:"nomad-user" env:"NOMAD_USER" default:"" description:"The username to use for the nomad API"`
		NomadPass          string `flag:"nomad-pass" env:"NOMAD_PASS" default:"" description:"The password to use for the nomad API"`
		NomadCronPrefix    string `flag:"nomad-cron-prefix" env:"NOMAD_CRON_PREFIX" default:"wlc-" description:"The prefix to use for cron jobs in nomad"`
		NomadWLDockerImage string `flag:"nomad-wl-docker-image" env:"NOMAD_WL_DOCKER_IMAGE" default:"" description:"The Docker image to use for running wl commands in Nomad"`
	}
)

func init() {
	rconfig.Parse(&config)
}

func main() {
	router := mux.NewRouter()

	nomadClient, _ := api.NewClient(&api.Config{
		Address: config.NomadURI,
		HttpAuth: &api.HttpBasicAuth{
			Username: config.NomadUser,
			Password: config.NomadPass,
		},
	})

	v1.New(&v1.Config{
		Service: &service.CronService{
			Store: cron.NewNomadCronStore(&cron.NomadCronStoreConfig{
				CronPrefix:    config.NomadCronPrefix,
				Datacenters:   []string{os.Getenv("AWS_REGION")},
				Client:        nomadClient,
				WLDockerImage: config.NomadWLDockerImage,
				WLEnvironment: os.Getenv("WONDERLAND_ENV"),
				WLUser:        config.NomadUser,
				WLPass:        config.NomadPass,
			}),
			Validator: validation.New(),
		},
		Router: router.PathPrefix("/v1").Subrouter(),
	}).Register()

	graceful.Run(config.Addr, config.ShutdownTimeout, router)
}

func abort(err error) {
	fmt.Fprintf(os.Stderr, "%s", err)
	os.Exit(1)
}
