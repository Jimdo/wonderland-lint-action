package aws

import (
	"fmt"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents/cloudwatcheventsiface"
)

type RuleCronManager interface {
	RunTaskDefinitionWithSchedule(string, string, string) error
	DeleteRule(string) error
}

type CloudwatchRuleCronManager struct {
	cloudwatchEvents cloudwatcheventsiface.CloudWatchEventsAPI
	cronRoleARN      string
	ecsClusterARN    string
}

func NewCloudwatchRuleCronManager(ce cloudwatcheventsiface.CloudWatchEventsAPI, clusterARN, roleARN string) *CloudwatchRuleCronManager {
	return &CloudwatchRuleCronManager{
		cloudwatchEvents: ce,
		cronRoleARN:      roleARN,
		ecsClusterARN:    clusterARN,
	}
}

func (cm *CloudwatchRuleCronManager) RunTaskDefinitionWithSchedule(ruleName, taskDefinitionARN, schedule string) error {
	_, err := cm.cloudwatchEvents.PutRule(&cloudwatchevents.PutRuleInput{
		Description:        awssdk.String("Foobar"),
		Name:               awssdk.String(ruleName),
		State:              awssdk.String(cloudwatchevents.RuleStateEnabled),
		ScheduleExpression: awssdk.String(fmt.Sprintf("cron(%s *)", schedule)),
	})
	if err != nil {
		return fmt.Errorf("could not put cloudwatch rule %q with error: %s", ruleName, err)
	}

	_, err = cm.cloudwatchEvents.PutTargets(&cloudwatchevents.PutTargetsInput{
		Rule: awssdk.String(ruleName),
		Targets: []*cloudwatchevents.Target{
			{
				Arn:     awssdk.String(cm.ecsClusterARN),
				Id:      awssdk.String(ruleName),
				RoleArn: awssdk.String(cm.cronRoleARN),
				EcsParameters: &cloudwatchevents.EcsParameters{
					TaskDefinitionArn: awssdk.String(taskDefinitionARN),
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("could not put target for cloudwatch rule %q with error: %s", ruleName, err)
	}

	return nil
}

func (cm *CloudwatchRuleCronManager) DeleteRule(ruleName string) error {
	out, err := cm.cloudwatchEvents.ListTargetsByRule(&cloudwatchevents.ListTargetsByRuleInput{
		Rule: awssdk.String(ruleName),
	})
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			if err.Code() != cloudwatchevents.ErrCodeResourceNotFoundException {
				return fmt.Errorf("could not list targets of rule %q with error: %s", ruleName, err)
			}
			// Return early, because the other resources cannot exist without a rule
			return nil
		} else {
			return fmt.Errorf("could not list targets of rule %q with error: %s", ruleName, err)
		}
	}

	var ids []*string
	for _, target := range out.Targets {
		ids = append(ids, target.Id)
	}

	_, err = cm.cloudwatchEvents.RemoveTargets(&cloudwatchevents.RemoveTargetsInput{
		Rule: awssdk.String(ruleName),
		Ids:  ids,
	})
	if err != nil {
		return fmt.Errorf("could not remove targets %q of rule %q with error: %s", ids, ruleName, err)
	}

	_, err = cm.cloudwatchEvents.DeleteRule(&cloudwatchevents.DeleteRuleInput{
		Name: awssdk.String(ruleName),
	})
	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			if err.Code() != cloudwatchevents.ErrCodeResourceNotFoundException {
				return fmt.Errorf("could not delete rule %q with error: %s", ruleName, err)
			}
		} else {
			return fmt.Errorf("could not delete rule %q with error: %s", ruleName, err)
		}
	}

	return nil
}
