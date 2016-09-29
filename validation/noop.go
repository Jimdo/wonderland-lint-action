package validation

import "github.com/Jimdo/wonderland-cron/cron"

func NewNoopValidator() Validator {
	return &noopValidator{}
}

type noopValidator struct {
}

func (v *noopValidator) ValidateCronDescription(*cron.CronDescription) error {
	return nil
}
