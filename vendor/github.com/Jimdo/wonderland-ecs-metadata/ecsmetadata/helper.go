package ecsmetadata

import (
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/sirupsen/logrus"

	"github.com/Jimdo/wonderland-ecs-metadata/logger"
)

var (
	familyFromTaskDefinitionRegexp = regexp.MustCompile(`.*/(.*):\d+$`)
)

func getFamilyFromECSTask(task *ecs.Task) (string, error) {
	taskDefinitionArn := aws.StringValue(task.TaskDefinitionArn)

	familySubmatch := familyFromTaskDefinitionRegexp.FindStringSubmatch(taskDefinitionArn)
	if len(familySubmatch) != 2 {
		return "", fmt.Errorf("Could not find family in task definition arn %q for task %q", taskDefinitionArn, aws.StringValue(task.TaskArn))
	}

	family := familySubmatch[1]
	logger.Task("", task).WithFields(logrus.Fields{
		"family": family,
	}).Debugf("getFamily result")
	return family, nil
}
