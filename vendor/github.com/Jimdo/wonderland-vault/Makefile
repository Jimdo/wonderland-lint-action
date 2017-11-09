prod: VERSION=0.6.1
prod: SHA256SUM=4f248214e4e71da68a166de60cc0c1485b194f4a2197da641187b745c8d5b8be
prod: YAML_VAULT_VERSION=v1.1.1
prod: YAML_VAULT_CHECKSUM=b76625ac1cfaa96d80a92b6712ec8ecd4d50bd6581ac42517b3fc787c8fbf9ac
prod: ZONE=jimdo-platform.net
prod: ENV=prod
prod: REGION=eu-west-1
prod: clean auto_init write_basic_configuration write_vault_acls write_aws_roles

stage: VERSION=0.6.1
stage: SHA256SUM=4f248214e4e71da68a166de60cc0c1485b194f4a2197da641187b745c8d5b8be
stage: YAML_VAULT_VERSION=v1.1.1
stage: YAML_VAULT_CHECKSUM=b76625ac1cfaa96d80a92b6712ec8ecd4d50bd6581ac42517b3fc787c8fbf9ac
stage: ZONE=jimdo-platform-stage.net
stage: ENV=stage
stage: REGION=eu-west-1
stage: clean auto_init write_basic_configuration write_vault_acls write_aws_roles

TAG=v$(VERSION)
IMAGE=wonderland-vault

# +++ BUILDING +++

clean:
	rm -rf vault

# +++ ASSETS +++

vault:
	curl -sSLo vault.zip https://releases.hashicorp.com/vault/$(VERSION)/vault_$(VERSION)_linux_amd64.zip
	echo "$(SHA256SUM)  vault.zip" | shasum -a 256 -c
	unzip vault.zip
	chmod 0755 vault

yaml-vault:
	curl -sSLo yaml-vault https://github.com/Jimdo/yaml-vault/releases/download/$(YAML_VAULT_VERSION)/yaml-vault_linux_amd64
	echo "$(YAML_VAULT_CHECKSUM)  yaml-vault" | shasum -a 256 -c
	chmod 0755 yaml-vault

dinah:
	# Install dinah
	@curl -sSL https://gist.github.com/white--rabbit/bca70b3215991e9e45905a1195388d09/raw | bash

wait_vault_ready: export VAULT_ADDR=https://vault.$(ZONE)/
wait_vault_ready: vault
	./scripts/wait_vault.sh

# +++ ROLES +++

write_vault_acls: export VAULT_ADDR=https://vault.$(ZONE)/
write_vault_acls: wait_vault_ready yaml-vault
	./yaml-vault --verbose --import --file=vault_acls.yaml

write_aws_roles: export VAULT_ADDR=https://vault.$(ZONE)/
write_aws_roles: wait_vault_ready yaml-vault
	./yaml-vault --verbose --import --file=vault_aws.yaml
	./yaml-vault --verbose --import --file=vault_aws_infra.yaml
	./yaml-vault --verbose --import --file=vault_aws_dolphin.yaml

# +++ Further configuration options +++

write_basic_configuration: export VAULT_ADDR=https://vault.$(ZONE)/
write_basic_configuration: wait_vault_ready yaml-vault
	./yaml-vault --verbose --import --ignore-errors --file=vault_config.yaml

# ++++ Restore initial "empty" database dump ++++
restore_initial:
	./dynamodump/dynamodump.py -m restore -r $(REGION) -s wonderland-vault

backup_initial:
	./dynamodump/dynamodump.py -m backup -r $(REGION) -s wonderland-vault

auto_init: export AWS_DEFAULT_REGION=$(REGION)
auto_init:
	./scripts/manage.sh

.PHONY: vault vault.hcl yaml-vault
