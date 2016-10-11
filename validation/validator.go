package validation

import "github.com/Jimdo/wonderland-crons/cron"

type Validator interface {
	ValidateCronDescription(*cron.CronDescription) error
}

func NewNoopValidator() Validator {
	return &noopValidator{}
}

type noopValidator struct {
}

func (v *noopValidator) ValidateCronDescription(*cron.CronDescription) error {
	return nil
}
