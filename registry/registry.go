package registry

import (
	"errors"
	"github.com/liuxp0827/grpc-lb/instance"
)

var ErrDupRegister = errors.New("duplicate register")
var ErrRegistryClosed = errors.New("has closed")
var ErrFailedRenew = errors.New("failed renew")

const MaxRenewRetry = 10

type Registry interface {
	Register(inst instance.Instance) <-chan error
	Close() error
}
