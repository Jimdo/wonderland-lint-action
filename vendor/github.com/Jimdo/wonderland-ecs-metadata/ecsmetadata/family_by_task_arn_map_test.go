package ecsmetadata

import (
	"testing"

	"github.com/garyburd/redigo/redis"
	"github.com/rafaeljusto/redigomock"
)

func TestFamilyByTaskMap_Get(t *testing.T) {
	conn := redigomock.NewConn()
	conn.Command("EXISTS", familyByTaskKey).Expect(redisTrue)
	conn.Command("HGET", familyByTaskKey, arn).Expect(testFamily)
	// "mock" Close command
	conn.Command("").Expect("ok")

	familyByTaskArnMap := newFamilyByTaskArnMap(conn)

	actualFamily, err := familyByTaskArnMap.Get(testCluster, arn)
	if err != nil {
		t.Fatalf("getting family by arn failed: %s", err)
	}
	if actualFamily != testFamily {
		t.Errorf("family does not match")
	}
	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("Failed to execute all redis calls: %s", err)
	}
}

func TestFamilyByTaskMap_Set(t *testing.T) {
	conn := redigomock.NewConn()
	conn.Command("HSET", familyByTaskKey, arn, testFamily).Expect("ok")
	// "mock" Close command
	conn.Command("").Expect("ok")

	familyByTaskArnMap := newFamilyByTaskArnMap(conn)

	err := familyByTaskArnMap.Set(testCluster, arn, testFamily)
	if err != nil {
		t.Fatalf("getting family by arn failed: %s", err)
	}
	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("Failed to execute all redis calls: %s", err)
	}
}

func TestFamilyByTaskMap_Del(t *testing.T) {
	conn := redigomock.NewConn()
	conn.Command("HDEL", familyByTaskKey, arn).Expect("ok")
	// "mock" Close command
	conn.Command("").Expect("ok")

	familyByTaskArnMap := newFamilyByTaskArnMap(conn)

	err := familyByTaskArnMap.Del(testCluster, arn)
	if err != nil {
		t.Fatalf("deleting arn failed: %s", err)
	}
	if err := conn.ExpectationsWereMet(); err != nil {
		t.Errorf("Failed to execute all redis calls: %s", err)
	}
}

func newFamilyByTaskArnMap(conn redis.Conn) *FamilyByTaskArnMap {
	return &FamilyByTaskArnMap{
		RedisPool: &redis.Pool{
			Dial: func() (redis.Conn, error) {
				return conn, nil
			},
		},
	}
}
