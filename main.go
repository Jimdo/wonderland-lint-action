package main

import (
	"fmt"
	"net/http"
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
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	graceful "gopkg.in/tylerb/graceful.v1"

	"github.com/Jimdo/wonderland-ecs-metadata/ecsmetadata"
	"github.com/Jimdo/wonderland-validator/docker/registry"
	wonderlandValidator "github.com/Jimdo/wonderland-validator/validator"
	"github.com/Jimdo/wonderland-vault/lib/role-credential-manager"

	"os/signal"
	"syscall"

	"github.com/Jimdo/wonderland-crons/api"
	"github.com/Jimdo/wonderland-crons/api/v2"
	"github.com/Jimdo/wonderland-crons/aws"
	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/cronitor"
	"github.com/Jimdo/wonderland-crons/events"
	"github.com/Jimdo/wonderland-crons/locking"
	"github.com/Jimdo/wonderland-crons/metrics"
	"github.com/Jimdo/wonderland-crons/notifications"
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

		// Vault
		VaultAddress string `flag:"vault-address" env:"VAULT_ADDR" default:"http://127.0.0.1:8282" description:"Vault Address"`
		VaultRoleID  string `flag:"vault-role-id" env:"VAULT_ROLE_ID" description:"RoleID with sufficient access"`

		// Docker registries
		WonderlandRegistryAddress string `flag:"wonderland-registry-address" env:"WONDERLAND_REGISTRY_ADDRESS" description:"The address of the Wonderland registry"`
		WonderlandRegistryUser    string `flag:"wonderland-registry-user" env:"WONDERLAND_REGISTRY_USER" description:"The username for the Wonderland registry"`
		WonderlandRegistryPass    string `flag:"wonderland-registry-pass" env:"WONDERLAND_REGISTRY_PASS" description:"The password for the Wonderland registry"`
		QuayRegistryAddress       string `flag:"query-registry-address" env:"QUAY_REGISTRY_ADDRESS" default:"quay.io" description:"The address of the Quay registry"`
		QuayRegistryUser          string `flag:"query-registry-user" env:"QUAY_REGISTRY_USER" description:"The username for the Quay registry"`
		QuayRegistryPass          string `flag:"query-registry-pass" env:"QUAY_REGISTRY_PASS" description:"The passwordfor the Quay registry"`

		// AWS
		CronRoleARN                     string        `flag:"cron-role-arn" env:"CRON_ROLE_ARN" description:"The IAM Role that grants Cloudwatch access to ECS"`
		ECSClusterARN                   string        `flag:"cluster-arn" env:"ECS_CLUSTER_ARN" description:"The ARN of the ECS cluster crons should run on"`
		ECSEventsQueueURL               string        `flag:"ecs-events-queue-url" env:"ECS_EVENTS_QUEUE_URL" description:"The URL of the SQS queue receiving ECS events"`
		ECSEventQueuePollInterval       time.Duration `flag:"ecs-events-queue-poll-interval" default:"100ms" description:"The interval in which to poll new ECS events"`
		CronsTableName                  string        `flag:"crons-table-name" env:"CRONS_TABLE_NAME" description:"Name of the DynamoDB Table used for storing crons"`
		ExecutionsTableName             string        `flag:"executions-table-name" env:"EXECUTIONS_TABLE_NAME" description:"Name of the DynamoDB Table used for storing cron executions"`
		WorkerLeaderLockRefreshInterval time.Duration `flag:"worker-leader-lock-refresh-interval" default:"1m" description:"The interval in which to refresh the workers leader lock"`
		WorkerLeaderLockTableName       string        `flag:"worker-leader-lock-table-name" env:"WORKER_LEADER_LOCK_TABLE_NAME" description:"Name of the DynamoDB Table used for worker leadership locking"`
		ExecutionTriggerTopicARN        string        `flag:"exec-trigger-topic-arn" env:"EXEC_TRIGGER_TOPIC_ARN" description:"ARN of the SNS topic that triggers cron executions"`
		ECSNoScheduleMarkerAttribute    string        `flag:"no-schedule-attribute" env:"NO_SCHEDULE_ATTRIBUTE" description:"The name of an ECS attribute that marks an ECS instance as 'not available' for scheduling"`

		// ECS Metadata
		ECSMetadataAPIAddress string `flag:"ecs-metadata-api" env:"ECS_METADATA_API" default:"" description:"The address of the ECS API"`
		ECSMetadataAPIUser    string `flag:"ecs-metadata-user" env:"ECS_METADATA_USER" default:"" description:"The username to use for the ECS API"`
		ECSMetadataAPIPass    string `flag:"ecs-metadata-pass" env:"ECS_METADATA_PASS" default:"" description:"The password to use for the ECS API"`

		// Logz.io
		LogzioURL       string `flag:"logzio-url" env:"LOGZIO_URL" default:"https://app-eu.logz.io" description:"The URL of the Logz.io endpoint to use for Kibana and other services"`
		LogzioAccountID string `flag:"logzio-account-id" env:"LOGZIO_ACCOUNT_ID" description:"The Logz.io account ID to use for Kibana URLs"`

		// Cronitor
		CronitorApiKey                 string `flag:"cronitor-api-key" env:"CRONITOR_API_KEY" description:"Cronitor API Key"`
		CronitorAuthKey                string `flag:"cronitor-auth-key" env:"CRONITOR_AUTH_KEY" description:"Cronitor Auth Key"`
		CronitorWlNotificationsAPIUser string `flag:"cronitor-wl-notifications-user" env:"CRONITOR_WL_NOTIFICATIONS_USER" default:"" description:"The username that cronitor should use for the notifications API"`
		CronitorWlNotificationsAPIPass string `flag:"cronitor-wl-notifications-pass" env:"CRONITOR_WL_NOTIFICATIONS_PASS" default:"" description:"The pawssword that cronitor should use for the notifications API"`

		// Timeout
		TimeoutImage string `flag:"timeout-image" env:"TIMEOUT_IMAGE" descriptions "Docker image that should be used as timeout container"`

		// Notifications
		NotificationsAPIAddress string `flag:"notifications-api" env:"NOTIFICATIONS_API" description:"The address of the notifications API"`
		NotificationsAPIUser    string `flag:"notifications-user" env:"NOTIFICATIONS_API_USER" default:"" description:"The username to use for the notifications API"`
		NotificationsAPIPass    string `flag:"notifications-pass" env:"NOTIFICATIONS_API_PASS" default:"" description:"The password to use for the notifications API"`
		NotificationsAPITeam    string `flag:"notifications-team" default:"werkzeugschmiede" description:"The notifications team to use"`
	}
	programIdentifier = "wonderland-crons"
	programVersion    = "dev"

	awsSession *session.Session
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

	if config.TimeoutImage == "" {
		abort("Please pass a Timeout Image")
	}

	if config.NotificationsAPIAddress == "" {
		abort("Please pass notifications api address")
	}

	stop := make(chan interface{})
	defer close(stop)

	router := mux.NewRouter()

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

	dynamoDBExecutionStore, err := store.NewDynamoDBExecutionStore(dynamoDBClient, config.ExecutionsTableName)
	if err != nil {
		log.Fatalf("Failed to initialize Task store: %s", err)
	}

	ecstdm := aws.NewECSTaskDefinitionMapper(vaultSecretProvider, vaultAppRoleProvider)
	ecstds := aws.NewECSTaskDefinitionStore(
		ecsClient,
		ecstdm,
		ecsMetadata(),
		config.ECSClusterARN,
		fmt.Sprintf("%s/%s", programIdentifier, programVersion),
		config.ECSNoScheduleMarkerAttribute,
		config.TimeoutImage,
	)
	cloudwatchcm := aws.NewCloudwatchRuleCronManager(cwClient, config.ECSClusterARN, config.CronRoleARN)
	dynamoDBCronStore, err := store.NewDynamoDBCronStore(dynamoDBClient, config.CronsTableName)
	if err != nil {
		log.Fatalf("Failed to initialize Cron store: %s", err)
	}

	hc := &http.Client{Timeout: time.Duration(10) * time.Second}
	cronitorClient := cronitor.New(config.CronitorApiKey, config.CronitorAuthKey, hc)

	userAgent := fmt.Sprintf("%s/%s", programIdentifier, programVersion)
	notificationClient := notifications.NewClient(http.DefaultTransport, config.NotificationsAPIAddress, config.NotificationsAPIUser, config.NotificationsAPIPass, userAgent, config.NotificationsAPITeam)

	metricsUpdater := metrics.NewPrometheus()
	urlGenerator := notifications.NewURLGenerator(config.CronitorWlNotificationsAPIUser, config.CronitorWlNotificationsAPIPass, config.NotificationsAPIAddress)

	service := aws.NewService(validator, cloudwatchcm, ecstds, dynamoDBCronStore, dynamoDBExecutionStore, config.ExecutionTriggerTopicARN, cronitorClient, metricsUpdater, notificationClient, urlGenerator)

	eventDispatcher := events.NewEventDispatcher()
	eventDispatcher.On(events.EventCronExecutionStateChanged, events.CronExecutionStatePersister(dynamoDBExecutionStore))
	eventDispatcher.On(events.EventCronExecutionStateChanged, events.ExecutionReporter(
		dynamoDBExecutionStore,
		dynamoDBCronStore,
		cronitorClient,
		metricsUpdater,
	))

	lm := locking.NewDynamoDBLockManager(dynamoDBClient, config.WorkerLeaderLockTableName)
	w := events.NewWorker(lm, sqsClient, config.ECSEventsQueueURL, eventDispatcher, metricsUpdater,
		events.WithPollInterval(config.ECSEventQueuePollInterval),
		events.WithLockRefreshInterval(config.WorkerLeaderLockRefreshInterval))

	go func() {
		if err := w.Run(stop); err != nil {
			log.Fatalf("Error consuming ECS events: %s", err)
		}
	}()

	router.Handle("/metrics", prometheus.Handler())
	router.HandleFunc("/status", api.StatusHandler)

	v2.New(&v2.Config{
		Router:  router.PathPrefix("/v2").Subrouter(),
		Service: service,
		URI: &v2.URIGenerator{
			LogzioAccountID: config.LogzioAccountID,
			LogzioURL:       config.LogzioURL,
		},
	}).Register()

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
	c := ecs.New(getAWSSession())

	// prefix user-agent with program name and version
	c.Handlers.Build.PushFront(request.MakeAddToUserAgentHandler(programIdentifier, programVersion))

	return c
}

func cloudwatchEventsClient() *cloudwatchevents.CloudWatchEvents {
	c := cloudwatchevents.New(getAWSSession())

	// prefix user-agent with program name and version
	c.Handlers.Build.PushFront(request.MakeAddToUserAgentHandler(programIdentifier, programVersion))

	return c
}

func dynamoDBClient() *dynamodb.DynamoDB {
	c := dynamodb.New(getAWSSession())

	// prefix user-agent with program name and version
	c.Handlers.Build.PushFront(request.MakeAddToUserAgentHandler(programIdentifier, programVersion))

	return c
}

func sqsClient() *sqs.SQS {
	c := sqs.New(getAWSSession())

	// prefix user-agent with program name and version
	c.Handlers.Build.PushFront(request.MakeAddToUserAgentHandler(programIdentifier, programVersion))

	return c
}

func getAWSSession() *session.Session {
	if awsSession == nil {
		httpClient := &http.Client{
			Timeout: time.Duration(10) * time.Second,
		}
		awsSession = session.Must(session.NewSession(awssdk.NewConfig().WithHTTPClient(httpClient)))
	}
	return awsSession
}

func ecsMetadata() ecsmetadata.Metadata {
	return &ecsmetadata.Fallback{
		Primary: ecsmetadata.NewECSMetadataService(ecsmetadata.Config{
			BaseURI:   config.ECSMetadataAPIAddress,
			Username:  config.ECSMetadataAPIUser,
			Password:  config.ECSMetadataAPIPass,
			UserAgent: fmt.Sprintf("%s/%s", programIdentifier, programVersion),
		}),
		Secondary: &ecsmetadata.ECSAPI{
			ECS: ecsClient(),
		},
	}
}
