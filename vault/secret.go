package vault

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// {
//   "auth": null,
//   "data": {
//       "KEY1": "foo",
//       "KEY2": "bar",
//       "key3": "goo"
//   },
//   "lease_duration": 2592000,
//   "lease_id": "secret/kahlers/testapp/test/2c6cd460-9036-e853-091d-79b47f6e581e",
//   "renewable": false
// }
type vaultSecretResponse struct {
	Data   map[string]string `json:"data"`
	Errors []string          `json:"errors"`
}

// SecretProvider implements an EnvProvider to access static key-value-pairs stored
// in the /secret path of the Vault
type SecretProvider struct {
	HTTPClient       *http.Client
	VaultAccessToken string
}

// GetValues reads key-value-pairs from Vault and returns a map of them as env variables
func (v SecretProvider) GetValues(src *url.URL) (map[string]string, error) {
	if v.HTTPClient == nil {
		v.HTTPClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	var (
		token    string
		hasToken bool
	)

	if src.User == nil && v.VaultAccessToken == "" {
		return nil, fmt.Errorf("You need to provide a token")
	}

	if src.User != nil {
		token, hasToken = src.User.Password()
	}

	host := src.Host
	path := src.Path

	if !hasToken && v.VaultAccessToken == "" {
		return nil, fmt.Errorf("You need to provide a token")
	}

	if token == "" {
		token = v.VaultAccessToken
	}

	// curl \
	// -H "X-Vault-Token: f3b09679-3001-009d-2b80-9c306ab81aa6" \
	// -X GET \
	//  http://127.0.0.1:8200/v1/secret/foo

	vaultURL := url.URL{}
	vaultURL.Scheme = "https" // We will not support unsecure vaults.
	vaultURL.Host = host
	vaultURL.Path = fmt.Sprintf("/v1/secret%s", path)

	req, err := http.NewRequest("GET", vaultURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Vault-Token", token)

	resp, err := v.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 404:
		return nil, fmt.Errorf("Path secret%s not found in Vault", path)
	case 403:
		return nil, fmt.Errorf("Authorization against Vault failed")
	}

	decodedResponse := vaultSecretResponse{}
	err = json.NewDecoder(resp.Body).Decode(&decodedResponse)
	if err != nil {
		return nil, err
	}

	if len(decodedResponse.Errors) > 0 {
		return nil, fmt.Errorf("Unhandled Errors occurred: %s", strings.Join(decodedResponse.Errors, ", "))
	}

	return decodedResponse.Data, nil
}
