package main

import (
	stdlog "log"
)

type logger int

var log logger

func (l *logger) Warn(format string, value ...any) {
	stdlog.Printf("\u001B[0;33m[WARN]\u001B[0;39m "+format, value...)
}

func (l *logger) Err(format string, value ...any) {
	stdlog.Printf("\u001B[0;31m[ERROR]\u001B[0;39m "+format, value...)
}

func (l *logger) Info(format string, value ...any) {
	stdlog.Printf("\u001B[0;32m[INFO]\u001B[0;39m "+format, value...)
}

func (l *logger) Fatal(format string, value ...any) {
	stdlog.Fatalf("\u001B[0;31m[FATAL]\u001B[0;39m "+format, value)
}
