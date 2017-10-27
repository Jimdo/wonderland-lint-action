package validator

import "testing"

func TestValidateCPUCapacity(t *testing.T) {
	valid := []string{
		"XS",
		"S",
		"1023",
	}

	v := &ContainerCapacity{
		CPUCapacitySpecifications: map[string]uint{
			"XS": 128,
			"S":  256,
		},
	}
	for _, capacity := range valid {
		if err := v.ValidateCPUCapacity(capacity); err != nil {
			t.Errorf("%q should be a valid capacity: %s", capacity, err)
		}
	}
}

func TestValidateCPUCapacity_Invalid(t *testing.T) {
	invalid := []string{
		"L",
		"3XL",
	}

	v := &ContainerCapacity{
		CPUCapacitySpecifications: map[string]uint{
			"XS": 128,
			"S":  256,
		},
	}
	for _, capacity := range invalid {
		if err := v.ValidateCPUCapacity(capacity); err == nil {
			t.Errorf("%q should be an invalid capacity: %s", capacity, err)
		}
	}
}

func TestValidateCPUCapacityBoundaries(t *testing.T) {
	valid := []uint{
		128,
		129,
		256,
	}

	v := &ContainerCapacity{
		CPUMinCapacity: 128,
		CPUMaxCapacity: 256,
	}
	for _, capacity := range valid {
		if err := v.ValidateCPUCapacityBoundaries(capacity); err != nil {
			t.Errorf("%d should be inside the valid CPU capacity: %s", capacity, err)
		}
	}
}

func TestValidateCPUCapacityBoundaries_Invalid(t *testing.T) {
	invalid := []uint{
		127,
		257,
	}

	v := &ContainerCapacity{
		CPUMinCapacity: 128,
		CPUMaxCapacity: 256,
	}
	for _, capacity := range invalid {
		if err := v.ValidateCPUCapacityBoundaries(capacity); err == nil {
			t.Errorf("%d should be outside the valid CPU capacity: %s", capacity, err)
		}
	}
}

func TestValidateMemoryCapacity(t *testing.T) {
	valid := []string{
		"XS",
		"S",
		"1023",
	}

	v := &ContainerCapacity{
		MemoryCapacitySpecifications: map[string]uint{
			"XS": 128,
			"S":  256,
		},
	}
	for _, capacity := range valid {
		if err := v.ValidateMemoryCapacity(capacity); err != nil {
			t.Errorf("%q should be a valid capacity: %s", capacity, err)
		}
	}
}

func TestValidateMemoryCapacity_Invalid(t *testing.T) {
	valid := []string{
		"L",
		"3XL",
	}

	v := &ContainerCapacity{
		MemoryCapacitySpecifications: map[string]uint{
			"XS": 128,
			"S":  256,
		},
	}
	for _, capacity := range valid {
		if err := v.ValidateMemoryCapacity(capacity); err == nil {
			t.Errorf("%q should be an invalid capacity: %s", capacity, err)
		}
	}
}

func TestValidateMemoryCapacityBoundaries(t *testing.T) {
	valid := []uint{
		128,
		129,
		256,
	}

	v := &ContainerCapacity{
		MemoryMinCapacity: 128,
		MemoryMaxCapacity: 256,
	}
	for _, capacity := range valid {
		if err := v.ValidateMemoryCapacityBoundaries(capacity); err != nil {
			t.Errorf("%d should be inside the valid Memory capacity: %s", capacity, err)
		}
	}
}

func TestValidateMemoryCapacityBoundaries_Invalid(t *testing.T) {
	invalid := []uint{
		127,
		257,
	}

	v := &ContainerCapacity{
		MemoryMinCapacity: 128,
		MemoryMaxCapacity: 256,
	}
	for _, capacity := range invalid {
		if err := v.ValidateMemoryCapacityBoundaries(capacity); err == nil {
			t.Errorf("%d should be outside the valid Memory capacity: %s", capacity, err)
		}
	}
}
