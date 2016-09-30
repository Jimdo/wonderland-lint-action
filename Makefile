PROJECT_NAME=wonderland-cron

BRANCH ?= master
CRON_IMAGE  = $(PROJECT_NAME)
AUTH_PROXY_IMAGE=auth-proxy

JIMDO_ENVIRONMENT=stage
ZONE=jimdo-platform-stage.net

prod: JIMDO_ENVIRONMENT=prod
prod: ZONE=jimdo-platform.net
prod: deploy

stage: deploy

set-credentials:
	WONDERLAND_ENV=$(JIMDO_ENVIRONMENT) \
		wl vault write secret/wonderland/cron \
			AWS_ACCESS_KEY_ID="$(AWS_ACCESS_KEY_ID)" \
			AWS_SECRET_ACCESS_KEY="$(AWS_SECRET_ACCESS_KEY)"
	WONDERLAND_ENV=$(JIMDO_ENVIRONMENT) \
		wl vault write secret/wonderland/cron/proxy \
			HTTP_USER="$(HTTP_USER)" \
			HTTP_PASSWORD="$(HTTP_PASSWORD)"

deploy: set-credentials dinah
	AUTH_PROXY_IMAGE=$(shell WONDERLAND_ENV=$(JIMDO_ENVIRONMENT) dinah docker image $(AUTH_PROXY_IMAGE)) \
	CRON_IMAGE=$(shell WONDERLAND_ENV=$(JIMDO_ENVIRONMENT) dinah docker image --branch $(BRANCH) $(CRON_IMAGE)) \
	WONDERLAND_ENV=$(JIMDO_ENVIRONMENT) \
	ZONE=$(ZONE) \
		wl deploy $(PROJECT_NAME) -f wonderland.yaml

test: container
	docker run -i --rm --entrypoint ./test.sh $(CRON_IMAGE)

container:
	docker build -t $(CRON_IMAGE) .

push: container dinah
	# Push Docker images
	@dinah docker push --stage --user $(QUAY_USER_STAGE) --pass $(QUAY_PASS_STAGE) --branch $(BRANCH) $(CRON_IMAGE)
	@dinah docker push --user $(QUAY_USER_PROD) --pass $(QUAY_PASS_PROD) --branch $(BRANCH) $(CRON_IMAGE)

notify-jenkins: dinah
	# Notify Jenkins
	-@dinah jenkins build --stage --user $(JENKINS_USER_STAGE) --pass $(JENKINS_PASS_STAGE) --parameter BRANCH=$(BRANCH) Cron-Deploy
	@dinah jenkins build --user $(JENKINS_USER_PROD) --pass $(JENKINS_PASS_PROD) --parameter BRANCH=$(BRANCH) Cron-Deploy

dinah:
	# Install dinah
	@curl -sSL https://gist.github.com/white--rabbit/bca70b3215991e9e45905a1195388d09/raw | bash
