package glock_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/danielgerlag/glock"
)

func TestScratch(t *testing.T) {
	config := &aws.Config{
		Region:   aws.String("us-east-1"),
		Endpoint: aws.String("http://127.0.0.1:8000"),
	}
	subject := glock.New(config, "glock")

	subject.Start()
	fmt.Println("started")

	result1 := subject.AcquireLock("1")
	log.Println(result1)

	subject.Stop()
}
