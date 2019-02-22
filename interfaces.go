package glock

type DistributedLockManager interface {
	AcquireLock(id string) bool
	ReleaseLock(id string)
	Start()
	Stop()
}
