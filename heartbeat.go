package glock

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func sendHeartbeat(dlm dynamoLockManager) {
	for range dlm.ticker.C {
		renewLeases(dlm)
	}
}

func renewLeases(dlm dynamoLockManager) {
	dlm.mux.Lock()
	defer dlm.mux.Unlock()

	for id := range dlm.localLocks {
		lock := lock{
			ID:        id,
			LockOwner: dlm.nodeID,
			Expires:   time.Now().Unix() + dlm.defaultLeaseTime,
		}
		dmap, err := dynamodbattribute.MarshalMap(lock)
		if err != nil {
			panic("Cannot marshal lockitem into AttributeValue map")
		}

		params := &dynamodb.PutItemInput{
			TableName:           aws.String(dlm.tableName),
			Item:                dmap,
			ConditionExpression: aws.String("lock_owner = :node_id"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":node_id": {
					S: aws.String(dlm.nodeID),
				},
			},
		}

		_, err = dlm.dynamo.PutItem(params)

		if err != nil {
			log.Println("Error renewing lease for " + id + " - " + err.Error())
			delete(dlm.localLocks, id)
		}
	}
}
