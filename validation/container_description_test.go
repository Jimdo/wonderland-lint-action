// +build integration

package validation

import (
	"testing"

	wonderlandValidator "github.com/Jimdo/wonderland-validator/validator"

	"github.com/Jimdo/wonderland-crons/cron"
)

func TestValidateContainerName_Valid(t *testing.T) {
	name := "valid-container-name"

	v := &containerDescription{
		Name: &wonderlandValidator.WonderlandName{},
	}
	if err := v.validateContainerName(name); err != nil {
		t.Errorf("%s should be a valid container name", name)
	}
}

func TestValidateContainerName_Invalid(t *testing.T) {
	name := "098_+baaaaz_"

	v := &containerDescription{
		Name: &wonderlandValidator.WonderlandName{},
	}
	if err := v.validateContainerName(name); err == nil {
		t.Errorf("%s should not be a valid container name", name)
	}
}

func TestValidateCapacityDescription_Valid(t *testing.T) {
	valid := []cron.CapacityDescription{
		{Memory: "XS", CPU: "XS"},
		{Memory: "S", CPU: "L"},
		{Memory: "M", CPU: "XL"},
		{Memory: "XXL", CPU: "2XL"},
		{Memory: "XXXL", CPU: "3XL"},
		{Memory: "1023", CPU: "1028"},
		{Memory: "S", CPU: "1023"},
		{Memory: "1023", CPU: "XL"},
	}

	v := &containerDescription{
		Capacity: &wonderlandValidator.ContainerCapacity{
			CPUCapacitySpecifications: cron.CPUCapacitySpecifications,
			CPUMinCapacity:            cron.MinCPUCapacity,
			CPUMaxCapacity:            cron.MaxCPUCapacity,

			MemoryCapacitySpecifications: cron.MemoryCapacitySpecifications,
			MemoryMinCapacity:            cron.MinMemoryCapacity,
			MemoryMaxCapacity:            cron.MaxMemoryCapacity,
		},
	}
	for _, capacity := range valid {
		if err := v.validateCapacityDescription(&capacity); err != nil {
			t.Errorf("%+v should be a valid capacity description: %s", capacity, err)
		}
	}
}

func TestValidateCapacityDescription_Invalid(t *testing.T) {
	invalid := []cron.CapacityDescription{
		{Memory: "A", CPU: "XS"},
		{Memory: "XS", CPU: "X"},
		{Memory: "XXL", CPU: "XXXXL"},
		{Memory: "4XL", CPU: "XXXXL"},
		{Memory: "1", CPU: "XL"},
		{Memory: "L", CPU: "2"},
		{Memory: "1", CPU: "2"},
		{Memory: "10000", CPU: "XL"},
		{Memory: "L", CPU: "2096"},
		{Memory: "10000", CPU: "2096"},
	}

	v := &containerDescription{
		Capacity: &wonderlandValidator.ContainerCapacity{
			CPUCapacitySpecifications: cron.CPUCapacitySpecifications,
			CPUMinCapacity:            cron.MinCPUCapacity,
			CPUMaxCapacity:            cron.MaxCPUCapacity,

			MemoryCapacitySpecifications: cron.MemoryCapacitySpecifications,
			MemoryMinCapacity:            cron.MinMemoryCapacity,
			MemoryMaxCapacity:            cron.MaxMemoryCapacity,
		},
	}
	for _, capacity := range invalid {
		if err := v.validateCapacityDescription(&capacity); err == nil {
			t.Errorf("%+v should not be a valid capacity description", capacity)
		}
	}
}
