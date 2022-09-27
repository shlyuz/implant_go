package logging

import (
	"log"
	"runtime"
	"time"
)

func GetLogPrefix() string {
	log.SetFlags(log.Lmsgprefix)
	pc, _, _, _ := runtime.Caller(1)
	return "[" + time.Now().UTC().Format(time.RFC3339) + "] " + "<" + runtime.FuncForPC(pc).Name() + "> "
}
