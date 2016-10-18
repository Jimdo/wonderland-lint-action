package main

import (
	"fmt"
	"net/http/pprof"
	"os"
	"time"

	graceful "gopkg.in/tylerb/graceful.v1"

	"github.com/Luzifer/rconfig"
	"github.com/gorilla/mux"
	"github.com/hashicorp/nomad/api"

	"github.com/Jimdo/wonderland-validator/docker/registry"
	wonderlandValidator "github.com/Jimdo/wonderland-validator/validator"

	"github.com/Jimdo/wonderland-crons/api/v1"
	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/service"
	"github.com/Jimdo/wonderland-crons/validation"
	"github.com/Jimdo/wonderland-crons/vault"
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

		// Vault
		VaultAccessToken string `flag:"vault-token" env:"VAULT_TOKEN" default:"" description:"Token to read variables and IAM credentials from Vault with"`

		// Docker registries
		WonderlandRegistryAddress string `flag:"wonderland-registry-address" env:"WONDERLAND_REGISTRY_ADDRESS" description:"The address of the Wonderland registry"`
		WonderlandRegistryUser    string `flag:"wonderland-registry-user" env:"WONDERLAND_REGISTRY_USER" description:"The username for the Wonderland registry"`
		WonderlandRegistryPass    string `flag:"wonderland-registry-pass" env:"WONDERLAND_REGISTRY_PASS" description:"The password for the Wonderland registry"`
		QuayRegistryAddress       string `flag:"query-registry-address" env:"QUAY_REGISTRY_ADDRESS" default:"quay.io" description:"The address of the Quay registry"`
		QuayRegistryUser          string `flag:"query-registry-user" env:"QUAY_REGISTRY_USER" description:"The username for the Quay registry"`
		QuayRegistryPass          string `flag:"query-registry-pass" env:"QUAY_REGISTRY_PASS" description:"The passwordfor the Quay registry"`
	}
)

func main() {
	if err := rconfig.Parse(&config); err != nil {
		abort("Could not parse config: %s", err)
	}

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
			Validator: validation.New(validation.Configuration{
				WonderlandNameValidator: &wonderlandValidator.WonderlandName{},
				DockerImageValidator: &wonderlandValidator.DockerImage{
					DockerImageService: registry.NewImageService([]registry.Credential{{
						Host:     config.QuayRegistryAddress,
						Username: config.QuayRegistryUser,
						Password: config.QuayRegistryPass,
					}, {
						Host:     config.WonderlandRegistryAddress,
						Username: config.WonderlandRegistryUser,
						Password: config.WonderlandRegistryPass,
					}}),
				},
				CapacityValidator: &wonderlandValidator.ContainerCapacity{
					CPUCapacitySpecifications: cron.CPUCapacitySpecifications,
					CPUMinCapacity:            cron.MinCPUCapacity,
					CPUMaxCapacity:            cron.MaxCPUCapacity,

					MemoryCapacitySpecifications: cron.MemoryCapacitySpecifications,
					MemoryMinCapacity:            cron.MinMemoryCapacity,
					MemoryMaxCapacity:            cron.MaxMemoryCapacity,
				},
				EnvironmentVariables: &wonderlandValidator.EnvironmentVariables{
					VaultSecretProvider: &vault.SecretProvider{
						VaultAccessToken: config.VaultAccessToken,
					},
				},
			}),
		},
		Router: router.PathPrefix("/v1").Subrouter(),
	}).Register()

	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	router.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)

	graceful.Run(config.Addr, config.ShutdownTimeout, router)
}

func abort(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}
