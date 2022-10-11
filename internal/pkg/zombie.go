package main

import (
	"flag"
	"shlyuz/pkg/execution/ipc"
)

func main() {
	namedPipe := flag.Args()[0]
	ipc.Read(namedPipe)
	// TODO: Do we wanna do something with this data here? Write it to another pipe to be collected by implant maybe?
}
