package aws

import (
	"strings"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"

	"github.com/Jimdo/wonderland-crons/cron"
)

type ECSTaskDefinitionMapper struct{}

func NewECSTaskDefinitionMapper() *ECSTaskDefinitionMapper {
	return &ECSTaskDefinitionMapper{}
}

func (tds *ECSTaskDefinitionMapper) ContainerDefinitionFromCronDescription(containerName string, cron *cron.CronDescription, cronName string) *ecs.ContainerDefinition {
	var envVars []*ecs.KeyValuePair
	for key, value := range cron.Description.Environment {
		envVars = append(envVars, &ecs.KeyValuePair{
			Name:  awssdk.String(key),
			Value: awssdk.String(value),
		})
	}

	return &ecs.ContainerDefinition{
		Command: awssdk.StringSlice(cron.Description.Arguments),
		Cpu:     awssdk.Int64(int64(cron.Description.Capacity.CPULimit())),
		DockerLabels: map[string]*string{
			"com.jimdo.wonderland.cron":     awssdk.String(cronName),
			"com.jimdo.wonderland.logtypes": awssdk.String(strings.Join(cron.Description.LoggingTypes(), ",")),
		},
		Environment: envVars,
		Image:       awssdk.String(cron.Description.Image),
		Memory:      awssdk.Int64(int64(cron.Description.Capacity.MemoryLimit())),
		Name:        awssdk.String(containerName),
	}
}
