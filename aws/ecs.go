package aws

import (
	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"

	"fmt"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/sirupsen/logrus"
)

type ECSTaskDefinitionStore struct {
	ecs ecsiface.ECSAPI
}

func NewECSTaskDefinitionStore(e ecsiface.ECSAPI) *ECSTaskDefinitionStore {
	return &ECSTaskDefinitionStore{
		ecs: e,
	}
}

func (tds *ECSTaskDefinitionStore) AddRevisionFromCronDescription(family string, cron *cron.CronDescription) (string, error) {
	var envVars []*ecs.KeyValuePair
	for key, value := range cron.Description.Environment {
		envVars = append(envVars, &ecs.KeyValuePair{
			Name:  awssdk.String(key),
			Value: awssdk.String(value),
		})
	}

	out, err := tds.ecs.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{
			{
				Command: awssdk.StringSlice(cron.Description.Arguments),
				Cpu:     awssdk.Int64(int64(cron.Description.Capacity.CPULimit())),
				DockerLabels: map[string]*string{
					"com.jimdo.wonderland.cron": awssdk.String(cron.Name),
				},
				Environment: envVars,
				Image:       awssdk.String(cron.Description.Image),
				Memory:      awssdk.Int64(int64(cron.Description.Capacity.MemoryLimit())),
				Name:        awssdk.String(family),
			},
		},
		Family: awssdk.String(family),
	})
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
				logrus.
					WithField("ARN", awssdk.StringValue(arn)).
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
