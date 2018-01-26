package ecsmetadata

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/garyburd/redigo/redis"
	"github.com/rafaeljusto/redigomock"
	"github.com/stretchr/testify/assert"
)

var (
	testCluster = "test-cluster"
	testFamily  = "test-family"
	arn         = "arn:...:task/test-task:1"

	taskDefinitionArn = fmt.Sprintf("arn:...:task-definition/%s:1", testFamily)
	tasksByFamilyKey  = fmt.Sprintf("%s.%s.%s", testCluster, TasksByFamilyKey, testFamily)
	familyByTaskKey   = fmt.Sprintf("%s.%s", testCluster, FamilyByTaskArnKey)

	redisTrue = []byte("1")

	defaultTask = &ecs.Task{
		TaskArn:           aws.String(arn),
		TaskDefinitionArn: aws.String(taskDefinitionArn),
		DesiredStatus:     aws.String(ecs.DesiredStatusRunning),
	}
)

func TestStatefulRedis_UpdateTasks(t *testing.T) {
	tasks := []*ecs.Task{defaultTask}

	redisConn, err := setupRedisForTaskUpdate(tasks[0])
	if err != nil {
		t.Fatalf("Redis setup failed: %s", err)
	}
	redisMetadata := newRedisMetadataWithRedisConn(redisConn)

	err = redisMetadata.UpdateTasks(testCluster, tasks)
	if err != nil {
		t.Fatalf("Failed to Update Tasks: %s", err)
	}
}

func setupRedisForTaskUpdate(task *ecs.Task) (redis.Conn, error) {
	serializedTask, err := json.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("Marshalling task failed: %s", err)
	}

	redisConn := redigomock.NewConn()
	redisConn.
		Command("HGET", tasksByFamilyKey, arn).
		Expect(nil)

	redisConn.
		Command("HSET", tasksByFamilyKey, arn, serializedTask).
		Expect("ok")

	redisConn.
		Command("HSET", familyByTaskKey, arn, testFamily).
		Expect("ok")

	return redisConn, nil
}

func newRedisMetadataWithRedisConn(conn redis.Conn) *StatefulRedis {
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return conn, nil
		},
	}
	return NewStatefulRedis(pool)
}

func TestStatefulRedis_GetTask(t *testing.T) {
	redisConn, err := setupRedisForGetTask(defaultTask)
	if err != nil {
		t.Fatalf("Setting up redis failed: %s", err)
	}

	redisMetadata := newRedisMetadataWithRedisConn(redisConn)

	actualTask, err := redisMetadata.GetTask(testCluster, arn)
	if err != nil {
		t.Errorf("Getting the task failed: %s", err)
	}
	if actualTask == nil {
		t.Error("got nil instead of task")
	}
	if aws.StringValue(actualTask.TaskArn) != aws.StringValue(defaultTask.TaskArn) {
		t.Errorf("Expected %q to match %q", aws.StringValue(actualTask.TaskArn), aws.StringValue(defaultTask.TaskArn))
	}
}

func setupRedisForGetTask(task *ecs.Task) (redis.Conn, error) {
	serializedTask, err := json.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("Failed to serialize task: %s", err)
	}

	conn := redigomock.NewConn()
	conn.Command("EXISTS", familyByTaskKey).Expect(redisTrue)
	conn.Command("EXISTS", tasksByFamilyKey).Expect(redisTrue)
	conn.Command("HGET", familyByTaskKey, arn).Expect(testFamily)
	conn.Command("HGET", tasksByFamilyKey, arn).Expect(serializedTask)
	return conn, nil
}

func TestStatefulRedis_GetTasks(t *testing.T) {
	expectedTasks := []*ecs.Task{defaultTask}

	conn, err := setupRedisForGetTasks(expectedTasks)
	if err != nil {
		t.Fatalf("Failed to set up redis: %s", err)
	}
	redisMetadata := newRedisMetadataWithRedisConn(conn)

	tasks, err := redisMetadata.GetTasks(testCluster, testFamily, ecs.DesiredStatusRunning)
	if len(tasks) != len(expectedTasks) {
		t.Errorf("found %d tasks, but expected %d tasks", len(tasks), len(expectedTasks))
	}
	if err != nil {
		t.Fatalf("Failed to get tasks from redis: %s", err)
	}
	for _, task := range tasks {
		spotted := false
		for _, expected := range expectedTasks {
			if aws.StringValue(expected.TaskArn) == aws.StringValue(task.TaskArn) {
				spotted = true
				break
			}
		}
		if !spotted {
			t.Errorf("Could not find task with arn %s", aws.StringValue(task.TaskArn))
		}
	}
}

func setupRedisForGetTasks(tasks []*ecs.Task) (*redigomock.Conn, error) {
	serializedTasks := map[string]string{}
	for _, task := range tasks {
		serializedTask, err := json.Marshal(task)
		if err != nil {
			return nil, fmt.Errorf("Failed to serialize task: %s", err)
		}
		serializedTasks[aws.StringValue(task.TaskArn)] = string(serializedTask)
	}

	if len(serializedTasks) != 1 {
		return nil, fmt.Errorf("Serializing the tasks didn't work out")
	}

	conn := redigomock.NewConn()
	conn.Command("EXISTS", tasksByFamilyKey).Expect(redisTrue)
	conn.Command("HGETALL", tasksByFamilyKey).ExpectMap(serializedTasks)
	return conn, nil
}

func TestStatefulRedis_RemoveTask(t *testing.T) {
	conn, err := setupRedisForRemoveTask()
	if err != nil {
		t.Fatalf("Failed to set up redis: %s", err)
	}
	redisMetadata := newRedisMetadataWithRedisConn(conn)

	err = redisMetadata.RemoveTask(testCluster, defaultTask)
	if err != nil {
		t.Errorf("Removing task failed: %s", err)
	}
}

func setupRedisForRemoveTask() (redis.Conn, error) {
	conn := redigomock.NewConn()
	conn.Command("HDEL", tasksByFamilyKey, arn)
	conn.Command("HLEN", tasksByFamilyKey).Expect([]byte("1"))
	conn.Command("HDEL", familyByTaskKey, arn)
	return conn, nil
}

func TestStatefulRedis_RemoveTaskAndKey(t *testing.T) {
	conn, err := setupRedisForRemoveTaskAndKey()
	if err != nil {
		t.Fatalf("Failed to set up redis: %s", err)
	}
	redisMetadata := newRedisMetadataWithRedisConn(conn)

	err = redisMetadata.RemoveTask(testCluster, defaultTask)
	if err != nil {
		t.Errorf("Removing task failed: %s", err)
	}
}

func setupRedisForRemoveTaskAndKey() (redis.Conn, error) {
	conn := redigomock.NewConn()
	conn.Command("HDEL", tasksByFamilyKey, arn)
	conn.Command("HLEN", tasksByFamilyKey).Expect([]byte("0"))
	conn.Command("DEL", tasksByFamilyKey)
	conn.Command("HDEL", familyByTaskKey, arn)
	return conn, nil
}

func TestStatefulRedis_GetContainerInstance_withClusterSupport(t *testing.T) {
	wantInstance := &ecs.ContainerInstance{
		ContainerInstanceArn: aws.String("arn:aws:ecs:eu-west-1:...:container-instance/abc123"),
	}
	serializedInstance, err := json.Marshal(wantInstance)
	if err != nil {
		t.Fatalf("Failed to serialize container instance: %s", err)
	}
	redis := redigomock.NewConn()
	redis.Command("EXISTS", fmt.Sprintf("%s.%s", "cluster-a", ContainerInstancesKey)).Expect(redisTrue)
	redis.Command("HGET", fmt.Sprintf("%s.%s", "cluster-a", ContainerInstancesKey), aws.StringValue(wantInstance.ContainerInstanceArn)).Expect(serializedInstance)

	metadata := newRedisMetadataWithRedisConn(redis)

	gotInstance, err := metadata.GetContainerInstance("cluster-a", aws.StringValue(wantInstance.ContainerInstanceArn))
	if err != nil {
		t.Fatalf("Fetching Instance for 'cluster-a' and container instance ARN 'arn:aws:ecs:eu-west-1:...:container-instance/abc123' failed: %s", err)
	}

	if aws.StringValue(gotInstance.ContainerInstanceArn) != aws.StringValue(wantInstance.ContainerInstanceArn) {
		t.Errorf(`metadata.GetContainerInstance("cluster-a", %q) = %q; want %q`,
			aws.StringValue(wantInstance.ContainerInstanceArn),
			aws.StringValue(gotInstance.ContainerInstanceArn),
			aws.StringValue(wantInstance.ContainerInstanceArn))
	}
}

func TestStatefulRedis_GetContainerInstances(t *testing.T) {
	instanceARN := "arn:aws:ecs:eu-west-1:...:container-instance/abc123"
	cluster := "test-cluster"
	wantInstance := &ecs.ContainerInstance{
		ContainerInstanceArn: aws.String(instanceARN),
		Status:               aws.String(ecs.ContainerInstanceStatusActive),
	}
	wantSerializedInstance, err := json.Marshal(wantInstance)
	if err != nil {
		t.Fatalf("Serializing the instance failed: %s", err)
	}
	wantInstances := map[string]string{
		instanceARN: string(wantSerializedInstance),
	}
	redis := redigomock.NewConn()
	key := fmt.Sprintf("%s.%s", cluster, ContainerInstancesKey)
	redis.Command("EXISTS", key).Expect(redisTrue)
	redis.Command("HGETALL", key).ExpectMap(wantInstances)

	metadata := newRedisMetadataWithRedisConn(redis)

	gotInstances, err := metadata.GetContainerInstances(cluster)
	assert.NoError(t, err)
	assert.NotNil(t, gotInstances)
	assert.Equal(t, len(gotInstances), 1)
	assert.Equal(t, aws.StringValue(gotInstances[0].ContainerInstanceArn), instanceARN)
}

func TestStatefulRedis_UpdateContainerInstance(t *testing.T) {
	instanceARN := "arn:aws:ecs:eu-west-1:...:container-instance/abc123"
	cluster := "test-cluster"
	instance := &ecs.ContainerInstance{
		ContainerInstanceArn: aws.String(instanceARN),
	}
	serializedInstance, err := json.Marshal(instance)
	if err != nil {
		t.Fatalf("Failed to serialize instance: %s", err)
	}
	key := fmt.Sprintf("%s.%s", cluster, ContainerInstancesKey)
	redis := redigomock.NewConn()
	redis.Command("HGET", key, instanceARN)
	redis.Command("HSET", key, instanceARN, serializedInstance)

	metadata := newRedisMetadataWithRedisConn(redis)
	if err := metadata.UpdateContainerInstance(cluster, instance); err != nil {
		t.Fatalf("Updating container instance failed: %s", err)
	}
}

func TestStatefulRedis_RemoveContainerInstance(t *testing.T) {
	instanceARN := "arn:aws:ecs:eu-west-1:...:container-instance/abc123"
	instance := &ecs.ContainerInstance{
		ContainerInstanceArn: aws.String(instanceARN),
	}
	cluster := "test-cluster"
	key := fmt.Sprintf("%s.%s", cluster, ContainerInstancesKey)
	redis := redigomock.NewConn()
	redis.Command("HDEL", key, instanceARN)
	metadata := newRedisMetadataWithRedisConn(redis)
	assert.NoError(t, metadata.RemoveContainerInstance(cluster, instance))
}
