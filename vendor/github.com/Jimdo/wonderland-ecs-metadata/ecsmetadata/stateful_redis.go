package ecsmetadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/garyburd/redigo/redis"
	"github.com/sirupsen/logrus"

	"github.com/Jimdo/wonderland-ecs-metadata/logger"
)

const (
	ContainerInstancesKey = "container-instances"
)

type StatefulRedis struct {
	RedisPool          *redis.Pool
	TasksByFamilyMap   *TasksByFamilyMap
	FamilyByTaskArnMap *FamilyByTaskArnMap
}

func NewStatefulRedis(pool *redis.Pool) *StatefulRedis {
	return &StatefulRedis{
		RedisPool: pool,
		TasksByFamilyMap: &TasksByFamilyMap{
			RedisPool: pool,
		},
		FamilyByTaskArnMap: &FamilyByTaskArnMap{
			RedisPool: pool,
		},
	}
}

func (s *StatefulRedis) GetContainerInstance(cluster, arn string) (*ecs.ContainerInstance, error) {
	key := buildKeyWithClusterName(cluster, ContainerInstancesKey)
	if err := s.mustBeInitialized(key); err != nil {
		return nil, err
	}

	return s.getContainerInstance(cluster, arn)
}

func (s *StatefulRedis) getContainerInstance(cluster, arn string) (*ecs.ContainerInstance, error) {
	c := s.RedisPool.Get()
	defer c.Close()

	key := buildKeyWithClusterName(cluster, ContainerInstancesKey)
	jsonData, err := redis.Bytes(c.Do("HGET", key, arn))
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil
		}
		return nil, fmt.Errorf("could not load container instance from Redis: %s", err)
	}

	instance := &ecs.ContainerInstance{}
	if err := json.Unmarshal(jsonData, &instance); err != nil {
		return nil, fmt.Errorf("could not decode JSON container instance: %s", err)
	}

	return instance, nil
}

func (s *StatefulRedis) GetContainerInstances(cluster string) ([]*ecs.ContainerInstance, error) {
	key := buildKeyWithClusterName(cluster, ContainerInstancesKey)
	if err := s.mustBeInitialized(key); err != nil {
		return nil, err
	}

	c := s.RedisPool.Get()
	defer c.Close()

	instancesJSONMap, err := redis.StringMap(c.Do("HGETALL", key))
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil
		}
		return nil, fmt.Errorf("could not load container instance from Redis: %s", err)
	}

	instances := []*ecs.ContainerInstance{}
	for _, instanceJSON := range instancesJSONMap {
		instance := &ecs.ContainerInstance{}
		if err := json.Unmarshal([]byte(instanceJSON), &instance); err != nil {
			return nil, fmt.Errorf("could not decode JSON container instance: %s", err)
		}
		if aws.StringValue(instance.Status) == "ACTIVE" || aws.StringValue(instance.Status) == "DRAINING" {
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

func (s *StatefulRedis) GetService(cluster, service string) (*ecs.Service, error) {
	return nil, errors.New("Not implemented yet")
}

func (s *StatefulRedis) GetServices(cluster string) ([]*ecs.Service, error) {
	return nil, errors.New("Not implemented yet")
}

func (s *StatefulRedis) GetTask(cluster, arn string) (*ecs.Task, error) {
	family, err := s.FamilyByTaskArnMap.Get(cluster, arn)
	if err != nil {
		return nil, fmt.Errorf("Fetching family for arn %s failed: %s", arn, err)
	}

	key := buildKeyWithClusterName(cluster, TasksByFamilyKey, family)
	if err := s.mustBeInitialized(key); err != nil {
		return nil, err
	}

	return s.TasksByFamilyMap.Get(cluster, family, arn)
}

func (s *StatefulRedis) GetTaskDefinition(arn string) (*ecs.TaskDefinition, error) {
	return nil, errors.New("Not implemented yet")
}

// listFamilies returns all task families for the given cluster
func (s *StatefulRedis) listFamilies(cluster string) ([]string, error) {
	var (
		families     []string
		keys         []string
		cursor       = 0
		matchPattern = buildKeyWithClusterName(cluster, TasksByFamilyKey, "*")
	)

	c := s.RedisPool.Get()
	defer c.Close()

	for {
		reply, err := redis.Values(c.Do("SCAN", cursor, "MATCH", matchPattern))
		if err != nil {
			return nil, fmt.Errorf("could not scan for tasks (matchPattern): %s", err)
		}
		if _, err := redis.Scan(reply, &cursor, &keys); err != nil {
			fmt.Printf("could not scan reply: %s", err)
		}
		for _, key := range keys {
			families = append(families, strings.TrimPrefix(key, buildKeyWithClusterName(cluster, TasksByFamilyKey)+"."))
		}
		if cursor == 0 {
			break
		}
	}
	return families, nil

}

func (s *StatefulRedis) GetTasks(cluster, family, desiredStatus string) ([]*ecs.Task, error) {
	var families []string
	var err error

	if family == "" {
		families, err = s.listFamilies(cluster)
		if err != nil {
			return nil, fmt.Errorf("Getting all tasks-by-family keys for cluster %s failed: %s", cluster, err)
		}
	} else {
		families = append(families, family)
	}

	logrus.WithFields(logrus.Fields{
		"cluster":  cluster,
		"families": families,
	}).Debugf("Got families")

	tasks := []*ecs.Task{}
	for _, family := range families {
		logrus.WithFields(logrus.Fields{
			"cluster":        cluster,
			"desired_status": desiredStatus,
			"family":         family,
		}).Debugf("Getting tasks by family")

		tasksByFamily, err := s.TasksByFamilyMap.GetAll(cluster, family)
		if err != nil {
			return nil, fmt.Errorf("could not load tasks from Redis: %s", err)
		}

		for _, task := range tasksByFamily {
			if desiredStatus == "" || aws.StringValue(task.DesiredStatus) == desiredStatus {
				logger.Task(cluster, task).Debugf("Task with matched status found")
				tasks = append(tasks, task)
			}

			logger.Task(cluster, task).WithFields(logrus.Fields{
				"family":      family,
				"tasks_count": len(tasks),
			}).Debugf("Task found for Family")
		}
	}

	return tasks, nil
}

func (s *StatefulRedis) UpdateContainerInstance(cluster string, i *ecs.ContainerInstance) error {
	c := s.RedisPool.Get()
	defer c.Close()

	key := buildKeyWithClusterName(cluster, ContainerInstancesKey)

	arn := aws.StringValue(i.ContainerInstanceArn)
	old, err := s.getContainerInstance(cluster, arn)
	if err != nil {
		return fmt.Errorf("could not fetch existing container instance: %s", err)
	}

	if old == nil || aws.Int64Value(old.Version) < aws.Int64Value(i.Version) {
		logrus.WithFields(logrus.Fields{
			"cluster":                cluster,
			"container_instance_arn": arn,
		}).Debug("updating container instance")
		jsonInstance, err := json.Marshal(i)
		if err != nil {
			return fmt.Errorf("could not JSON encode instance %q: %s", arn, err)
		}

		if _, err = c.Do("HSET", key, arn, jsonInstance); err != nil {
			return fmt.Errorf("could not set instance %q: %s", arn, err)
		}
	}
	return nil
}

func (s *StatefulRedis) UpdateContainerInstances(cluster string, instances []*ecs.ContainerInstance) error {
	for _, i := range instances {
		if err := s.UpdateContainerInstance(cluster, i); err != nil {
			return err
		}
	}
	return nil
}

func (s *StatefulRedis) RemoveContainerInstance(cluster string, i *ecs.ContainerInstance) error {
	c := s.RedisPool.Get()
	defer c.Close()

	arn := aws.StringValue(i.ContainerInstanceArn)
	logrus.WithFields(logrus.Fields{
		"cluster":                cluster,
		"container_instance_arn": arn,
	}).Debug("deleting container instance")

	key := buildKeyWithClusterName(cluster, ContainerInstancesKey)
	if _, err := c.Do("HDEL", key, arn); err != nil {
		return fmt.Errorf("could not delete instance %q: %s", arn, err)
	}
	return nil
}

func (s *StatefulRedis) UpdateTask(cluster string, t *ecs.Task) error {
	c := s.RedisPool.Get()
	defer c.Close()

	arn := aws.StringValue(t.TaskArn)

	family, err := getFamilyFromECSTask(t)
	if err != nil {
		return fmt.Errorf("Could not get family for task %q: %s", t, err)
	}

	old, err := s.TasksByFamilyMap.Get(cluster, family, arn)
	if err != nil {
		return fmt.Errorf("could not fetch existing task: %s", err)
	}

	if old == nil || aws.Int64Value(old.Version) < aws.Int64Value(t.Version) {
		logger.Task(cluster, t).WithFields(logrus.Fields{
			"family": family,
		}).Debug("Updating task")

		if err := s.TasksByFamilyMap.Set(cluster, family, arn, t); err != nil {
			return fmt.Errorf("Setting task failed: %s", err)
		}

		familyByTaskMap := &FamilyByTaskArnMap{
			RedisPool: s.RedisPool,
		}
		// write arn to family matching
		if err = familyByTaskMap.Set(cluster, arn, family); err != nil {
			return fmt.Errorf("could not set family %q for task %q: %s", family, arn, err)
		}
	}
	return nil
}

func (s *StatefulRedis) UpdateTasks(cluster string, t []*ecs.Task) error {
	for _, task := range t {
		if err := s.UpdateTask(cluster, task); err != nil {
			return err
		}
	}
	return nil
}

func (s *StatefulRedis) RemoveTask(cluster string, t *ecs.Task) error {
	family, err := getFamilyFromECSTask(t)
	if err != nil {
		return fmt.Errorf("Could not get family for task %q: %s", t, err)
	}

	c := s.RedisPool.Get()
	defer c.Close()

	arn := aws.StringValue(t.TaskArn)

	logger.Task(cluster, t).WithFields(logrus.Fields{
		"family": family,
	}).Debugf("Removing Task")

	s.TasksByFamilyMap.Del(cluster, family, arn)

	if err := s.FamilyByTaskArnMap.Del(cluster, arn); err != nil {
		return fmt.Errorf("RemoveTask: Deleting arn %q failed: %s", arn, err)
	}
	return nil
}

func (s *StatefulRedis) mustBeInitialized(redisKey string) error {
	initialized, err := s.isInitialized(redisKey)
	if err != nil {
		return fmt.Errorf("could not check whether store %q is initialized: %s", redisKey, err)
	}
	if !initialized {
		return errors.New("store is not initialized")
	}
	return nil
}

func (s *StatefulRedis) isInitialized(redisKey string) (bool, error) {
	c := s.RedisPool.Get()
	defer c.Close()

	exists, err := redis.Bool(c.Do("EXISTS", redisKey))
	if err != nil {
		return false, fmt.Errorf("error testing existence of redis key %q: %s", redisKey, err)
	}
	return exists, nil
}

func buildKeyWithClusterName(cluster string, keys ...string) string {
	joinedKeys := strings.Join(keys, ".")
	return fmt.Sprintf("%s.%s", cluster, joinedKeys)
}
