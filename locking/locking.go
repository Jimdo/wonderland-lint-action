package locking

import (
	"errors"
	"time"
)

var ErrLeadershipAlreadyTaken = errors.New("Leadership is already taken")

type LockManager interface {
	Acquire(name string, duration time.Duration) error
	Refresh(name string, duration time.Duration) error
	Release(name string) error
}
