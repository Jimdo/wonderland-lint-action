package validation

import "github.com/Jimdo/wonderland-crons/cron"

type Validator interface {
	ValidateCronDescription(*cron.CronDescription) error
}

func New() Validator {
	return &validator{
		cron: &cronDescription{
			Component: &containerDescription{},
			Name:      &name{},
		},
	}
}

// A Validator is an object that can validate user input. Its main
// purpose is to validate service descriptions provided in deployment
// requests.
//
// The advantage of a Validator struct is that it can be configured
// from the outside to be aware of the deployer's configuration.
// The main entry point is the method ValidateserviceDescription
// which will call all other validation steps internally.
type validator struct {
	cron *cronDescription
}

func (v *validator) ValidateCronDescription(cd *cron.CronDescription) error {
	return v.cron.validate(cd)
}
