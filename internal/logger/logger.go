package logger

import "log"

type Logger interface {
	Printf(format string, args ...interface{})
}

var DefaultLogger logger

type logger struct{}

func (logger) Printf(format string, args ...interface{}) {
	log.Printf(format, args)
}
