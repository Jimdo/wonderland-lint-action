package validator

import (
	"fmt"
	"strconv"
)

type ContainerCapacity struct {
	CPUCapacitySpecifications map[string]uint
	CPUMinCapacity            uint
	CPUMaxCapacity            uint

	MemoryCapacitySpecifications map[string]uint
	MemoryMinCapacity            uint
	MemoryMaxCapacity            uint
}

func (v *ContainerCapacity) ValidateCPUCapacity(capacity string) error {
	if v.isCapacityShirtSize(capacity) {
		if _, ok := v.CPUCapacitySpecifications[capacity]; !ok {
			return fmt.Errorf("%q is not a valid cpu capacity", capacity)
		}
	}
	return nil
}

func (v *ContainerCapacity) ValidateCPUCapacityBoundaries(capacity uint) error {
	if capacity < v.CPUMinCapacity {
		return fmt.Errorf("CPU capacity must be greater than %d", v.CPUMinCapacity)
	}
	if capacity > v.CPUMaxCapacity {
		return fmt.Errorf("CPU capacity must be lower than %d", v.CPUMaxCapacity)
	}
	return nil
}

func (v *ContainerCapacity) ValidateMemoryCapacity(capacity string) error {
	if v.isCapacityShirtSize(capacity) {
		if _, ok := v.MemoryCapacitySpecifications[capacity]; !ok {
			return fmt.Errorf("%q is not a valid memory capacity", capacity)
		}
	}
	return nil
}

func (v *ContainerCapacity) ValidateMemoryCapacityBoundaries(capacity uint) error {
	if capacity < v.MemoryMinCapacity {
		return fmt.Errorf("Memory capacity must be greater than %d", v.MemoryMinCapacity)
	}
	if capacity > v.MemoryMaxCapacity {
		return fmt.Errorf("Memory capacity must be lower than %d", v.MemoryMaxCapacity)
	}
	return nil
}

func (v *ContainerCapacity) isCapacityShirtSize(capacity string) bool {
	_, err := strconv.Atoi(capacity)
	return err != nil
}
