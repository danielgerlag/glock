package glock_test

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/danielgerlag/glock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var config = &aws.Config{
	Region:   aws.String("us-east-1"),
	Endpoint: aws.String("http://127.0.0.1:8000"),
}

var tableName = "glock_tests"

func Test_AcquireLock(t *testing.T) {
	resource1 := uuid.New().String()
	node1 := glock.New(config, tableName)
	node2 := glock.New(config, tableName)
	node1.SetHeartbeatSecs(2)
	node2.SetHeartbeatSecs(2)
	node1.Start()
	node2.Start()
	defer node1.Stop()
	defer node2.Stop()

	result1 := node1.AcquireLock(resource1)
	result2 := node1.AcquireLock(resource1)
	result3 := node2.AcquireLock(resource1)

	assert.True(t, result1)
	assert.False(t, result2)
	assert.False(t, result3)
}

func Test_ReleaseLock(t *testing.T) {
	resource1 := uuid.New().String()
	node1 := glock.New(config, tableName)
	node1.Start()
	defer node1.Stop()

	if !node1.AcquireLock(resource1) {
		t.Fail()
	}

	node1.ReleaseLock(resource1)

	if !node1.AcquireLock(resource1) {
		t.Fail()
	}
}

func Test_Expiry(t *testing.T) {
	resource1 := uuid.New().String()
	node1 := glock.New(config, tableName)
	node1.SetHeartbeatSecs(100)
	node1.SetLeaseSecs(2)
	node1.Start()
	defer node1.Stop()

	if !node1.AcquireLock(resource1) {
		t.Fail()
	}

	time.Sleep(5 * time.Second)

	if !node1.AcquireLock(resource1) {
		t.Fail()
	}
}

func Test_Heartbeat(t *testing.T) {
	resource1 := uuid.New().String()
	node1 := glock.New(config, tableName)
	node1.SetHeartbeatSecs(1)
	node1.SetLeaseSecs(2)
	node1.Start()
	defer node1.Stop()

	if !node1.AcquireLock(resource1) {
		t.Fail()
	}

	time.Sleep(5 * time.Second)

	if node1.AcquireLock(resource1) {
		t.Fail()
	}
}

