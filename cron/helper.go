package cron

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

const (
	cronPrefix = "cron--"
)

func GetNameByResource(resourceName string) string {
	return strings.TrimPrefix(resourceName, cronPrefix)
}

func GetResourceByName(cronName string) string {
	return cronPrefix + cronName
}

func IsCron(task *ecs.Task) (bool, error) {
	if task.Overrides == nil || len(task.Overrides.ContainerOverrides) == 0 {
		return false, fmt.Errorf("Task has no overrides defined, cannot discover name")
	}
	return strings.HasPrefix(aws.StringValue(task.Overrides.ContainerOverrides[0].Name), cronPrefix), nil
}
