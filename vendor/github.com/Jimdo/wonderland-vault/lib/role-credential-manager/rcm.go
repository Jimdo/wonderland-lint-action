package rcm

import (
	"errors"
	"fmt"
	"log"
	"time"

	"math"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/hashicorp/vault/api"
)

const (
	defaultAWSLeaseDurationSeconds = 60 * 60
	defaultVaultMaxRetries         = 3

	minimumLeaseRenewIntervalSeconds = 30

	vaultURIPathApproleLogin   = "auth/approle/login"
	vaultURIPathRenewToken     = "auth/token/renew-self"
	vaultURIPathSTSCredentials = "aws/sts/"
)

// New creates a new instance of the RoleCredentialManager
func New(vaultAddress, vaultRoleID string, options ...Option) (*RoleCredentialManager, error) {
	rcm := &RoleCredentialManager{
		cfg: config{
			vaultAddress:    vaultAddress,
			vaultMaxRetries: defaultVaultMaxRetries,
			vaultRoleID:     vaultRoleID,
		},
	}

	for _, option := range options {
		option(&rcm.cfg)
	}

	if err := rcm.cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %s", err)
	}

	return rcm, nil
}

type RoleCredentialManager struct {
	VaultClient      *api.Client
	vaultClientLease int

	cfg config

	awsTicker, vaultTicker *time.Ticker
}

// Init explicitly initializes all required tokens, so that VaultClient is ready to use
func (r *RoleCredentialManager) Init() error {
	if err := r.initializeVaultClient(); err != nil {
		return err
	}
	if err := r.renewVaultToken(r.vaultClientLease); err != nil {
		return fmt.Errorf("error getting Vault token: %s", err)
	}

	awsLeaseDuration := defaultAWSLeaseDurationSeconds
	if r.hasAWSConfig() {
		var err error
		if awsLeaseDuration, err = r.updateAWSCredentials(); err != nil {
			return fmt.Errorf("error getting AWS credentials: %s", err)
		}
	}

	r.awsTicker = time.NewTicker(calcRenewLeaseInterval(awsLeaseDuration))
	r.vaultTicker = time.NewTicker(calcRenewLeaseInterval(r.vaultClientLease))

	return nil
}

func (r *RoleCredentialManager) isInitialized() bool { return r.VaultClient != nil }

func (r *RoleCredentialManager) hasAWSConfig() bool {
	return r.cfg.awsIAMRole != "" && len(r.cfg.awsClientConfigs) != 0
}

// Run executes the runloop which refreshes the RoleCredentialManager's own Vault token
func (r *RoleCredentialManager) Run(stop <-chan interface{}) error {
	if !r.isInitialized() {
		r.Init()
	}

	defer r.awsTicker.Stop()
	defer r.vaultTicker.Stop()

	for {
		select {
		case <-r.vaultTicker.C:
			if err := r.renewVaultToken(r.vaultClientLease); err != nil {
				if !r.cfg.ignoreErrors {
					return fmt.Errorf("error renewing Vault token: %s", err)
				}
				r.warnf("Could not renew Vault token: %s", err)
			}
		case <-r.awsTicker.C:
			if r.hasAWSConfig() {
				if _, err := r.updateAWSCredentials(); err != nil {
					if !r.cfg.ignoreErrors {
						return fmt.Errorf("error updating AWS credentials: %s", err)
					}
					r.warnf("Could not update AWS credentials: %s", err)
				}
			}
		case <-stop:
			return nil
		}
	}
}

func (r *RoleCredentialManager) initializeVaultClient() error {
	vc, err := api.NewClient(&api.Config{
		Address:    r.cfg.vaultAddress,
		MaxRetries: r.cfg.vaultMaxRetries,
	})
	if err != nil {
		return err
	}

	sec, err := vc.Logical().Write(vaultURIPathApproleLogin, map[string]interface{}{
		"role_id": r.cfg.vaultRoleID,
	})
	if err != nil {
		return err
	}

	if sec.Auth == nil || sec.Auth.ClientToken == "" {
		return errors.New("Vault response did not contain a token")
	}

	vc.SetToken(sec.Auth.ClientToken)
	r.debugf("Vault token has a lease duration of %ds", sec.Auth.LeaseDuration)

	r.VaultClient = vc
	r.vaultClientLease = sec.Auth.LeaseDuration
	return nil
}

func (r *RoleCredentialManager) renewVaultToken(leaseDuration int) error {
	r.debug("Renewing Vault token")

	sec, err := r.VaultClient.Auth().Token().RenewSelf(leaseDuration)
	if err != nil {
		return err
	}

	r.debugf("Renewed Vault token has a lease duration of %ds", sec.Auth.LeaseDuration)
	return nil
}

func (r *RoleCredentialManager) updateAWSCredentials() (int, error) {
	r.debug("Updating AWS credentials")

	d, err := r.VaultClient.Logical().Read(vaultURIPathSTSCredentials + r.cfg.awsIAMRole)
	if err != nil {
		return 0, err
	}

	if d.Data == nil {
		return 0, errors.New("vault response did not contain AWS credentials")
	}

	for _, k := range []string{"access_key", "secret_key", "security_token"} {
		if v, ok := d.Data[k]; !ok || v == "" {
			return 0, fmt.Errorf("vault response was missing field '%s'", k)
		}
	}

	r.debugf("New AWS credentials have a lease duration of %ds", d.LeaseDuration)

	for _, awsCfg := range r.cfg.awsClientConfigs {
		awsCfg.MergeIn(&aws.Config{
			Credentials: credentials.NewStaticCredentials(
				d.Data["access_key"].(string),
				d.Data["secret_key"].(string),
				d.Data["security_token"].(string),
			),
		})
	}

	return d.LeaseDuration, nil
}

func (r *RoleCredentialManager) debug(msg string) {
	if r.cfg.logger != nil {
		r.cfg.logger.Debug(msg)
	} else {
		log.Printf("debug: " + msg)
	}
}

func (r *RoleCredentialManager) debugf(format string, args ...interface{}) {
	if r.cfg.logger != nil {
		r.cfg.logger.Debugf(format, args...)
	} else {
		log.Printf("debug: "+format, args...)
	}
}

func (r *RoleCredentialManager) warnf(format string, args ...interface{}) {
	if r.cfg.logger != nil {
		r.cfg.logger.Warnf(format, args...)
	} else {
		log.Printf("warn: "+format, args...)
	}
}

// calcRenewLeaseInterval returns the duration after which a Vault lease should be renewed. This number depends on the
// lease duration. It is at least 10% of the lease duration and at most 30s before the lease expires.
func calcRenewLeaseInterval(leaseDuration int) time.Duration {
	secondsBeforeExpiry := math.Floor(float64(leaseDuration) * 0.1)

	if secondsBeforeExpiry < minimumLeaseRenewIntervalSeconds && leaseDuration-minimumLeaseRenewIntervalSeconds > 0 {
		secondsBeforeExpiry = minimumLeaseRenewIntervalSeconds
	}

	return time.Duration(leaseDuration-int(secondsBeforeExpiry)) * time.Second
}
