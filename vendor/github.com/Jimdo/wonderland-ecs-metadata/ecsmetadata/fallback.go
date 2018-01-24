package ecsmetadata

import (
	"github.com/afex/hystrix-go/hystrix"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/sirupsen/logrus"
)

const (
	getContainerInstanceHystrixCommand  = "ecsmetadata.GetContainerInstance"
	getContainerInstancesHystrixCommand = "ecsmetadata.GetContainerInstances"
	getServiceHystrixCommand            = "ecsmetadata.GetService"
	getServicesHystrixCommand           = "ecsmetadata.GetServices"
	getTaskHystrixCommand               = "ecsmetadata.GetTask"
	getTasksHystrixCommand              = "ecsmetadata.GetTasks"
	getTaskDefinitionHystrixCommand     = "ecsmetadata.GetTaskDefinition"
)

func init() {
	commands := []string{
		getContainerInstanceHystrixCommand,
		getContainerInstancesHystrixCommand,
		getServiceHystrixCommand,
		getServicesHystrixCommand,
		getTaskHystrixCommand,
		getTasksHystrixCommand,
		getTaskDefinitionHystrixCommand,
	}
	for _, command := range commands {
		hystrix.ConfigureCommand(command, hystrix.CommandConfig{
			Timeout: int(ecsMetadataServiceMaxElapsedRetryTime.Seconds() * 1000),
		})
	}
}

type Fallback struct {
	Primary   Metadata
	Secondary Metadata
}

func (p *Fallback) GetContainerInstance(cluster, arn string) (*ecs.ContainerInstance, error) {
	var result *ecs.ContainerInstance
	err := hystrix.Do(
		getContainerInstanceHystrixCommand,
		func() error {
			instance, err := p.Primary.GetContainerInstance(cluster, arn)
			if err != nil {
				return err
			}
			result = instance
			return nil
		},
		func(error) error {
			instance, err := p.Secondary.GetContainerInstance(cluster, arn)
			if err != nil {
				return err
			}
			result = instance
			return nil
		},
	)
	return result, err
}

func (p *Fallback) GetContainerInstances(cluster string) ([]*ecs.ContainerInstance, error) {
	var result []*ecs.ContainerInstance
	err := hystrix.Do(
		getContainerInstancesHystrixCommand,
		func() error {
			instances, err := p.Primary.GetContainerInstances(cluster)
			if err != nil {
				return err
			}
			result = instances
			return nil
		},
		func(error) error {
			instances, err := p.Secondary.GetContainerInstances(cluster)
			if err != nil {
				return err
			}
			result = instances
			return nil
		},
	)
	return result, err
}

func (p *Fallback) GetService(cluster, service string) (*ecs.Service, error) {
	return p.Secondary.GetService(cluster, service)
}

func (p *Fallback) GetServices(cluster string) ([]*ecs.Service, error) {
	return p.Secondary.GetServices(cluster)
}

func (p *Fallback) GetTask(cluster, arn string) (*ecs.Task, error) {
	var result *ecs.Task
	err := hystrix.Do(
		getTaskHystrixCommand,
		func() error {
			task, err := p.Primary.GetTask(cluster, arn)
			if err != nil {
				return err
			}
			result = task
			return nil
		},
		func(error) error {
			task, err := p.Secondary.GetTask(cluster, arn)
			if err != nil {
				return err
			}
			result = task
			return nil
		},
	)
	return result, err
}

func (p *Fallback) GetTaskDefinition(arn string) (*ecs.TaskDefinition, error) {
	return p.Secondary.GetTaskDefinition(arn)
}

func (p *Fallback) GetTasks(cluster, family, desiredStatus string) ([]*ecs.Task, error) {
	var result []*ecs.Task
	err := hystrix.Do(
		getTasksHystrixCommand,
		func() error {
			tasks, err := p.Primary.GetTasks(cluster, family, desiredStatus)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"cluster":        cluster,
					"family":         family,
					"desired_status": desiredStatus,
				}).Debugf("Primary GetTasks lookup failed")
				return err
			}
			result = tasks
			return nil
		},
		func(error) error {
			tasks, err := p.Secondary.GetTasks(cluster, family, desiredStatus)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"cluster":        cluster,
					"family":         family,
					"desired_status": desiredStatus,
				}).Debugf("Secondary GetTasks lookup failed")
				return err
			}
			result = tasks
			return nil
		},
	)
	return result, err
}
