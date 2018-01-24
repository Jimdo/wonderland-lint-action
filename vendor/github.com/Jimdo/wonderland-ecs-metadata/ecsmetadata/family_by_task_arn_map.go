package ecsmetadata

import (
	"fmt"

	"github.com/garyburd/redigo/redis"
)

const (
	FamilyByTaskArnKey = "family-by-task-arn"
)

type FamilyByTaskArnMap struct {
	RedisPool *redis.Pool
}

func (f *FamilyByTaskArnMap) Get(cluster, taskArn string) (string, error) {
	key := buildKeyWithClusterName(cluster, FamilyByTaskArnKey)
	c := f.RedisPool.Get()
	defer c.Close()

	if _, err := redis.Bool(c.Do("EXISTS", key)); err != nil {
		return "", fmt.Errorf("error testing existence of redis key %q: %s", key, err)
	}

	return redis.String(c.Do("HGET", key, taskArn))
}

func (f *FamilyByTaskArnMap) Set(cluster, taskArn, family string) error {
	key := buildKeyWithClusterName(cluster, FamilyByTaskArnKey)

	c := f.RedisPool.Get()
	defer c.Close()

	if _, err := c.Do("HSET", key, taskArn, family); err != nil {
		return fmt.Errorf("could not set family %q for task %q: %s", family, taskArn, err)
	}
	return nil
}

func (f *FamilyByTaskArnMap) Del(cluster, taskArn string) error {
	key := buildKeyWithClusterName(cluster, FamilyByTaskArnKey)

	c := f.RedisPool.Get()
	defer c.Close()

	if _, err := c.Do("HDEL", key, taskArn); err != nil {
		return fmt.Errorf("could not delete arn %q: %s", taskArn, err)
	}
	return nil
}
