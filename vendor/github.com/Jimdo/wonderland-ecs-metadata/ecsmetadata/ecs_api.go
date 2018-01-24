package ecsmetadata

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
)

type ECSAPI struct {
	ECS ecsiface.ECSAPI
}

func (a *ECSAPI) GetContainerInstance(cluster, arn string) (*ecs.ContainerInstance, error) {
	out, err := a.ECS.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(cluster),
		ContainerInstances: aws.StringSlice([]string{arn}),
	})
	if err != nil {
		return nil, err
	}
	if len(out.ContainerInstances) == 0 {
		return nil, nil
	}
	return out.ContainerInstances[0], nil
}

func (a *ECSAPI) GetContainerInstances(cluster string) ([]*ecs.ContainerInstance, error) {
	var instances []*ecs.ContainerInstance
	var describeError error

	err := a.ECS.ListContainerInstancesPages(&ecs.ListContainerInstancesInput{
		Cluster: aws.String(cluster),
	}, func(listOut *ecs.ListContainerInstancesOutput, last bool) bool {
		if len(listOut.ContainerInstanceArns) == 0 {
			return false
		}
		out, err := a.ECS.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(cluster),
			ContainerInstances: listOut.ContainerInstanceArns,
		})
		if err != nil {
			describeError = err
			return false
		}
		instances = append(instances, out.ContainerInstances...)
		return !last
	})
	if err != nil {
		return nil, err
	}
	if describeError != nil {
		return nil, describeError
	}

	return instances, nil
}

func (a *ECSAPI) GetService(cluster, name string) (*ecs.Service, error) {
	out, err := a.ECS.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: aws.StringSlice([]string{name}),
	})
	if err != nil {
		return nil, err
	}
	if len(out.Services) == 0 {
		return nil, nil
	}
	return out.Services[0], nil
}

func (a *ECSAPI) GetServices(cluster string) ([]*ecs.Service, error) {
	var services []*ecs.Service
	var describeError error

	err := a.ECS.ListServicesPages(&ecs.ListServicesInput{
		Cluster: aws.String(cluster),
	}, func(listOut *ecs.ListServicesOutput, last bool) bool {
		if len(listOut.ServiceArns) == 0 {
			return false
		}
		out, err := a.ECS.DescribeServices(&ecs.DescribeServicesInput{
			Cluster:  aws.String(cluster),
			Services: listOut.ServiceArns,
		})
		if err != nil {
			describeError = err
			return false
		}

		services = append(services, out.Services...)
		return !last
	})
	if err != nil {
		return nil, err
	}
	if describeError != nil {
		return nil, describeError
	}

	return services, nil
}

func (a *ECSAPI) GetTask(cluster, arn string) (*ecs.Task, error) {
	out, err := a.ECS.DescribeTasks(&ecs.DescribeTasksInput{
		Cluster: aws.String(cluster),
		Tasks:   aws.StringSlice([]string{arn}),
	})
	if err != nil {
		return nil, err
	}
	if len(out.Failures) != 0 {
		return nil, fmt.Errorf("Error looking up task status: %s", aws.StringValue(out.Failures[0].Reason))
	}
	if len(out.Tasks) == 0 {
		return nil, nil
	}
	return out.Tasks[0], nil
}

func (a *ECSAPI) GetTaskDefinition(arn string) (*ecs.TaskDefinition, error) {
	out, err := a.ECS.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(arn),
	})
	if err != nil {
		return nil, err
	}
	return out.TaskDefinition, nil
}

func (a *ECSAPI) GetTasks(cluster, family, status string) ([]*ecs.Task, error) {
	var tasks []*ecs.Task
	var describeError error

	input := &ecs.ListTasksInput{
		Cluster:       aws.String(cluster),
		DesiredStatus: aws.String(status),
	}

	if family != "" {
		input.Family = aws.String(family)
	}

	err := a.ECS.ListTasksPages(input, func(listOut *ecs.ListTasksOutput, last bool) bool {
		if len(listOut.TaskArns) == 0 {
			return false
		}
		out, err := a.ECS.DescribeTasks(&ecs.DescribeTasksInput{
			Cluster: aws.String(cluster),
			Tasks:   listOut.TaskArns,
		})
		if err != nil {
			describeError = err
			return false
		}
		tasks = append(tasks, out.Tasks...)
		return !last
	})
	if err != nil {
		return nil, err
	}
	if describeError != nil {
		return nil, describeError
	}

	return tasks, nil
}
