package registry

import (
	"errors"
	"github.com/liuxp0827/grpc-lb/app"
)

var ErrDupRegister = errors.New("duplicate register")
var ErrRegistryClosed = errors.New("has closed")
var ErrFailedRenew = errors.New("failed renew")

const MaxRenewRetry = 10

type Registry interface {
	Register(a app.App) <-chan error
	Close() error
}
