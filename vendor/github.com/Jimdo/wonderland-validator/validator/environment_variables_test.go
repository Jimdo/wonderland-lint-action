package validator

import (
	"testing"

	"github.com/Jimdo/wonderland-validator/vault"
)

func TestValidate_EnvironmentVariablesValid(t *testing.T) {
	env := map[string]string{
		"FOO":          "bar",
		"foo_123":      "barxyz",
		"$ref_secrets": "vault+secret://some.url.com",
	}

	v := &EnvironmentVariables{}
	if err := v.Validate(env); err != nil {
		t.Errorf("%+v should be valid environment variables", env)
	}
}

func TestValidate_EnvironmentVariablesInvalid(t *testing.T) {
	envs := []map[string]string{{
		"": "bar",
	}, {
		"?123":         "FOO",
		"$ref_secrets": "vault+secret://",
	}}

	v := &EnvironmentVariables{}
	for _, env := range envs {
		if err := v.Validate(env); err == nil {
			t.Errorf("%+v should not be valid environment variables", env)
		}
	}
}

func TestValidateEnvironmentVariables_ValidVaultSecretPath(t *testing.T) {
	envs := map[string]string{
		"$ref_secrets": "vault+secret://my-vault.com/secrets",
	}

	v := &EnvironmentVariables{
		VaultSecretProvider: &vault.StubSecretProvider{},
	}

	if err := v.Validate(envs); err != nil {
		t.Errorf("Vault reference should be valid: %s", err)
	}
}

func TestValidateEnvironmentVariables_InvalidVaultURL(t *testing.T) {
	envs := map[string]string{
		"$ref_secrets": "vault+secret://unknown-vault.com/secrets",
	}

	v := &EnvironmentVariables{
		VaultSecretProvider: &vault.StubSecretProvider{ResolveError: true},
	}

	if err := v.Validate(envs); err == nil {
		t.Errorf("Vault reference should not be valid: %s", err)
	}
}
