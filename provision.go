package glock

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func (dlm *dynamoLockManager) provisionTable() {
	dt, err := dlm.dynamo.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(dlm.tableName),
	})

	log.Println(dt.String())

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeResourceNotFoundException:
				dlm.createTable()
			default:
				log.Panicln(aerr.Error())
			}
		} else {
			log.Panicln(aerr.Error())
		}
	}
}

func (dlm *dynamoLockManager) createTable() {
	params := &dynamodb.CreateTableInput{
		TableName: aws.String(dlm.tableName),
		KeySchema: []*dynamodb.KeySchemaElement{
			{AttributeName: aws.String("id"), KeyType: aws.String("HASH")},
		},
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{AttributeName: aws.String("id"), AttributeType: aws.String("S")},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
	}

	_, err := dlm.dynamo.CreateTable(params)
	if err != nil {
		log.Panicln(err.Error())
	}

	for i := 1; i <= 20; i++ {
		time.Sleep(time.Second)
		poll, _ := dlm.dynamo.DescribeTable(&dynamodb.DescribeTableInput{
			TableName: aws.String(dlm.tableName),
		})

		if *poll.Table.TableStatus == "ACTIVE" {
			return
		}
	}
}
