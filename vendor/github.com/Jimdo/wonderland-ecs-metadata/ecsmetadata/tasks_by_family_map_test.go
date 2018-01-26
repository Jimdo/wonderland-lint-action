package ecsmetadata

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/garyburd/redigo/redis"
	"github.com/rafaeljusto/redigomock"
)

func TestTasksByFamilyMap_Get(t *testing.T) {
	serializedTask, err := json.Marshal(defaultTask)
	if err != nil {
		t.Fatalf("marshalling task for test failed: %s", err)
	}

	conn := redigomock.NewConn()
	conn.Command("HGET", tasksByFamilyKey, arn).Expect(serializedTask)
	// "mock" Close command
	conn.Command("").Expect("ok")

	tasksByFamilyMap := newTasksByFamilyMap(conn)

	task, err := tasksByFamilyMap.Get(testCluster, testFamily, arn)
	if err != nil {
		t.Fatalf("Getting task failed: %s", err)
	}
	if aws.StringValue(task.TaskArn) != aws.StringValue(defaultTask.TaskArn) {
		t.Errorf("Resulting Task ARN %q does not match default task ARN %q", aws.StringValue(task.TaskArn), aws.StringValue(defaultTask.TaskArn))
	}
	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("Failed to execute all redis calls: %s", err)
	}
}

func TestTasksByFamilyMap_Get_NotExistingTask(t *testing.T) {
	conn := redigomock.NewConn()
	conn.Command("HGET", tasksByFamilyKey, arn).Expect(nil)
	// "mock" Close command
	conn.Command("").Expect("ok")

	tasksByFamilyMap := newTasksByFamilyMap(conn)

	task, err := tasksByFamilyMap.Get(testCluster, testFamily, arn)
	if err != nil {
		t.Fatalf("Getting task failed: %s", err)
	}
	if task != nil {
		t.Errorf("expected task to be nil, got %q", task)
	}
	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("Failed to execute all redis calls: %s", err)
	}
}

func TestTasksByFamilyMap_GetAll(t *testing.T) {
	expectedTasks := []*ecs.Task{defaultTask}
	serializedTasks := map[string]string{}
	for _, task := range expectedTasks {
		serializedTask, err := json.Marshal(task)
		if err != nil {
			t.Fatalf("Failed to serialize task: %s", err)
		}
		serializedTasks[aws.StringValue(task.TaskArn)] = string(serializedTask)
	}

	conn := redigomock.NewConn()
	conn.Command("HGETALL", tasksByFamilyKey).ExpectMap(serializedTasks)
	// "mock" Close command
	conn.Command("").Expect("ok")

	tasksByFamilyMap := newTasksByFamilyMap(conn)

	tasks, err := tasksByFamilyMap.GetAll(testCluster, testFamily)
	if err != nil {
		t.Fatalf("Getting all tasks by family failed: %s", err)
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
	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("Failed to execute all redis calls: %s", err)
	}
}
func TestTasksByFamilyMap_Set(t *testing.T) {
	serializedTask, err := json.Marshal(defaultTask)
	if err != nil {
		t.Fatalf("marshalling task for test failed: %s", err)
	}

	conn := redigomock.NewConn()
	conn.Command("HSET", tasksByFamilyKey, arn, serializedTask).Expect("ok")
	// "mock" Close command
	conn.Command("").Expect("ok")

	tasksByFamilyMap := newTasksByFamilyMap(conn)

	err = tasksByFamilyMap.Set(testCluster, testFamily, arn, defaultTask)
	if err != nil {
		t.Fatalf("Setting task failed: %s", err)
	}
	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("Failed to execute all redis calls: %s", err)
	}
}

func TestTasksByFamilyMap_Del(t *testing.T) {
	conn := redigomock.NewConn()
	conn.Command("HDEL", tasksByFamilyKey, arn).Expect("ok")
	conn.Command("HLEN", tasksByFamilyKey).Expect([]byte("1"))
	// "mock" Close command
	conn.Command("").Expect("ok")

	tasksByFamilyMap := newTasksByFamilyMap(conn)

	err := tasksByFamilyMap.Del(testCluster, testFamily, arn)
	if err != nil {
		t.Fatalf("deleting task failed: %s", err)
	}
	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("Failed to execute all redis calls: %s", err)
	}
}

func TestTasksByFamilyMap_Del_LastKey(t *testing.T) {
	conn := redigomock.NewConn()
	conn.Command("HDEL", tasksByFamilyKey, arn).Expect("ok")
	conn.Command("HLEN", tasksByFamilyKey).Expect([]byte("0"))
	conn.Command("DEL", tasksByFamilyKey).Expect("ok")
	// "mock" Close command
	conn.Command("").Expect("ok")

	tasksByFamilyMap := newTasksByFamilyMap(conn)

	err := tasksByFamilyMap.Del(testCluster, testFamily, arn)
	if err != nil {
		t.Fatalf("deleting task failed: %s", err)
	}
	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("Failed to execute all redis calls: %s", err)
	}
}

func newTasksByFamilyMap(conn redis.Conn) *TasksByFamilyMap {
	return &TasksByFamilyMap{
		RedisPool: &redis.Pool{
			Dial: func() (redis.Conn, error) {
				return conn, nil
			},
		},
	}
}
