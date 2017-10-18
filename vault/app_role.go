package vault

import (
	"fmt"

	"github.com/hashicorp/vault/api"
)

const (
	vaultRoleIDPathTemplate = "auth/approle/role/%s/role-id"
	vaultRoleIDKey          = "role_id"
)

type AppRoleProvider struct {
	VaultClient *api.Client
}

func (p *AppRoleProvider) RoleID(role string) (string, error) {
	path := fmt.Sprintf(vaultRoleIDPathTemplate, role)
	values, err := p.VaultClient.Logical().Read(path)
	if err != nil {
		return "", fmt.Errorf("Unable to read Vault app role ID: %s", err)
	}
	if values == nil {
		return "", nil
	}
	return values.Data[vaultRoleIDKey].(string), nil
}

func (p *AppRoleProvider) VaultAddress() string {
	return p.VaultClient.Address()
}
