package glock

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
)

type dynamoLockManager struct {
	tableName        string
	nodeID           string
	defaultLeaseTime int64
	jitterTolerance  int64
	localLocks       map[string]interface{}
	mux              sync.Mutex
	ticker           *time.Ticker
	dynamo           dynamodb.DynamoDB
	awsConfig        *aws.Config
	session          *session.Session
}

func New(config *aws.Config, tableName string) DistributedLockManager {
	return dynamoLockManager{
		tableName:        tableName,
		defaultLeaseTime: 30000,
		jitterTolerance:  1000,
		localLocks:       make(map[string]interface{}),
		awsConfig:        config,
		nodeID:           uuid.New().String(),
	}
}

func (dlm dynamoLockManager) AcquireLock(id string) bool {

	lock := lock{
		ID:        id,
		LockOwner: dlm.nodeID,
		Expires:   time.Now().Unix() + dlm.defaultLeaseTime,
	}
	dmap, err := dynamodbattribute.MarshalMap(lock)
	if err != nil {
		panic("Cannot marshal lockitem into AttributeValue map")
	}
	expired := time.Now().Unix() + dlm.jitterTolerance
	params := &dynamodb.PutItemInput{
		TableName:           aws.String(dlm.tableName),
		Item:                dmap,
		ConditionExpression: aws.String("attribute_not_exists(id) OR (expires < :expired)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":expired": {
				N: aws.String(strconv.FormatInt(expired, 10)),
			},
		},
	}

	resp, err := dlm.dynamo.PutItem(params)

	if err == nil {
		log.Println(resp.GoString())
		dlm.mux.Lock()
		defer dlm.mux.Unlock()
		dlm.localLocks[id] = nil
		return true
	}

	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case dynamodb.ErrCodeConditionalCheckFailedException:
			return false
		default:
			log.Println(aerr.Error())
		}
	} else {
		log.Panicln(aerr.Error())
	}
	return false
}

func (dlm dynamoLockManager) ReleaseLock(id string) {
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

func (dlm dynamoLockManager) Start() {
	//sess := session.Must(session.NewSession(dlm.awsConfig))
	dlm.session = session.Must(session.NewSession(dlm.awsConfig))
	dlm.dynamo = &dynamodb.New(dlm.session)
	dlm.provisionTable()
	//dlm.ticker = time.NewTicker(500 * time.Millisecond)
	//go sendHeartbeat(dlm)
}

func (dlm dynamoLockManager) Stop() {
	dlm.ticker.Stop()
}
