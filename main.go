package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"time"

	graceful "gopkg.in/tylerb/graceful.v1"
	"github.com/Luzifer/rconfig"
	"github.com/gorilla/mux"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/nomad/api"
	vaultapi "github.com/hashicorp/vault/api"

	"github.com/Jimdo/wonderland-validator/docker/registry"
	wonderlandValidator "github.com/Jimdo/wonderland-validator/validator"
	"github.com/Jimdo/wonderland-vault/lib/role-credential-manager"

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
		VaultAddress string `flag:"vault-address" env:"VAULT_ADDR" default:"http://127.0.0.1:8282" description:"Vault Address"`
		VaultRoleID  string `flag:"vault-role-id" env:"VAULT_ROLE_ID" description:"RoleID with sufficient access"`

		// wl authentication
		WonderlandGitHubToken string `flag:"wonderland-github-token" env:"WONDERLAND_GITHUB_TOKEN" default:"" description:"The GitHub token to use for wl authentication"`

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

	vaultURL, err := url.Parse(config.VaultAddress)
	if err != nil {
		abort("Could not parse Vault address: %s", err)
	}

	rcm, err := rcm.NewWithDebug(config.VaultAddress, config.VaultRoleID)
	if err != nil {
		abort("Could not set up role credential manager: %s", err)
	}

	router := mux.NewRouter()

	clientConfig := &api.Config{
		Address:    config.NomadURI,
		HttpClient: cleanhttp.DefaultPooledClient(),
		HttpAuth: &api.HttpBasicAuth{
			Username: config.NomadUser,
			Password: config.NomadPass,
		},
		TLSConfig: &api.TLSConfig{},
	}

	// This is required for the client to work with a custom HttpClient
	transport := clientConfig.HttpClient.Transport.(*http.Transport)
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	nomadClient, err := api.NewClient(clientConfig)
	if err != nil {
		abort("Could not create Nomad client: %s", err)
	}

	v1.New(&v1.Config{
		Service: &service.CronService{
			Store: cron.NewNomadCronStore(&cron.NomadCronStoreConfig{
				CronPrefix:    config.NomadCronPrefix,
				Datacenters:   []string{os.Getenv("AWS_REGION")},
				Client:        nomadClient,
				WLDockerImage: config.NomadWLDockerImage,
				WLEnvironment: os.Getenv("WONDERLAND_ENV"),
				WLGitHubToken: config.WonderlandGitHubToken,
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
						VaultClients: map[string]*vaultapi.Client{
							vaultURL.Host: rcm.VaultClient,
						},
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
