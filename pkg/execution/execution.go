package execution

import (
	"encoding/json"
	"log"
	"shlyuz/pkg/component"
	"shlyuz/pkg/execution/fire"
	"shlyuz/pkg/instructions"
	"shlyuz/pkg/transport"
)

type cmdArgs struct {
	Cmd         int // 1=f, 2=ff, 3=fc, 3=fi
	ComponentId string
	Args        string
}

func makeExecutionChannels(instruction instructions.InstructionFrame) component.ComponentExecutionChannel {
	// TODO: We should PROBABLY encrypt or at least encode these outputs somehow. -#? EncryptCmdChannels
	execChannel := new(component.ComponentExecutionChannel)
	execChannel.CmdId = instruction.TxId
	execChannel.StdIn = make(chan string)
	execChannel.StdOut = make(chan string)
	execChannel.StdErr = make(chan string)
	execChannel.Pid = make(chan int)

	return *execChannel
}

func RouteCmd(client *transport.RegisteredComponent, instruction instructions.InstructionFrame) {
	var parsedCmd cmdArgs
	err := json.Unmarshal([]byte(instruction.CmdArgs), &parsedCmd)
	if err != nil {
		log.Println("unable to unmarshal parsed cmd")
	}

	execChannel := makeExecutionChannels(instruction)
	client.ComponentExecutionChannels = append(client.ComponentExecutionChannels, &execChannel)

	switch cmd := parsedCmd.Cmd; cmd {
	case 1:
		fire.ExecuteCmd(string(parsedCmd.Args), &execChannel)
	}
}
