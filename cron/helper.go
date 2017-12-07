package cron

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

const (
	CronPrefix           = "cron--"
	TimeoutExitCode      = 201
	TimeoutContainerName = "timeout"
)

func GetNameByResource(resourceName string) string {
	return strings.TrimPrefix(resourceName, CronPrefix)
}

func GetResourceByName(cronName string) string {
	return CronPrefix + cronName
}

func IsCron(container *ecs.Container) (bool, error) {
	return strings.HasPrefix(aws.StringValue(container.Name), CronPrefix), nil
}

func GetUserContainerFromTask(t *ecs.Task) *ecs.Container {
	for _, c := range t.Containers {
		if aws.StringValue(c.Name) != TimeoutContainerName {
			return c
		}
	}
	return nil
}

func GetTimeoutContainerFromTask(t *ecs.Task) *ecs.Container {
	for _, c := range t.Containers {
		if aws.StringValue(c.Name) == TimeoutContainerName {
			return c
		}
	}
	return nil
}
