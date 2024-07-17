package domain

import "time"

type NamedMutex interface {
	Release()
}

type NamedMutexAcquirer interface {
	AcquireNamedMutex(name string, timeout time.Duration) (NamedMutex, error)
}
