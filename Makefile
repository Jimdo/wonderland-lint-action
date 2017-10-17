PROJECT_NAME=wonderland-crons

TEST_TAGS=integration

BRANCH ?= master

CRONS_IMAGE = $(PROJECT_NAME)
CRONS_TEST_IMAGE = $(CRONS_IMAGE).test
AUTH_PROXY_IMAGE = auth-proxy

JIMDO_ENVIRONMENT=stage
ZONE=jimdo-platform-stage.net

WONDERLAND_REGISTRY=registry.jimdo-platform-stage.net

NOMAD_API=https://nomad.jimdo-platform-stage.net
NOMAD_WL_DOCKER_IMAGE=quay.io/jimdo/wl
NOMAD_AWS_REGION=$(AWS_REGION)
NOMAD_USER=$(WONDERLAND_USER)
NOMAD_PASS=$(WONDERLAND_PASS)

AUTH_USER=$(WONDERLAND_USER)
AUTH_PASS=$(WONDERLAND_PASS)

prod: JIMDO_ENVIRONMENT=prod
prod: ZONE=jimdo-platform.net
prod: WONDERLAND_REGISTRY=registry.jimdo-platform.net
prod: NOMAD_API=https://nomad.jimdo-platform.net
prod: deploy

stage: deploy

set-credentials:
	WONDERLAND_ENV=$(JIMDO_ENVIRONMENT) \
		wl vault write secret/wonderland/crons \
			NOMAD_USER=$(NOMAD_USER) \
			NOMAD_PASS=$(NOMAD_PASS) \
			WONDERLAND_GITHUB_TOKEN=$(CRONS_GITHUB_TOKEN) \
			WONDERLAND_REGISTRY_USER=$(WONDERLAND_USER) \
			WONDERLAND_REGISTRY_PASS=$(WONDERLAND_PASS) \
			QUAY_REGISTRY_USER=$(QUAY_USER) \
			QUAY_REGISTRY_PASS=$(QUAY_PASS) \
			LOGZIO_ACCOUNT_ID=$(LOGZIO_ACCOUNT_ID)
	WONDERLAND_ENV=$(JIMDO_ENVIRONMENT) \
		wl vault write secret/wonderland/crons/proxy \
			HTTP_USER="$(AUTH_USER)" \
			HTTP_PASSWORD="$(AUTH_PASS)"

deploy: set-credentials dinah
	AUTH_PROXY_IMAGE=$(shell dinah docker image $(AUTH_PROXY_IMAGE)) \
	CRONS_IMAGE=$(shell dinah docker image --branch $(BRANCH) $(CRONS_IMAGE)) \
	WONDERLAND_REGISTRY=$(WONDERLAND_REGISTRY) \
	NOMAD_API=$(NOMAD_API) \
	NOMAD_WL_DOCKER_IMAGE=$(NOMAD_WL_DOCKER_IMAGE) \
	NOMAD_AWS_REGION=$(NOMAD_AWS_REGION) \
	WONDERLAND_ENV=$(JIMDO_ENVIRONMENT) \
	ZONE=$(ZONE) \
		wl deploy $(PROJECT_NAME) -f wonderland.yaml

unit-test: TEST_TAGS=""
unit-test: test

test-container:
	docker build \
		-t $(CRONS_TEST_IMAGE) \
		-f Dockerfile.test \
		.

test: test-container
	docker run --rm -i \
		-v $(PWD):/go/src/github.com/Jimdo/$(PROJECT_NAME) \
		-e TEST_TAGS=$(TEST_TAGS) \
		$(CRONS_TEST_IMAGE) ./script/test

lint: test-container
	docker run --rm -i \
		-v $(PWD):/go/src/github.com/Jimdo/$(PROJECT_NAME) \
		-e TESt_TAGS=$(TEST_TAGS) \
		$(CRONS_TEST_IMAGE) ./script/lint

ci: lint test

container:
	docker build -t $(CRONS_IMAGE) .

push: container dinah
	# Push Docker images
	@dinah docker push --user $(QUAY_USER_PROD) --pass $(QUAY_PASS_PROD) --branch $(BRANCH) $(CRONS_IMAGE)

notify-jenkins: dinah
	# Notify Jenkins
	-@dinah jenkins build --stage --user $(JENKINS_USER_STAGE) --pass $(JENKINS_PASS_STAGE) --parameter BRANCH=$(BRANCH) Crons-Deploy
	@dinah jenkins build --user $(JENKINS_USER_PROD) --pass $(JENKINS_PASS_PROD) --parameter BRANCH=$(BRANCH) Crons-Deploy

dinah:
	# Install dinah
	@curl -sSL https://gist.github.com/white--rabbit/bca70b3215991e9e45905a1195388d09/raw | bash

gen-mocks:
	mockgen -package mock github.com/Jimdo/wonderland-crons/aws CronValidator > mock/cron_validator.go
	mockgen -package mock github.com/Jimdo/wonderland-crons/aws RuleCronManager > mock/rule_cron_manager.go
	mockgen -package mock github.com/Jimdo/wonderland-crons/aws TaskDefinitionStore > mock/task_definition_store.go
	mockgen -package mock github.com/Jimdo/wonderland-crons/aws CronStore > mock/cron_store.go
	mockgen -package mock github.com/Jimdo/wonderland-crons/aws CronExecutionStore > mock/cron_execution_store.go
	mockgen -package mock github.com/Jimdo/wonderland-crons/aws VaultSecretProvider > mock/vault_secret_provider.go
	mockgen -package mock github.com/Jimdo/wonderland-crons/events TaskStore > mock/task_store.go
	mockgen -package mock github.com/Jimdo/wonderland-crons/events CronStateToggler > mock/cron_state_toggler.go
	mockgen -package mock github.com/aws/aws-sdk-go/service/sqs/sqsiface SQSAPI > mock/sqsapi.go

