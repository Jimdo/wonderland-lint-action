package aws

import (
	"fmt"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents/cloudwatcheventsiface"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/sirupsen/logrus"

	"github.com/Jimdo/wonderland-crons/cron"
	"github.com/Jimdo/wonderland-crons/validation"
	"github.com/aws/aws-sdk-go/aws/awserr"
)

type Service struct {
	cloudwatchEvents cloudwatcheventsiface.CloudWatchEventsAPI
	cronRoleARN      string
	ecs              ecsiface.ECSAPI
	ecsClusterARN    string
	validator        *validation.Validator
}

func NewService(v *validation.Validator, ce cloudwatcheventsiface.CloudWatchEventsAPI, e ecsiface.ECSAPI, clusterARN, roleARN string) *Service {
	return &Service{
		cloudwatchEvents: ce,
		cronRoleARN:      roleARN,
		ecs:              e,
		ecsClusterARN:    clusterARN,
		validator:        v,
	}
}

func (s *Service) Create(cron *cron.CronDescription) error {
	if err := s.validator.ValidateCronDescription(cron); err != nil {
		return err
	}

	baseName := "cron--" + cron.Name

	var envs []*ecs.KeyValuePair
	for key, value := range cron.Description.Environment {
		envs = append(envs, &ecs.KeyValuePair{
			Name:  awssdk.String(key),
			Value: awssdk.String(value),
		})
	}

	def, err := s.ecs.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{
			{
				Command: awssdk.StringSlice(cron.Description.Arguments),
				Cpu:     awssdk.Int64(int64(cron.Description.Capacity.CPULimit())),
				DockerLabels: map[string]*string{
					"com.jimdo.wonderland.cron": awssdk.String(cron.Name),
				},
				Environment: envs,
				Image:       awssdk.String(cron.Description.Image),
				Memory:      awssdk.Int64(int64(cron.Description.Capacity.MemoryLimit())),
				Name:        awssdk.String(baseName),
			},
		},
		Family: awssdk.String(baseName),
	})
	if err != nil {
		return err
	}

	logrus.WithField("schedule", fmt.Sprintf("cron(%s *)", cron.Schedule)).
		Info("Putting CloudwatchEventsRule")
	// TODO: Change the format generation
	_, err = s.cloudwatchEvents.PutRule(&cloudwatchevents.PutRuleInput{
		Description:        awssdk.String("Foobar"),
		Name:               awssdk.String(baseName),
		State:              awssdk.String(cloudwatchevents.RuleStateEnabled),
		ScheduleExpression: awssdk.String(fmt.Sprintf("cron(%s *)", cron.Schedule)),
	})

	if err != nil {
		return err
	}

	_, err = s.cloudwatchEvents.PutTargets(&cloudwatchevents.PutTargetsInput{
		Rule: awssdk.String(baseName),
		Targets: []*cloudwatchevents.Target{
			{
				Arn:     awssdk.String("arn:aws:ecs:eu-west-1:062052581233:cluster/crims"),
				Id:      awssdk.String(baseName),
				RoleArn: awssdk.String("arn:aws:iam::062052581233:role/ecsEventsRole"),
				EcsParameters: &cloudwatchevents.EcsParameters{
					TaskDefinitionArn: def.TaskDefinition.TaskDefinitionArn,
				},
			},
		},
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *Service) Delete(cronName string) error {
	baseName := "cron--" + cronName

	ruleExists := true
	out, err := s.cloudwatchEvents.ListTargetsByRule(&cloudwatchevents.ListTargetsByRuleInput{
		Rule: awssdk.String(baseName),
	})
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			if err.Code() != cloudwatchevents.ErrCodeResourceNotFoundException {
				return err
			}
			ruleExists = false
		} else {
			return err
		}
	}

	if ruleExists {
		var ids []*string
		for _, target := range out.Targets {
			ids = append(ids, target.Id)
		}

		_, err = s.cloudwatchEvents.RemoveTargets(&cloudwatchevents.RemoveTargetsInput{
			Rule: awssdk.String(baseName),
			Ids:  ids,
		})
		if err != nil {
			return err
		}

		_, err = s.cloudwatchEvents.DeleteRule(&cloudwatchevents.DeleteRuleInput{
			Name: awssdk.String(baseName),
		})
		if err != nil {
			if err, ok := err.(awserr.Error); ok {
				if err.Code() != cloudwatchevents.ErrCodeResourceNotFoundException {
					return err
				}
			} else {
				return err
			}
		}
	}

	err = s.ecs.ListTaskDefinitionsPages(&ecs.ListTaskDefinitionsInput{
		FamilyPrefix: awssdk.String(baseName),
	}, func(out *ecs.ListTaskDefinitionsOutput, last bool) bool {
		for _, arn := range out.TaskDefinitionArns {
			_, err := s.ecs.DeregisterTaskDefinition(&ecs.DeregisterTaskDefinitionInput{
				TaskDefinition: arn,
			})
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
		return err
	}

	return nil
}
