package aws

import (
	"fmt"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/sirupsen/logrus"

	"github.com/Jimdo/wonderland-crons/cron"
)

type TaskDefinitionStore interface {
	AddRevisionFromCronDescription(cronName, family string, desc *cron.CronDescription) (string, error)
	DeleteByFamily(family string) error
}

type ECSTaskDefinitionStore struct {
	ecs ecsiface.ECSAPI
	tdm *ECSTaskDefinitionMapper
}

func NewECSTaskDefinitionStore(e ecsiface.ECSAPI, tdm *ECSTaskDefinitionMapper) *ECSTaskDefinitionStore {
	return &ECSTaskDefinitionStore{
		ecs: e,
		tdm: tdm,
	}
}

func (tds *ECSTaskDefinitionStore) AddRevisionFromCronDescription(cronName, family string, desc *cron.CronDescription) (string, error) {
	rtdInput := &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{tds.tdm.ContainerDefinitionFromCronDescription(family, desc, cronName)},
		Family:               awssdk.String(family),
	}

	// add timeout sidecar container
	timeoutCmd := fmt.Sprintf("sleep %d; exit %d", desc.Timeout, cron.TimeoutExitCode)
	timeoutSidecarDefinition := &ecs.ContainerDefinition{
		Command: awssdk.StringSlice([]string{"/bin/sh", "-c", timeoutCmd}),
		Cpu:     awssdk.Int64(int64(16)),
		DockerLabels: map[string]*string{
			"com.jimdo.wonderland.cron": awssdk.String(cronName),
		},
		Image:  awssdk.String("alpine:3.6"),
		Memory: awssdk.Int64(int64(32)),
		Name:   awssdk.String("timeout"),
	}

	rtdInput.ContainerDefinitions = append(rtdInput.ContainerDefinitions, timeoutSidecarDefinition)
	out, err := tds.ecs.RegisterTaskDefinition(rtdInput)
	if err != nil {
		return "", fmt.Errorf("could not register task definition for family %q with error: %s", family, err)
	}
	return awssdk.StringValue(out.TaskDefinition.TaskDefinitionArn), nil
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
