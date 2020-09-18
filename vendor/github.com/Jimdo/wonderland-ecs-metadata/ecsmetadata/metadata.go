package ecsmetadata

import "github.com/aws/aws-sdk-go/service/ecs"

// The Metadata interface defines the requirements for ECS metadata providers. The primary metadata provider
// is always the ECS API, but other (caching) providers can be used to optimize ECS API usage.
type Metadata interface {
	// GetContainerInstance returns the container instance with the given ARN
	GetContainerInstance(cluster, arn string) (*ecs.ContainerInstance, error)
	// GetContainerInstances returns a list of all container instances running in a cluster
	GetContainerInstances(cluster string) ([]*ecs.ContainerInstance, error)
	// GetService returns the service with the given name
	GetService(cluster, service string) (*ecs.Service, error)
	// GetServices returns a list of all services running in a cluster
	GetServices(cluster string) ([]*ecs.Service, error)
	// GetTask returns the task with the given ARN
	GetTask(cluster, arn string) (*ecs.Task, error)
	// GetTaskDefinition returns the task definition with the given ARN
	GetTaskDefinition(arn string) (*ecs.TaskDefinition, error)
	// GetTasks returns a list of all tasks of a family that are in the given status
	GetTasks(cluster, family, desiredStatus string) ([]*ecs.Task, error)
	// GetTasksByService returns a list of all tasks of a service that are in the given status
	GetTasksByService(cluster, service, desiredStatus string) ([]*ecs.Task, error)
}
