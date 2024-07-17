package juju

import (
	"time"

	jujuclock "github.com/juju/clock"
	jujumutex "github.com/juju/mutex"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

type NamedMutexAcquirer struct{}

type namedMutex struct {
	releaser jujumutex.Releaser
}

func NewNamedMutexAcquirer() *NamedMutexAcquirer {
	return &NamedMutexAcquirer{}
}

func (n *NamedMutexAcquirer) AcquireNamedMutex(name string, timeout time.Duration) (domain.NamedMutex, error) {
	jujuReleaser, err := jujumutex.Acquire(jujumutex.Spec{
		Name:    name,
		Clock:   jujuclock.WallClock,
		Delay:   time.Second,
		Timeout: timeout,
	})
	if err != nil {
		return nil, err
	}
	return &namedMutex{
		releaser: jujuReleaser,
	}, nil
}

func (n *namedMutex) Release() {
	n.releaser.Release()
}
