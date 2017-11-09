package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"time"

	"github.com/Luzifer/rconfig"
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/gorilla/mux"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/nomad/api"
	log "github.com/sirupsen/logrus"
	graceful "gopkg.in/tylerb/graceful.v1"

	"github.com/Jimdo/wonderland-validator/docker/registry"
	wonderlandValidator "github.com/Jimdo/wonderland-validator/validator"
	"github.com/Jimdo/wonderland-vault/lib/role-credential-manager"

	"os/signal"
	"syscall"

	"github.com/Jimdo/wonderland-crons/api/v1"
	"github.com/Jimdo/wonderland-crons/api/v2"
	"github.com/Jimdo/wonderland-crons/aws"
	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/events"
	"github.com/Jimdo/wonderland-crons/locking"
	"github.com/Jimdo/wonderland-crons/nomad"
	"github.com/Jimdo/wonderland-crons/store"
	"github.com/Jimdo/wonderland-crons/validation"
	"github.com/Jimdo/wonderland-crons/vault"
)

var (
	config struct {
		LogLevel string `flag:"log" env:"LOG_LEVEL" default:"info" description:"The minimum LogLevel of messages to write. Possible values: []"`

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

		// AWS
		AWSRegion                       string        `flag:"region" env:"AWS_REGION" default:"eu-west-1" description:"The AWS region to use"`
		CronRoleARN                     string        `flag:"cron-role-arn" env:"CRON_ROLE_ARN" description:"The IAM Role that grants Cloudwatch access to ECS"`
		ECSClusterARN                   string        `flag:"cluster-arn" env:"ECS_CLUSTER_ARN" description:"The ARN of the ECS cluster crons should run on"`
		RefreshAWSCredentialsInterval   time.Duration `flag:"aws-credentials-interval" env:"AWS_CREDENTIALS_INTERVAL" default:"10m" description:"Interval in which to fetch new AWS credentials from Vault"`
		ECSEventsQueueURL               string        `flag:"ecs-events-queue-url" env:"ECS_EVENTS_QUEUE_URL" description:"The URL of the SQS queue receiving ECS events"`
		ECSEventQueuePollInterval       time.Duration `flag:"ecs-events-queue-poll-interval" default:"100ms" description:"The interval in which to poll new ECS events"`
		CronsTableName                  string        `flag:"crons-table-name" env:"CRONS_TABLE_NAME" description:"Name of the DynamoDB Table used for storing crons"`
		ExecutionsTableName             string        `flag:"executions-table-name" env:"EXECUTIONS_TABLE_NAME" description:"Name of the DynamoDB Table used for storing cron executions"`
		WorkerLeaderLockRefreshInterval time.Duration `flag:"worker-leader-lock-refresh-interval" default:"1m" description:"The interval in which to refresh the workers leader lock"`
		WorkerLeaderLockTableName       string        `flag:"worker-leader-lock-table-name" env:"WORKER_LEADER_LOCK_TABLE_NAME" description:"Name of the DynamoDB Table used for worker leadership locking"`

		// Logz.io
		LogzioURL       string `flag:"logzio-url" env:"LOGZIO_URL" default:"https://app-eu.logz.io" description:"The URL of the Logz.io endpoint to use for Kibana and other services"`
		LogzioAccountID string `flag:"logzio-account-id" env:"LOGZIO_ACCOUNT_ID" description:"The Logz.io account ID to use for Kibana URLs"`
	}
	programIdentifier = "wonderland-crons"
	programVersion    = "dev"
)

func main() {
	if err := rconfig.Parse(&config); err != nil {
		abort("Could not parse config: %s", err)
	}

	level, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		abort("Invalid log level %q: %s", config.LogLevel, err)
	}
	log.SetLevel(level)
	log.SetFormatter(&log.JSONFormatter{})

	if config.CronsTableName == "" {
		abort("Please pass a crons DynamoDB table name")
	}

	if config.ExecutionsTableName == "" {
		abort("Please pass a executions DynamoDB table name")
	}

	if config.WorkerLeaderLockTableName == "" {
		abort("Please pass a lock DynamoDB table name")
	}

	stop := make(chan interface{})
	defer close(stop)

	router := mux.NewRouter()

	nomadURI, err := url.Parse(config.NomadURI)
	if err != nil {
		abort("Could not parse Nomad URI: %s", err)
	}

	clientConfig := &api.Config{
		Address:    config.NomadURI,
		HttpClient: cleanhttp.DefaultPooledClient(),
		HttpAuth: &api.HttpBasicAuth{
			Username: config.NomadUser,
			Password: config.NomadPass,
		},
		TLSConfig: &api.TLSConfig{
			TLSServerName: nomadURI.Hostname(),
		},
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

	ecsClient := ecsClient()
	cwClient := cloudwatchEventsClient()
	dynamoDBClient := dynamoDBClient()
	sqsClient := sqsClient()

	r, err := rcm.New(
		config.VaultAddress,
		config.VaultRoleID,
		rcm.WithAWSClientConfigs(&ecsClient.Config, &cwClient.Config, &dynamoDBClient.Config, &sqsClient.Config),
		rcm.WithAWSIAMRole(programIdentifier),
		rcm.WithIgnoreErrors(),
		rcm.WithLogger(log.StandardLogger()),
	)
	if err != nil {
		log.Fatalf("Error creating RCM library: %s", err)
	}
	if err := r.Init(); err != nil {
		log.Fatalf("Error initializing RCM library: %s", err)
	}

	vaultSecretProvider := &vault.SecretProvider{
		VaultClient: r.VaultClient,
	}
	vaultAppRoleProvider := &vault.AppRoleProvider{
		VaultClient: r.VaultClient,
	}

	validator := validation.New(validation.Configuration{
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
			VaultSecretProvider: vaultSecretProvider,
		},
	})

	go func() {
		if err := r.Run(stop); err != nil {
			log.Fatalf("RCM library returned error even though ignore errors option was set: %s", err)
		}
	}()

	v1.New(&v1.Config{
		Service: &nomad.CronService{
			Store: nomad.NewCronStore(&nomad.CronStoreConfig{
				CronPrefix:    config.NomadCronPrefix,
				Datacenters:   []string{os.Getenv("AWS_REGION")},
				Client:        nomadClient,
				WLDockerImage: config.NomadWLDockerImage,
				WLEnvironment: os.Getenv("WONDERLAND_ENV"),
				WLGitHubToken: config.WonderlandGitHubToken,
			}),
			Validator: validator,
		},
		Router: router.PathPrefix("/v1").Subrouter(),
	}).Register()

	dynamoDBExecutionStore, err := store.NewDynamoDBExecutionStore(dynamoDBClient, config.ExecutionsTableName)
	if err != nil {
		log.Fatalf("Failed to initialize Task store: %s", err)
	}

	ecstdm := aws.NewECSTaskDefinitionMapper(vaultSecretProvider, vaultAppRoleProvider)
	ecstds := aws.NewECSTaskDefinitionStore(ecsClient, ecstdm)
	cloudwatchcm := aws.NewCloudwatchRuleCronManager(cwClient, config.ECSClusterARN, config.CronRoleARN)
	dynamoDBCronStore, err := store.NewDynamoDBCronStore(dynamoDBClient, config.CronsTableName)
	if err != nil {
		log.Fatalf("Failed to initialize Cron store: %s", err)
	}

	service := aws.NewService(validator, cloudwatchcm, ecstds, dynamoDBCronStore, dynamoDBExecutionStore)

	eventDispatcher := events.NewEventDispatcher()
	eventDispatcher.On(events.EventCronExecutionStarted, events.CronDeactivator(service))
	eventDispatcher.On(events.EventCronExecutionStopped, events.CronActivator(service))
	eventDispatcher.On(events.EventCronExecutionStateChanged, events.CronExecutionStatePersister(dynamoDBExecutionStore))

	lm := locking.NewDynamoDBLockManager(dynamoDBClient, config.WorkerLeaderLockTableName)
	w := events.NewWorker(lm, sqsClient, config.ECSEventsQueueURL, eventDispatcher,
		events.WithPollInterval(config.ECSEventQueuePollInterval),
		events.WithLockRefreshInterval(config.WorkerLeaderLockRefreshInterval))

	go func() {
		if err := w.Run(stop); err != nil {
			log.Fatalf("Error consuming ECS events: %s", err)
		}
	}()

	v2.New(&v2.Config{
		Router:  router.PathPrefix("/v2").Subrouter(),
		Service: service,
		URI: &v2.URIGenerator{
			LogzioAccountID: config.LogzioAccountID,
			LogzioURL:       config.LogzioURL,
		},
	}).Register()

	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	router.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)

	signals := make(chan os.Signal, 1)
	go func() {
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		<-signals
		stop <- struct{}{}
	}()

	graceful.Run(config.Addr, config.ShutdownTimeout, router)
}

func abort(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}

func ecsClient() *ecs.ECS {
	httpClient := &http.Client{
		Timeout: time.Duration(10) * time.Second,
	}
	s := session.Must(session.NewSession(awssdk.NewConfig().WithHTTPClient(httpClient)))
	c := ecs.New(s)

	// prefix user-agent with program name and version
	c.Handlers.Build.PushFront(request.MakeAddToUserAgentHandler(programIdentifier, programVersion))

	return c
}

func cloudwatchEventsClient() *cloudwatchevents.CloudWatchEvents {
	httpClient := &http.Client{
		Timeout: time.Duration(10) * time.Second,
	}
	s := session.Must(session.NewSession(awssdk.NewConfig().WithHTTPClient(httpClient)))
	c := cloudwatchevents.New(s)

	// prefix user-agent with program name and version
	c.Handlers.Build.PushFront(request.MakeAddToUserAgentHandler(programIdentifier, programVersion))

	return c
}

func dynamoDBClient() *dynamodb.DynamoDB {
	httpClient := &http.Client{
		Timeout: time.Duration(10) * time.Second,
	}
	s := session.Must(session.NewSession(awssdk.NewConfig().WithHTTPClient(httpClient)))
	c := dynamodb.New(s)

	// prefix user-agent with program name and version
	c.Handlers.Build.PushFront(request.MakeAddToUserAgentHandler(programIdentifier, programVersion))

	return c
}

func sqsClient() *sqs.SQS {
	httpClient := &http.Client{
		Timeout: time.Duration(10) * time.Second,
	}
	s := session.Must(session.NewSession(awssdk.NewConfig().WithHTTPClient(httpClient)))
	c := sqs.New(s)

	// prefix user-agent with program name and version
	c.Handlers.Build.PushFront(request.MakeAddToUserAgentHandler(programIdentifier, programVersion))

	return c
}
