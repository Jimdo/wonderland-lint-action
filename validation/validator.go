package validation

import (
	"github.com/Jimdo/wonderland-crons/cron"

	wonderlandValidator "github.com/Jimdo/wonderland-validator/validator"
)

type Validator struct {
	cron *cronDescription
}

type Configuration struct {
	CapacityValidator        *wonderlandValidator.ContainerCapacity
	DockerImageValidator     *wonderlandValidator.DockerImage
	WonderlandNameValidator  *wonderlandValidator.WonderlandName
	MetaInformationValidator *wonderlandValidator.MetaInformation
	EnvironmentVariables     *wonderlandValidator.EnvironmentVariables
}

func New(cfg Configuration) *Validator {
	cron := &cronDescription{
		Container: &containerDescription{
			Capacity:             cfg.CapacityValidator,
			Image:                cfg.DockerImageValidator,
			EnvironmentVariables: cfg.EnvironmentVariables,
		},
		Name:             cfg.WonderlandNameValidator,
		MetaInformation:  cfg.MetaInformationValidator,
		CronNotification: &cronNotification{},
	}

	return &Validator{
		cron: cron,
	}
}

func (v *Validator) ValidateCronDescription(cd *cron.Description) error {
	return v.cron.validate(cd)
}

func (v *Validator) ValidateCronName(name string) error {
	return v.cron.ValidateCronName(name)
}
