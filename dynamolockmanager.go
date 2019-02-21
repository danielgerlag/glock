package glock

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type dynamoLockManager struct {
	tableName        string
	nodeID           string
	defaultLeaseTime int64
	jitterTolerance  int64
	localLocks       []string
	mux              sync.Mutex
	ticker           *time.Ticker
	dynamo           *dynamodb.DynamoDB
}

func New(tableName string) dynamoLockManager {
	return dynamoLockManager{
		tableName:        tableName,
		defaultLeaseTime: 30000,
	}
}

func (dlm *dynamoLockManager) AcquireLock(id string) bool {

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

	_, err = dlm.dynamo.PutItem(params)

	if err == nil {
		dlm.mux.Lock()
		defer dlm.mux.Unlock()
		dlm.localLocks = append(dlm.localLocks, id)
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

func (dlm *dynamoLockManager) Start() {
	dlm.provisionTable()
	dlm.ticker = time.NewTicker(500 * time.Millisecond)
	go dlm.sendHeartbeat()
}

func (dlm *dynamoLockManager) Stop() {
	dlm.ticker.Stop()
}
