//go:build implant && (!lp || !teamserver)

package fire

import (
	"bufio"
	"log"
	"os/exec"
	"shlyuz/pkg/component"
)

func ExecuteCmd(cmd string, execChannels *component.ComponentExecutionChannel) {
	command := exec.Command(cmd)
	cmdOutReader, _ := command.StdoutPipe()
	// cmdErrReader, _ := command.StderrPipe()
	// cmdInReader, _ := command.StdinPipe()

	outScanner := bufio.NewScanner(cmdOutReader)
	go reader(outScanner, execChannels.StdOut)

	go func() {
		value := <-execChannels.StdOut
		println(value)
		execChannels.Pid <- command.Process.Pid
	}()

	_ = command.Run()

	err := command.Run()
	if err != nil {
		log.Println("encountered error while executing ", execChannels.CmdId)
		execChannels.StdErr <- err.Error()
	}
}

func reader(scanner *bufio.Scanner, out chan string) {
	for scanner.Scan() {
		out <- scanner.Text()
	}
}

func writer(scanner *bufio.Scanner, in chan string) {
	for scanner.Scan() {
		<-in // TODO: what do here?
	}
}
