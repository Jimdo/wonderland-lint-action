package aws

import (
	"fmt"
	"strconv"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/sirupsen/logrus"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-ecs-metadata/ecsmetadata"
)

type TaskDefinitionStore interface {
	AddRevisionFromCronDescription(string, *cron.CronDescription) (string, string, error)
	DeleteByFamily(string) error
	RunTaskDefinition(string) (*ecs.Task, error)
	GetRunningTasksByFamily(string) ([]string, error)
}

type ECSTaskDefinitionStore struct {
	ecs         ecsiface.ECSAPI
	tdm         *ECSTaskDefinitionMapper
	ecsMetadata ecsmetadata.Metadata

	clusterARN                string
	ecsRunnerIdentifier       string
	noScheduleMarkerAttribute string
	timeoutImage              string
}

func NewECSTaskDefinitionStore(e ecsiface.ECSAPI, tdm *ECSTaskDefinitionMapper, ecsm ecsmetadata.Metadata, clusterARN, ecsRunnerIdentifier, noScheduleMarkerAttribute, timeoutImage string) *ECSTaskDefinitionStore {
	return &ECSTaskDefinitionStore{
		ecs:                       e,
		tdm:                       tdm,
		ecsMetadata:               ecsm,
		clusterARN:                clusterARN,
		ecsRunnerIdentifier:       ecsRunnerIdentifier,
		noScheduleMarkerAttribute: noScheduleMarkerAttribute,
		timeoutImage:              timeoutImage,
	}
}

func (tds *ECSTaskDefinitionStore) AddRevisionFromCronDescription(cronName string, desc *cron.CronDescription) (string, string, error) {
	tdFamilyName := cron.GetResourceByName(cronName)

	cd, err := tds.tdm.ContainerDefinitionFromCronDescription(tdFamilyName, desc, cronName)
	if err != nil {
		return "", "", fmt.Errorf("could not generate ECS container definition from cron description: %s", err)
	}

	rtdInput := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{cd},
		Family:               awssdk.String(tdFamilyName),
	}

	if tds.noScheduleMarkerAttribute != "" {
		rtdInput.PlacementConstraints = []*ecs.TaskDefinitionPlacementConstraint{
			{
				Expression: awssdk.String(fmt.Sprintf("attribute:%s not_exists", tds.noScheduleMarkerAttribute)),
				Type:       awssdk.String("memberOf"),
			},
		}
	}

	timeoutSidecarDefinition := tds.createTimeoutSidecarDefinition(cronName, desc)

	rtdInput.ContainerDefinitions = append(rtdInput.ContainerDefinitions, timeoutSidecarDefinition)
	out, err := tds.ecs.RegisterTaskDefinition(rtdInput)
	if err != nil {
		return "", "", fmt.Errorf("could not register task definition for family %q with error: %s", tdFamilyName, err)
	}
	return awssdk.StringValue(out.TaskDefinition.TaskDefinitionArn), tdFamilyName, nil
}

func (tds *ECSTaskDefinitionStore) createTimeoutSidecarDefinition(cronName string, desc *cron.CronDescription) *ecs.ContainerDefinition {
	timeoutString := strconv.FormatInt(*desc.Timeout, 10)
	timeoutExitCodeString := strconv.FormatInt(cron.TimeoutExitCode, 10)
	timeoutSidecarDefinition := &ecs.ContainerDefinition{
		Command: awssdk.StringSlice([]string{timeoutString, timeoutExitCodeString}),
		Cpu:     awssdk.Int64(int64(16)),
		DockerLabels: map[string]*string{
			"com.jimdo.wonderland.cron": awssdk.String(cronName),
		},
		Image:  awssdk.String(tds.timeoutImage),
		Memory: awssdk.Int64(int64(32)),
		Name:   awssdk.String(cron.TimeoutContainerName),
	}

	return timeoutSidecarDefinition

}

func (tds *ECSTaskDefinitionStore) DeleteByFamily(family string) error {
	err := tds.ecs.ListTaskDefinitionsPages(&ecs.ListTaskDefinitionsInput{
		FamilyPrefix: awssdk.String(family),
	}, func(out *ecs.ListTaskDefinitionsOutput, last bool) bool {
		for _, arn := range out.TaskDefinitionArns {
			_, err := tds.ecs.DeregisterTaskDefinition(&ecs.DeregisterTaskDefinitionInput{
				TaskDefinition: arn,
			})
			// TODO: How to handle this case best?
			if err != nil {
				logrus.WithField("ARN", awssdk.StringValue(arn)).
					WithError(err).
					Error("Could not delete TaskDefinition")
			}
		}
		return !last
	})
	if err != nil {
		return fmt.Errorf("could not list targets of family %q with error: %s", family, err)
	}
	return nil
}

func (tds *ECSTaskDefinitionStore) RunTaskDefinition(arn string) (*ecs.Task, error) {
	out, err := tds.ecs.RunTask(&ecs.RunTaskInput{
		Cluster:        awssdk.String(tds.clusterARN),
		StartedBy:      awssdk.String(tds.ecsRunnerIdentifier),
		TaskDefinition: awssdk.String(arn),
	})

	if err != nil {
		return nil, err
	}

	if len(out.Failures) > 0 {
		return nil, fmt.Errorf("couldn't start task: %s", awssdk.StringValue(out.Failures[0].Reason))
	}
	if len(out.Tasks) == 0 {
		return nil, fmt.Errorf("error: task status unknown")
	}
	return out.Tasks[0], nil
}

func (tds *ECSTaskDefinitionStore) GetRunningTasksByFamily(taskDefinitionFamily string) ([]string, error) {
	taskARNs := []string{}

	tasks, err := tds.ecsMetadata.GetTasks("marmoreal", taskDefinitionFamily, ecs.DesiredStatusRunning)
	if err != nil {
		return taskARNs, err
	}

	for _, task := range tasks {
		taskARNs = append(taskARNs, awssdk.StringValue(task.TaskArn))
	}

	return taskARNs, nil
}
