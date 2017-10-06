package cron

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

const (
	cronPrefix      = "cron--"
	TimeoutExitCode = 201
)

func GetNameByResource(resourceName string) string {
	return strings.TrimPrefix(resourceName, cronPrefix)
}

func GetResourceByName(cronName string) string {
	return cronPrefix + cronName
}

func IsCron(container *ecs.Container) (bool, error) {
	return strings.HasPrefix(aws.StringValue(container.Name), cronPrefix), nil
}

func GetUserContainerFromTask(t *ecs.Task) *ecs.Container {
	for _, c := range t.Containers {
		if aws.StringValue(c.Name) != "timeout" {
			return c
		}
	}
	return nil
}

func GetTimeoutContainerFromTask(t *ecs.Task) *ecs.Container {
	for _, c := range t.Containers {
		if aws.StringValue(c.Name) == "timeout" {
			return c
		}
	}
	return nil
}
