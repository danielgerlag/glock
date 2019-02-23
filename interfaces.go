package glock

type DistributedLockManager interface {
	AcquireLock(id string) bool
	ReleaseLock(id string)
	SetLeaseSecs(interval int64)
	SetHeartbeatSecs(interval int64)
	Start()
	Stop()
}
