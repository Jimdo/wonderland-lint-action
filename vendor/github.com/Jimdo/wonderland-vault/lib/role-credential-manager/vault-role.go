package rcm

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/hashicorp/vault/api"
)

// RoleCredentialManager is a helper to renew periodic Vault tokens
// issues by the approle backend and to fetch data from Vault
type RoleCredentialManager struct {
	// HashSecrets enables / disables secret hashing in debug logging
	HashSecrets bool

	// VaultClient exposes an authenticated Vault client whose token will
	// automatically get renewed by the RoleCredentialManager
	VaultClient *api.Client

	// The Logger used to print debug information
	Logger Logger

	roleID  string
	errChan chan error

	awsAuthExpire time.Time
	awsConfig     *aws.Config
}

// New creates a new instance of the RoleCredentialManager with debugging disabled
func New(vaultAddress, roleID string) (*RoleCredentialManager, error) {
	return NewWithLogger(vaultAddress, roleID, nil)
}

// NewWithDebug creates a new instance of the RoleCredentialManager with debugging enabled
func NewWithLogger(vaultAddress string, roleID string, logger Logger) (*RoleCredentialManager, error) {
	vc, err := api.NewClient(&api.Config{Address: vaultAddress, MaxRetries: 3})
	if err != nil {
		return nil, err
	}

	rcm := &RoleCredentialManager{
		Logger:      logger,
		HashSecrets: true,

		roleID:      roleID,
		VaultClient: vc,
		errChan:     make(chan error, 10),
	}

	if t, err := rcm.tokenFromRole(); err != nil {
		return nil, err
	} else {
		rcm.VaultClient.SetToken(t)
	}

	return rcm, nil
}

func (r *RoleCredentialManager) tokenFromRole() (string, error) {
	sec, err := r.VaultClient.Logical().Write("auth/approle/login", map[string]interface{}{
		"role_id": r.roleID,
	})

	if err != nil {
		return "", err
	}

	if sec.Auth == nil || sec.Auth.ClientToken == "" {
		return "", errors.New("Did not get a client token.")
	}

	r.dbg("Received token %s with lease time %ds", r.hashedSecret(sec.Auth.ClientToken), sec.Auth.LeaseDuration)

	go r.periodicRenew(sec.Auth.LeaseDuration)

	return sec.Auth.ClientToken, nil
}

func (r *RoleCredentialManager) periodicRenew(leaseDuration int) {
	for range time.Tick(time.Duration(leaseDuration-30) * time.Second) {
		if _, err := r.VaultClient.Auth().Token().RenewSelf(leaseDuration); err != nil {
			r.dbg("Internal error occurred: %s", err)
			r.errChan <- err
			return
		}
		r.dbg("Renewed token %s", r.hashedSecret(r.VaultClient.Token()))
	}
}

// GetAWSConfig retrieves an AWS configuration with credentials set
// after reading them from `/v1/aws/sts/{role}` endpoint in Vault
//
// Example usage:
//
//     c, err := rcmInstance.GetAWSConfig(cfg.VaultAWSRole, cfg.AWSRegion)
//     if err != nil {
//             log.Fatalf("Unable to get AWS Config: %s", err)
//     }
//     ec2Service := ec2.New(session.Must(session.NewSession(c)))
func (r *RoleCredentialManager) GetAWSConfig(role, region string) (*aws.Config, error) {
	if r.awsConfig != nil && time.Now().Before(r.awsAuthExpire) {
		return r.awsConfig, nil
	}

	d, err := r.VaultClient.Logical().Read("aws/sts/" + role)
	if err != nil {
		return nil, err
	}

	if d.Data == nil {
		return nil, errors.New("Did not receive AWS login data")
	}

	for _, k := range []string{"access_key", "secret_key", "security_token"} {
		if v, ok := d.Data[k]; !ok || v == "" {
			return nil, fmt.Errorf("Response from Vault was missing field '%s'", k)
		}
	}

	r.awsConfig = &aws.Config{
		Credentials: credentials.NewStaticCredentials(
			d.Data["access_key"].(string),
			d.Data["secret_key"].(string),
			d.Data["security_token"].(string),
		),
		Region: aws.String(region),
	}
	r.awsAuthExpire = time.Now().Add(time.Duration(d.LeaseDuration) * time.Second)

	r.dbg("Received AWS STS credentials with %ds lease duration: access_key=%s secret_key=%s security_token=%s",
		d.LeaseDuration,
		r.hashedSecret(d.Data["access_key"].(string)),
		r.hashedSecret(d.Data["secret_key"].(string)),
		r.hashedSecret(d.Data["security_token"].(string)),
	)

	return r.awsConfig, nil
}

// Error provides a reading channel which is triggered when internal
// errors occurs. For example this will yield an error when the
// renewal of the Vault token failed for some reason.
func (r *RoleCredentialManager) Error() <-chan error {
	return r.errChan
}

func (r *RoleCredentialManager) hashedSecret(secret string) string {
	if r.HashSecrets {
		return fmt.Sprintf("sha1:%x", sha1.Sum([]byte(secret)))
	}
	return secret
}

func (r *RoleCredentialManager) dbg(format string, v ...interface{}) {
	if r.Logger != nil {
		r.Logger.Debugf(fmt.Sprintf("rcm: %s", format), v...)
	}
}
