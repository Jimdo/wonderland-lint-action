package ecsmetadata

import (
	"errors"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/sirupsen/logrus"
)

type state struct {
	instances map[string]*ecs.ContainerInstance

	familyByTaskArn map[string]string
	tasksByFamily   map[string](map[string]*ecs.Task)
}

func newState() *state {
	return &state{
		instances: make(map[string]*ecs.ContainerInstance),

		familyByTaskArn: make(map[string]string),
		tasksByFamily:   make(map[string](map[string]*ecs.Task)),
	}
}

type StatefulInMemory map[string]*state

func NewStatefulInMemory() StatefulInMemory {
	return StatefulInMemory(make(map[string]*state))
}

func (s StatefulInMemory) isInitialized(cluster string) (bool, error) {
	cs, ok := s[cluster]
	if !ok {
		return false, nil
	}
	return cs.instances != nil &&
		cs.tasksByFamily != nil &&
		cs.familyByTaskArn != nil, nil
}

func (s StatefulInMemory) initializeCluster(cluster string) {
	s[cluster] = newState()
}

func (s StatefulInMemory) GetContainerInstance(cluster, arn string) (*ecs.ContainerInstance, error) {
	if err := s.mustBeInitialized(cluster); err != nil {
		return nil, err
	}
	return s[cluster].instances[arn], nil
}

func (s StatefulInMemory) GetContainerInstances(cluster string) ([]*ecs.ContainerInstance, error) {
	if err := s.mustBeInitialized(cluster); err != nil {
		return nil, err
	}

	var instances []*ecs.ContainerInstance
	for _, instance := range s[cluster].instances {
		if aws.StringValue(instance.Status) == "ACTIVE" || aws.StringValue(instance.Status) == "DRAINING" {
			instances = append(instances, instance)
		}
	}
	sort.Slice(instances, func(i, j int) bool {
		return aws.StringValue(instances[i].ContainerInstanceArn) < aws.StringValue(instances[j].ContainerInstanceArn)
	})

	return instances, nil
}

func (s StatefulInMemory) GetService(cluster, service string) (*ecs.Service, error) {
	return nil, errors.New("Not implemented yet")
}

func (s StatefulInMemory) GetServices(cluster string) ([]*ecs.Service, error) {
	return nil, errors.New("Not implemented yet")
}

func (s StatefulInMemory) GetTask(cluster, arn string) (*ecs.Task, error) {
	if err := s.mustBeInitialized(cluster); err != nil {
		return nil, err
	}

	family := s[cluster].familyByTaskArn[arn]
	return s[cluster].tasksByFamily[family][arn], nil
}

func (s StatefulInMemory) GetTaskDefinition(arn string) (*ecs.TaskDefinition, error) {
	return nil, errors.New("Not implemented yet")
}

func (s StatefulInMemory) GetTasks(cluster, family, status string) ([]*ecs.Task, error) {
	if err := s.mustBeInitialized(cluster); err != nil {
		return nil, err
	}

	tasks := []*ecs.Task{}

	if family == "" {
		for _, tasksByFamily := range s[cluster].tasksByFamily {
			for _, task := range tasksByFamily {
				if status == "" || aws.StringValue(task.DesiredStatus) == status {
					tasks = append(tasks, task)
				}
			}
		}
	} else {
		for _, task := range s[cluster].tasksByFamily[family] {
			if aws.StringValue(task.DesiredStatus) == status {
				tasks = append(tasks, task)
			}
		}
	}
	return tasks, nil
}

func (s StatefulInMemory) GetTasksByService(cluster, service, desiredStatus string) ([]*ecs.Task, error) {
	return nil, errors.New("not implemented")
}

func (s StatefulInMemory) UpdateContainerInstance(cluster string, instance *ecs.ContainerInstance) error {
	err := s.mustBeInitialized(cluster)
	if err != nil {
		return fmt.Errorf("State for cluster could not be ensured: %s", err)
	}

	arn := aws.StringValue(instance.ContainerInstanceArn)
	if oldInstance, ok := s[cluster].instances[arn]; !ok || aws.Int64Value(oldInstance.Version) < aws.Int64Value(instance.Version) {
		logrus.Debugf("updating container instance %s", arn)
		s[cluster].instances[arn] = instance
	}
	return nil
}

func (s StatefulInMemory) UpdateContainerInstances(cluster string, instances []*ecs.ContainerInstance) error {
	_, ok := s[cluster]
	if !ok {
		s[cluster] = newState()
	}
	for _, i := range instances {
		if err := s.UpdateContainerInstance(cluster, i); err != nil {
			return err
		}
	}
	return nil
}

func (s StatefulInMemory) RemoveContainerInstance(cluster string, instance *ecs.ContainerInstance) error {
	arn := aws.StringValue(instance.ContainerInstanceArn)
	delete(s[cluster].instances, arn)

	return nil
}

func (s StatefulInMemory) UpdateTask(cluster string, task *ecs.Task) error {
	s.mustBeInitialized(cluster)
	family, err := getFamilyFromECSTask(task)
	if err != nil {
		return err
	}

	arn := aws.StringValue(task.TaskArn)
	if s[cluster].familyByTaskArn == nil {
		s[cluster].familyByTaskArn = map[string]string{}
	}
	s[cluster].familyByTaskArn[arn] = family

	if s[cluster].tasksByFamily == nil {
		s[cluster].tasksByFamily = map[string]map[string]*ecs.Task{}
	}

	if _, ok := s[cluster].tasksByFamily[family]; !ok {
		s[cluster].tasksByFamily[family] = map[string]*ecs.Task{}
	}
	s[cluster].tasksByFamily[family][arn] = task

	return nil
}

func (s StatefulInMemory) UpdateTasks(cluster string, tasks []*ecs.Task) error {
	s[cluster].familyByTaskArn = map[string]string{}
	s[cluster].tasksByFamily = map[string]map[string]*ecs.Task{}

	for _, task := range tasks {
		if err := s.UpdateTask(cluster, task); err != nil {
			return fmt.Errorf("Updating task %q failed: %s", aws.StringValue(task.TaskArn), err)
		}
	}

	return nil
}

func (s StatefulInMemory) RemoveTask(cluster string, task *ecs.Task) error {
	arn := aws.StringValue(task.TaskArn)
	family := s[cluster].familyByTaskArn[arn]
	delete(s[cluster].familyByTaskArn, arn)
	delete(s[cluster].tasksByFamily[family], arn)
	if len(s[cluster].tasksByFamily[family]) == 0 {
		delete(s[cluster].tasksByFamily, family)
	}
	return nil
}

func (s StatefulInMemory) mustBeInitialized(cluster string) error {
	initialized, err := s.isInitialized(cluster)
	if err != nil {
		return fmt.Errorf("could not check whether store is initialized: %s", err)
	}
	if !initialized {
		s.initializeCluster(cluster)
	}
	return nil
}
