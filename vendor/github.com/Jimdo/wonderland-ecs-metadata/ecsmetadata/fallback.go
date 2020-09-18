package ecsmetadata

import (
	"github.com/afex/hystrix-go/hystrix"
	"github.com/aws/aws-sdk-go/service/ecs"
	"runtime"
)

func init() {
	hystrix.DefaultTimeout = int(ecsMetadataServiceMaxElapsedRetryTime.Seconds() * 1000)
}

type Fallback struct {
	Primary   Metadata
	Secondary Metadata
}

func (p *Fallback) GetContainerInstance(cluster, arn string) (*ecs.ContainerInstance, error) {
	var result *ecs.ContainerInstance
	err := p.doHystrix(func(metadata Metadata) (err error) {
		result, err = metadata.GetContainerInstance(cluster, arn)
		return err
	})

	return result, err
}

func (p *Fallback) GetContainerInstances(cluster string) ([]*ecs.ContainerInstance, error) {
	var result []*ecs.ContainerInstance
	err := p.doHystrix(func(metadata Metadata) (err error) {
		result, err = metadata.GetContainerInstances(cluster)
		return err
	})

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
	err := p.doHystrix(func(metadata Metadata) (err error) {
		result, err = metadata.GetTask(cluster, arn)
		return err
	})

	return result, err
}

func (p *Fallback) GetTaskDefinition(arn string) (*ecs.TaskDefinition, error) {
	return p.Secondary.GetTaskDefinition(arn)
}

func (p *Fallback) GetTasks(cluster, family, desiredStatus string) ([]*ecs.Task, error) {
	var result []*ecs.Task
	err := p.doHystrix(func(metadata Metadata) (err error) {
		result, err = metadata.GetTasks(cluster, family, desiredStatus)
		return err
	})

	return result, err
}

func (p *Fallback) GetTasksByService(cluster, service, desiredStatus string) ([]*ecs.Task, error) {
	var result []*ecs.Task
	err := p.doHystrix(func(metadata Metadata) (err error) {
		result, err = metadata.GetTasksByService(cluster, service, desiredStatus)
		return err
	})

	return result, err
}

func (p *Fallback) doHystrix(fn func(Metadata) error) error {
	// callingFn is something like "ecsmetadata.GetTasks"
	callingFn := getCaller()
	return hystrix.Do(
		callingFn,
		func() error {
			return fn(p.Primary)
		},
		func(error) error {
			return fn(p.Secondary)
		},
	)
}

// getCaller returns the calling function of the function that calls this function.
// e.g.
//	func alice() {
//		bob()
//	}
//	func bob() {
//		getCaller() // returns alice
//	}
func getCaller() string {
	pc, _, _, _ := runtime.Caller(2)
	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()
	return frame.Function
}
