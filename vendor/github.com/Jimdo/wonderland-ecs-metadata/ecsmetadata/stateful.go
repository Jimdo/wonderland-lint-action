package ecsmetadata

import "github.com/aws/aws-sdk-go/service/ecs"

type StatefulMetadata interface {
	Metadata

	UpdateContainerInstance(cluster string, instance *ecs.ContainerInstance) error
	UpdateContainerInstances(cluster string, instances []*ecs.ContainerInstance) error
	RemoveContainerInstance(cluster string, instances *ecs.ContainerInstance) error

	UpdateTask(cluster string, task *ecs.Task) error
	UpdateTasks(cluster string, tasks []*ecs.Task) error
	RemoveTask(cluster string, task *ecs.Task) error
}
