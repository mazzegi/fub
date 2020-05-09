package fub

import (
	"fmt"
	"time"
)

const timeFormat = "2006-01-02T15:04:05.000"

func Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s [INFO ] %s\n", time.Now().Format(timeFormat), msg)
}

func Warnf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s [WARN ] %s\n", time.Now().Format(timeFormat), msg)
}

func Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s [ERROR] %s\n", time.Now().Format(timeFormat), msg)
}
