package logging

import (
	"log"
	"runtime"
)

func GetLogPrefix() string {
	log.SetFlags(log.Flags())
	pc, _, _, _ := runtime.Caller(1)
	return "<" + runtime.FuncForPC(pc).Name() + "> "
}
