package locking

import (
	"errors"
	"time"
)

var ErrLockAlreadyTaken = errors.New("the lock is already taken by someone else")

type LockManager interface {
	Acquire(name string, duration time.Duration) error
	Refresh(name string, duration time.Duration) error
	Release(name string) error
}
