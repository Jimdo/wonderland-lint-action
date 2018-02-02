package validation

import (
	"fmt"

	wonderlandValidator "github.com/Jimdo/wonderland-validator/validator"

	"github.com/Jimdo/wonderland-crons/cron"
)

const (
	MaxCronNameLength = 64
)

type cronDescription struct {
	Container        *containerDescription
	CronNotification *cronNotification
	Name             *wonderlandValidator.WonderlandName
}

// validate ensures that a cron description provided by a user
// is valid and can be used to create a Cron.
func (v *cronDescription) validate(desc *cron.Description) error {
	if desc.Description == nil {
		return Error{"Crons require a description."}
	}
	if err := v.Container.validate(desc.Description); err != nil {
		return err
	}
	if desc.Notifications != nil {
		return v.CronNotification.validate(desc.Notifications)
	}
	return nil
}

func (v *cronDescription) ValidateCronName(name string) error {
	if len(name) > MaxCronNameLength {
		return Error{fmt.Sprintf("cron name %s is too long (max length is %d)", name, MaxCronNameLength)}
	}
	if err := v.Name.Validate(name); err != nil {
		return Error{fmt.Sprintf("'%s' is not a valid cron name. Please choose your crons name from the alphabet [a-zA-Z0-9-]+", name)}
	}
	return nil
}
