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
	tdm *ECSTaskDefinitionMapper
}

func NewECSTaskDefinitionStore(e ecsiface.ECSAPI, tdm *ECSTaskDefinitionMapper) *ECSTaskDefinitionStore {
	return &ECSTaskDefinitionStore{
		ecs: e,
		tdm: tdm,
	}
}

func (tds *ECSTaskDefinitionStore) AddRevisionFromCronDescription(family string, cron *cron.CronDescription) (string, error) {
	out, err := tds.ecs.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{tds.tdm.ContainerDefinitionFromCronDescription(family, cron)},
		Family:               awssdk.String(family),
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
