package glock

type DistributedLockManager interface {
	AcquireLock(id string) bool
}
