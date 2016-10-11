package vault

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/vault/api"
)

const (
	vaultRoleIDPathTemplate = "auth/approle/role/%s/role-id"
	vaultRoleIDKey          = "role_id"
)

type AppRoleProvider struct {
	VaultAddress string
	VaultToken   string
}

func (p *AppRoleProvider) RoleID(role string) (string, error) {
	client, err := p.vaultClient()
	if err != nil {
		return "", fmt.Errorf("Unable to initialize Vault client: %s", err)
	}

	path := fmt.Sprintf(vaultRoleIDPathTemplate, role)
	values, err := client.Logical().Read(path)
	if err != nil {
		return "", fmt.Errorf("Unable to read Vault app role ID: %s", err)
	}
	if values == nil {
		return "", nil
	}
	return values.Data[vaultRoleIDKey].(string), nil
}

func (p *AppRoleProvider) vaultClient() (*api.Client, error) {
	client, err := api.NewClient(&api.Config{
		Address:    p.VaultAddress,
		HttpClient: &http.Client{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, err
	}
	client.SetToken(p.VaultToken)

	return client, nil
}
