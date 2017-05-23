package vault

import (
	"fmt"
	"net/url"

	"github.com/hashicorp/vault/api"
)

// SecretProvider implements an EnvProvider to access static key-value-pairs stored
// in the /secret path of the Vault
type SecretProvider struct {
	VaultClient *api.Client
}

// GetValues reads key-value-pairs from Vault and returns a map of them as env variables
func (v SecretProvider) GetValues(src *url.URL) (map[string]string, error) {
	path := fmt.Sprintf("secret%s", src.Path)

	secret, err := v.VaultClient.Logical().Read(path)
	if err != nil {
		return nil, fmt.Errorf("Error accessing credentials in Vault: %s", err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("No credentials found in Vault at path %q", path)
	}

	result := map[string]string{}
	for k, v := range secret.Data {
		result[k] = v.(string)
	}
	return result, nil
}
