package validation

import (
	"fmt"

	"github.com/gorhill/cronexpr"

	wonderlandValidator "github.com/Jimdo/wonderland-validator/validator"

	"github.com/Jimdo/wonderland-crons/cron"
)

const (
	MaxCronNameLength = 64
)

type cronDescription struct {
	Container *containerDescription
	Name      *wonderlandValidator.WonderlandName
}

// validate ensures that a cron description provided by a user
// is valid and can be used to create a Cron.
func (v *cronDescription) validate(desc *cron.CronDescription) error {
	if err := v.validateCronName(desc.Name); err != nil {
		return err
	}
	if err := v.validateCronSchedule(desc.Schedule); err != nil {
		return err
	}

	if desc.Description == nil {
		return Error{"Crons require a description."}
	}
	if err := v.Container.validate(desc.Description); err != nil {
		return err
	}

	return nil
}

func (v *cronDescription) validateCronName(name string) error {
	if len(name) > MaxCronNameLength {
		return Error{fmt.Sprintf("cron name %s is too long (max length is %d)", name, MaxCronNameLength)}
	}
	if err := v.Name.Validate(name); err != nil {
		return Error{fmt.Sprintf("'%s' is not a valid cron name. Please choose your crons name from the alphabet [a-zA-Z0-9-]+", name)}
	}
	return nil
}

func (v *cronDescription) validateCronSchedule(schedule string) error {
	if schedule == "" {
		return Error{"Every cron requires a schedule"}
	}
	if _, err := cronexpr.Parse(schedule); err != nil {
		return Error{fmt.Sprintf("%q is not a valid cron schedule", schedule)}
	}
	return nil
}
