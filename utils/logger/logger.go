package logger

import (
	"fmt"
	"log"
)

var _log = NewLogger()

type logger struct {
	_logger *log.Logger
}

func (l *logger) Fatal(s interface{}) {
	l._logger.Fatalf("\033[35m[FATAL]\033[0m %s", s)
}

func (l *logger) Fatalf(format string, v ...interface{}) {
	l._logger.Fatalf("\033[35m[FATAL]\033[0m %s", fmt.Sprintf(format, v...))
}

func (l *logger) Error(s interface{}) {
	l._logger.Printf("\033[31m[ERROR]\033[0m %s", s)
}

func (l *logger) Errorf(format string, v ...interface{}) {
	l._logger.Printf("\033[31m[ERROR]\033[0m %s", fmt.Sprintf(format, v...))
}

func (l *logger) Warn(s interface{}) {
	l._logger.Printf("\033[33m[WARN]\033[0m %s", s)
}

func (l *logger) Warnf(format string, v ...interface{}) {
	l._logger.Printf("\033[33m[WARN]\033[0m %s", fmt.Sprintf(format, v...))
}

func (l *logger) Info(s interface{}) {
	l._logger.Printf("\033[32m[INFO]\033[0m %s", s)
}

func (l *logger) Infof(format string, v ...interface{}) {
	l._logger.Printf("\033[32m[INFO]\033[0m %s", fmt.Sprintf(format, v...))
}

func NewLogger() (res *logger) {
	res = &logger{
		_logger: log.Default(),
	}
	res._logger.SetPrefix("[protobuf-thrift] ")
	return
}

func Fatal(s interface{}) {
	_log.Fatal(s)
}

func Fatalf(format string, v ...interface{}) {
	_log.Fatalf(format, v...)
}

func Error(s interface{}) {
	_log.Error(s)
}

func Errorf(format string, v ...interface{}) {
	_log.Errorf(format, v...)
}

func Warn(s interface{}) {
	_log.Warn(s)
}

func Warnf(format string, v ...interface{}) {
	_log.Warnf(format, v...)
}

func Info(s interface{}) {
	_log.Info(s)
}

func Infof(format string, v ...interface{}) {
	_log.Infof(format, v...)
}
