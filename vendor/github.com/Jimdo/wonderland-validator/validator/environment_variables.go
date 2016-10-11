package validator

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	vaultReferenceKeyPrefix = "$ref"
	vaultReferenceURLScheme = "vault+secret"
)

var environmentVariableNameRegexp = regexp.MustCompile(`^(\$ref_)?[a-zA-Z_]+[a-zA-Z0-9_]*$`)

type EnvironmentVariables struct {
	VaultSecretProvider VaultSecretProvider
}

type VaultSecretProvider interface {
	GetValues(*url.URL) (map[string]string, error)
}

func (v *EnvironmentVariables) Validate(env map[string]string) error {
	for key, value := range env {
		if !environmentVariableNameRegexp.MatchString(key) {
			return fmt.Errorf("%q is not a valid name for an environment variable. It must match %q", key, environmentVariableNameRegexp.String())
		}

		if !strings.HasPrefix(key, vaultReferenceKeyPrefix) {
			continue
		}
		src, err := url.Parse(value)
		if err != nil {
			return fmt.Errorf("Unable to parse URL %q: %s", value, err)
		}

		if v.VaultSecretProvider != nil && src.Scheme == vaultReferenceURLScheme {
			if _, err := v.VaultSecretProvider.GetValues(src); err != nil {
				return fmt.Errorf("Error resolving env variables: %s", err)
			}
		}
	}

	return nil
}
