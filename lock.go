package glock

type lock struct {
	ID        string `dynamodbav:"id"`
	LockOwner string `dynamodbav:"lock_owner"`
	Expires   int64  `dynamodbav:"expires"`
}
