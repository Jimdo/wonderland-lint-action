package validation

import (
	"fmt"

	"github.com/Jimdo/wonderland-crons/cron"
)

type containerDescription struct{}

func (v *containerDescription) validate(component *cron.ContainerDescription) error {
	if err := v.validateCapacityDescription(component.Capacity); err != nil {
		return err
	}
	return nil
}

func (v *containerDescription) validateCapacityDescription(capacity *cron.CapacityDescription) error {
	if capacity.MemoryIsTShirtSize() {
		if _, ok := cron.MemoryCapacitySpecifications[capacity.Memory]; !ok {
			return Error{fmt.Sprintf("'%s' is not a valid memory capacity", capacity.Memory)}
		}
	}
	if capacity.MemoryLimit() < cron.MinMemoryCapacity {
		return Error{fmt.Sprintf("Memory capacity must be greater than %d", cron.MinMemoryCapacity)}
	}
	if capacity.MemoryLimit() > cron.MaxMemoryCapacity {
		return Error{fmt.Sprintf("Memory capacity must be lower than %d", cron.MaxMemoryCapacity)}
	}

	if capacity.CPUIsTShirtSize() {
		if _, ok := cron.CPUCapacitySpecifications[capacity.CPU]; !ok {
			return Error{fmt.Sprintf("'%s' is not a valid cpu capacity", capacity.CPU)}
		}
	}
	if capacity.CPULimit() < cron.MinCPUCapacity {
		return Error{fmt.Sprintf("CPUcapacity must be greater than %d", cron.MinCPUCapacity)}
	}
	if capacity.CPULimit() > cron.MaxCPUCapacity {
		return Error{fmt.Sprintf("CPU capacity must be lower than %d", cron.MaxCPUCapacity)}
	}
	return nil
}
