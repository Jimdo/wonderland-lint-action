package cron

import (
	"encoding/json"
	"strconv"
	"strings"
)

var MemoryCapacitySpecifications = map[string]uint{
	"XS":   1,
	"S":    2,
	"M":    3,
	"L":    4,
	"XL":   5,
	"XXL":  6,
	"2XL":  6,
	"XXXL": 7,
	"3XL":  7,
}

var MinMemoryCapacity = MemoryTShirtSizeToUInt("XS")
var MaxMemoryCapacity = MemoryTShirtSizeToUInt("3XL")

var CPUCapacitySpecifications = map[string]uint{
	"XS":   1,
	"S":    2,
	"M":    3,
	"L":    4,
	"XL":   5,
	"XXL":  6,
	"2XL":  6,
	"XXXL": 7,
	"3XL":  7,
}

var MinCPUCapacity = CPUTShirtSizeToUInt("XS")
var MaxCPUCapacity = CPUTShirtSizeToUInt("3XL")

func NewCronDescriptionFromJSON(data []byte) (*CronDescription, error) {
	desc := &CronDescription{}
	if err := json.Unmarshal(data, desc); err != nil {
		return nil, err
	}
	desc.Init()

	return desc, nil
}

func (d *CronDescription) Init() {
	if d.Description == nil {
		return
	}

	d.Description.init()
}

type CronDescription struct {
	// Name is the old attribute that was used to indicate a cron's name
	// Deprecated: a cron's name is now provided in the URL path of the API endpoint
	Name        string                `json:"name"`
	Schedule    string                `json:"schedule"`
	Description *ContainerDescription `json:"description"`
}

type ContainerDescription struct {
	Image       string               `json:"image"`
	Arguments   []string             `json:"arguments,omitempty"`
	Environment map[string]string    `json:"env,omitempty"`
	Capacity    *CapacityDescription `json:"capacity,omitempty"`
}

const (
	CPUBase uint = 4
	MemBase uint = 5
)

type CapacityDescription struct {
	Memory string `json:"memory"`
	CPU    string `json:"cpu"`
}

func (cd CapacityDescription) memoryIsTShirtSize() bool {
	_, err := strconv.Atoi(cd.Memory)
	return err != nil
}

func (cd CapacityDescription) MemoryLimit() uint {
	if cd.memoryIsTShirtSize() {
		return MemoryTShirtSizeToUInt(cd.Memory)
	}
	i, _ := strconv.Atoi(cd.Memory)
	return uint(i)
}

func (cd CapacityDescription) cpuIsTShirtSize() bool {
	_, err := strconv.Atoi(cd.CPU)
	return err != nil
}

func (cd CapacityDescription) CPULimit() uint {
	if cd.cpuIsTShirtSize() {
		return CPUTShirtSizeToUInt(cd.CPU)
	}
	i, _ := strconv.Atoi(cd.CPU)
	return uint(i)
}

func (cd *CapacityDescription) init() {
	if cd.Memory == "" {
		cd.Memory = "XS"
	}
	if cd.CPU == "" {
		cd.CPU = "XS"
	}

	cd.Memory = strings.ToUpper(cd.Memory)
	cd.CPU = strings.ToUpper(cd.CPU)
}

func (c *ContainerDescription) init() {
	if c.Capacity == nil {
		c.Capacity = &CapacityDescription{}
	}

	c.Capacity.init()
}

func MemoryTShirtSizeToUInt(size string) uint {
	return 1 << (MemoryCapacitySpecifications[size] + MemBase)
}

func CPUTShirtSizeToUInt(size string) uint {
	return 1 << (CPUCapacitySpecifications[size] + CPUBase)
}
