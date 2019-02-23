package glock

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
)

type dynamoLockManager struct {
	tableName         string
	nodeID            string
	defaultLeaseTime  int64
	jitterTolerance   int64
	heartbeatInterval int64
	localLocks        map[string]interface{}
	mux               sync.Mutex
	ticker            *time.Ticker
	dynamo            *dynamodb.DynamoDB
	awsConfig         *aws.Config
	session           *session.Session
}

func New(config *aws.Config, tableName string) DistributedLockManager {
	session := session.Must(session.NewSession(config))
	db := dynamodb.New(session)
	return &dynamoLockManager{
		tableName:         tableName,
		defaultLeaseTime:  30,
		heartbeatInterval: 10,
		jitterTolerance:   1,
		localLocks:        make(map[string]interface{}),
		awsConfig:         config,
		nodeID:            uuid.New().String(),
		session:           session,
		dynamo:            db,
	}
}

func (dlm *dynamoLockManager) AcquireLock(id string) bool {

	lock := lock{
		ID:        id,
		LockOwner: dlm.nodeID,
		Expires:   time.Now().Unix() + dlm.defaultLeaseTime,
	}
	lockAV, err := dynamodbattribute.MarshalMap(lock)
	if err != nil {
		panic("Cannot marshal lock into AttributeValue map")
	}

	params := &dynamodb.PutItemInput{
		TableName:           aws.String(dlm.tableName),
		Item:                lockAV,
		ConditionExpression: aws.String("attribute_not_exists(id) OR (expires < :expired)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":expired": {
				N: aws.String(strconv.FormatInt(time.Now().Unix()+dlm.jitterTolerance, 10)),
			},
		},
	}

	_, err2 := dlm.dynamo.PutItem(params)

	if err2 == nil {
		dlm.mux.Lock()
		defer dlm.mux.Unlock()
		dlm.localLocks[id] = nil
		return true
	}

	if awsErr, ok := err2.(awserr.Error); ok {
		switch awsErr.Code() {
		case dynamodb.ErrCodeConditionalCheckFailedException:
			return false
		default:
			log.Println(awsErr.Error())
		}
	} else {
		log.Panicln(awsErr.Error())
	}
	return false
}

func (dlm *dynamoLockManager) ReleaseLock(id string) {
	dlm.mux.Lock()
	delete(dlm.localLocks, id)
	dlm.mux.Unlock()

	dlm.dynamo.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(dlm.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
		},
		ConditionExpression: aws.String("lock_owner = :node_id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":node_id": {
				S: aws.String(dlm.nodeID),
			},
		},
	})
}

func (dlm *dynamoLockManager) Start() {
	dlm.provisionTable()
	dlm.ticker = time.NewTicker(time.Duration(dlm.heartbeatInterval) * time.Second)
	go sendHeartbeat(dlm)
}

func (dlm *dynamoLockManager) Stop() {
	dlm.ticker.Stop()
}

func (dlm *dynamoLockManager) SetHeartbeatSecs(interval int64) {
	dlm.heartbeatInterval = interval
}

func (dlm *dynamoLockManager) SetLeaseSecs(interval int64) {
	dlm.defaultLeaseTime = interval
}
