package rcm

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
)

// A Option is used to set configuration options in config structs
type Option func(*config)

// WithAWSIAMRole returns a Option that sets the AWS IAM role on a config struct
func WithAWSIAMRole(role string) Option {
	return func(cfg *config) {
		cfg.awsIAMRole = role
	}
}

// WithAWSClientConfigs returns a Option that sets the AWS client configs on a config struct
func WithAWSClientConfigs(configs ...*aws.Config) Option {
	return func(cfg *config) {
		cfg.awsClientConfigs = configs
	}
}

// WithIgnoreErrors returns a Option that sets the ignore errors option on a config struct
func WithIgnoreErrors() Option {
	return func(cfg *config) {
		cfg.ignoreErrors = true
	}
}

// WithLogger returns a Option that sets the logger on a config struct
func WithLogger(logger Logger) Option {
	return func(cfg *config) {
		cfg.logger = logger
	}
}

// WithMaxVaultRetries returns a Option that sets the maximum number of retries of Vault requests on a config struct
func WithMaxVaultRetries(numRetries int) Option {
	return func(cfg *config) {
		cfg.vaultMaxRetries = numRetries
	}
}

type config struct {
	awsIAMRole       string
	awsClientConfigs []*aws.Config

	ignoreErrors bool

	logger Logger

	vaultMaxRetries int

	vaultAddress string
	vaultRoleID  string
}

func (cfg config) validate() error {
	if cfg.vaultAddress == "" {
		return errors.New("no Vault address configured")
	}
	if cfg.vaultRoleID == "" {
		return errors.New("no Vault role ID configured")
	}
	if len(cfg.awsClientConfigs) > 0 && cfg.awsIAMRole == "" {
		return errors.New("no AWS IAM role given (required for retrieving AWS credentials)")
	}

	return nil
}
