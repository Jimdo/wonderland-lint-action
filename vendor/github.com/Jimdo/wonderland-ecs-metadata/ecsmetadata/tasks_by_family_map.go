package ecsmetadata

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/garyburd/redigo/redis"
)

const (
	TasksByFamilyKey = "tasks-by-family"
)

type TasksByFamilyMap struct {
	RedisPool *redis.Pool
}

func (t *TasksByFamilyMap) Get(cluster, family, arn string) (*ecs.Task, error) {
	key := buildKeyWithClusterName(cluster, TasksByFamilyKey, family)

	c := t.RedisPool.Get()
	defer c.Close()

	jsonData, err := redis.Bytes(c.Do("HGET", key, arn))
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil
		}
		return nil, fmt.Errorf("could not load task from Redis: %s", err)
	}

	task := &ecs.Task{}
	if err := json.Unmarshal(jsonData, task); err != nil {
		return nil, fmt.Errorf("could not decode JSON task: %s", err)
	}

	return task, nil
}

func (t *TasksByFamilyMap) GetAll(cluster, family string) ([]*ecs.Task, error) {
	key := buildKeyWithClusterName(cluster, TasksByFamilyKey, family)
	c := t.RedisPool.Get()
	defer c.Close()

	tasksMap, err := redis.StringMap(c.Do("HGETALL", key))
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil
		}
		return nil, fmt.Errorf("could not load tasks from Redis: %s", err)
	}
	tasks := []*ecs.Task{}
	for arn, taskJSON := range tasksMap {
		task := &ecs.Task{}
		if err := json.Unmarshal([]byte(taskJSON), task); err != nil {
			return nil, fmt.Errorf("Could not unmarshal task %q for family %q: %s", arn, family, err)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (t *TasksByFamilyMap) Set(cluster, family, arn string, task *ecs.Task) error {
	key := buildKeyWithClusterName(cluster, TasksByFamilyKey, family)
	jsonTask, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("could not JSON encode task %q: %s", arn, err)
	}

	c := t.RedisPool.Get()
	defer c.Close()

	if _, err = c.Do("HSET", key, arn, jsonTask); err != nil {
		return fmt.Errorf("could not set task %q for key %q: %s", arn, key, err)
	}
	return nil
}

func (t *TasksByFamilyMap) Del(cluster, family, arn string) error {
	fullTasksByFamilyKey := buildKeyWithClusterName(cluster, TasksByFamilyKey, family)

	c := t.RedisPool.Get()
	defer c.Close()

	_, err := c.Do("HDEL", fullTasksByFamilyKey, arn)
	if err != nil {
		return fmt.Errorf("Deleting arn %q from %q failed: %s", arn, fullTasksByFamilyKey, err)
	}

	// remove key if empty
	len, err := redis.Int(c.Do("HLEN", fullTasksByFamilyKey))
	if err != nil {
		return fmt.Errorf("Getting length of %q failed: %s", fullTasksByFamilyKey, err)
	}
	if len == 0 {
		_, err := c.Do("DEL", fullTasksByFamilyKey)
		if err != nil {
			return fmt.Errorf("Deleting %q failed: %s", fullTasksByFamilyKey, err)
		}
	}
	return nil
}
