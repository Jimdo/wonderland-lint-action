package validation

import (
	"fmt"

	wonderlandValidator "github.com/Jimdo/wonderland-validator/validator"

	"github.com/Jimdo/wonderland-crons/cron"
)

type containerDescription struct {
	Capacity *wonderlandValidator.ContainerCapacity
	Image    *wonderlandValidator.DockerImage
	Name     *wonderlandValidator.WonderlandName
}

func (v *containerDescription) validate(container *cron.ContainerDescription) error {
	if err := v.validateContainerName(container.Name); err != nil {
		return err
	}
	if err := v.validateCapacityDescription(container.Capacity); err != nil {
		return err
	}
	if err := v.Image.Validate(container.Image); err != nil {
		return Error{err.Error()}
	}
	return nil
}

func (v *containerDescription) validateContainerName(name string) error {
	if err := v.Name.Validate(name); err != nil {
		return Error{fmt.Sprintf("'%s' is not a valid component name. Please choose your components' name from the alphabet [a-zA-Z0-9-]+", name)}
	}
	return nil
}

func (v *containerDescription) validateCapacityDescription(capacity *cron.CapacityDescription) error {
	if err := v.Capacity.ValidateMemoryCapacity(capacity.Memory); err != nil {
		return Error{err.Error()}
	}
	if err := v.Capacity.ValidateMemoryCapacityBoundaries(capacity.MemoryLimit()); err != nil {
		return Error{err.Error()}
	}

	if err := v.Capacity.ValidateCPUCapacity(capacity.CPU); err != nil {
		return Error{err.Error()}
	}
	if err := v.Capacity.ValidateCPUCapacityBoundaries(capacity.CPULimit()); err != nil {
		return Error{err.Error()}
	}

	return nil
}
