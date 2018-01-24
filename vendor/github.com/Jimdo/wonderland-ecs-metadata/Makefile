PROJECT_NAME=wonderland-ecs-metadata

SERVICE_IMAGE=$(PROJECT_NAME)
TEST_IMAGE=$(PROJECT_NAME).test
AUTH_PROXY_IMAGE    = auth-proxy

JIMDO_ENVIRONMENT = stage
ZONE              = jimdo-platform-stage.net

stage: deploy

prod: JIMDO_ENVIRONMENT=prod
prod: ZONE=jimdo-platform.net
prod: deploy

container:
	docker build -t $(SERVICE_IMAGE) .

test-container:
	docker build \
		-t $(TEST_IMAGE) \
		-f Dockerfile.test \
		.

test: test-container
	docker run --rm -i \
		-v $(PWD):/go/src/github.com/Jimdo/$(PROJECT_NAME) \
		-e TEST_TAGS=$(TEST_TAGS) \
		$(TEST_IMAGE) ./script/test

lint:
	docker run --rm -i \
		-v $(PWD):/go/src/github.com/Jimdo/$(PROJECT_NAME) \
		-w /go/src/github.com/Jimdo/$(PROJECT_NAME) \
		quay.io/jimdo/gometalinter-container:latest

push: container dinah
	# Push Docker images
	@dinah docker push --user $(QUAY_USER_PROD) --pass $(QUAY_PASS_PROD) --branch $(BRANCH) $(SERVICE_IMAGE)

notify-jenkins: dinah
	# Notify Jenkins
	-@dinah jenkins build --stage --user $(JENKINS_USER_STAGE) --pass $(JENKINS_PASS_STAGE) --parameter BRANCH=$(BRANCH) ECS-Metadata-Deploy
	@dinah jenkins build --user $(JENKINS_USER_PROD) --pass $(JENKINS_PASS_PROD) --parameter BRANCH=$(BRANCH) ECS-Metadata-Deploy

set-credentials:
	WONDERLAND_ENV=$(JIMDO_ENVIRONMENT) \
		wl vault write secret/wonderland/ecs-metadata/proxy \
			HTTP_USER="$(WONDERLAND_USER)" \
			HTTP_PASSWORD="$(WONDERLAND_PASS)"

deploy: dinah set-credentials
	AUTH_PROXY_IMAGE=$(shell WONDERLAND_ENV=$(JIMDO_ENVIRONMENT) dinah docker image $(AUTH_PROXY_IMAGE)) \
	IMAGE=$(shell WONDERLAND_ENV=$(JIMDO_ENVIRONMENT) dinah docker image --branch $(BRANCH) $(SERVICE_IMAGE)) \
	REDIS_ADDRESS="$(REDIS_ADDRESS)" \
	REDIS_PORT="$(REDIS_PORT)" \
	WONDERLAND_ENV=$(JIMDO_ENVIRONMENT) \
	ZONE=$(ZONE) \
		wl deploy $(PROJECT_NAME)-api -f wonderland/api.yaml
	AUTH_PROXY_IMAGE=$(shell WONDERLAND_ENV=$(JIMDO_ENVIRONMENT) dinah docker image $(AUTH_PROXY_IMAGE)) \
	ECS_EVENTS_QUEUE_URL="$(ECS_EVENTS_QUEUE_URL)" \
	IMAGE=$(shell WONDERLAND_ENV=$(JIMDO_ENVIRONMENT) dinah docker image --branch $(BRANCH) $(SERVICE_IMAGE)) \
	REDIS_ADDRESS="$(REDIS_ADDRESS)" \
	REDIS_PORT="$(REDIS_PORT)" \
	WONDERLAND_ENV=$(JIMDO_ENVIRONMENT) \
	ZONE=$(ZONE) \
		wl deploy $(PROJECT_NAME)-worker -f wonderland/worker.yaml

dinah:
	# Install dinah
	@curl -sSL https://gist.github.com/white--rabbit/bca70b3215991e9e45905a1195388d09/raw | bash

gen-mocks:
	mockgen -package=mock github.com/aws/aws-sdk-go/service/ecs/ecsiface ECSAPI > mock/ecsapi.go
	mockgen -package=mock github.com/aws/aws-sdk-go/service/sqs/sqsiface SQSAPI > mock/sqsapi.go
	mockgen -package=mock github.com/Jimdo/wonderland-ecs-metadata/ecsmetadata Metadata > mock/ecsmetadata.go
