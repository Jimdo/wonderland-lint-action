package vault

import (
	"fmt"
	"net/url"

	"github.com/hashicorp/vault/api"
)

// SecretProvider implements an EnvProvider to access static key-value-pairs stored
// in the /secret path of the Vault
type SecretProvider struct {
	VaultClients map[string]*api.Client
}

// GetValues reads key-value-pairs from Vault and returns a map of them as env variables
func (v SecretProvider) GetValues(src *url.URL) (map[string]string, error) {
	host := src.Host
	path := fmt.Sprintf("secret/%s", src.Path)

	c, ok := v.VaultClients[host]
	if !ok {
		return nil, fmt.Errorf("Cannot access credentials in unknown Vault instance %s", host)
	}

	secret, err := c.Logical().Read(path)
	if err != nil {
		return nil, fmt.Errorf("Error accessing credentials in Vault: %s", err)
	}
	if secret == nil {
		return nil, fmt.Errorf("No credentials found in Vault at path %q", path)
	}

	result := map[string]string{}
	for k, v := range secret.Data {
		result[k] = v.(string)
	}
	return result, nil
}
