package transaction

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"shlyuz/internal/debugLp/pkg/instructions"
	"shlyuz/internal/debugLp/pkg/transport"
	"shlyuz/pkg/component"
	routine "shlyuz/pkg/crypto"
	"shlyuz/pkg/crypto/asymmetric"
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
	transmitFrame, _ := routine.PrepareSealedFrame(frameMap, client.CurPubKey, client.XorKey, client.InitSignature)
	// shlyuzComponent.CurrentKeypair = frameKeyPair
	// TODO: We can rotate keys here for ourselves - #? KEYROAT
	// Generate a new keypair for the LP to use
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

	client.CurPubKey = implantInitInstruction.Pk // TODO: We can rotate keys here - #? KEYROAT
	client.Id = implantInitInstruction.ComponentId

	// Prepares ack frame
	var ackTransaction instructions.Transaction
	ackTransaction.Cmd = "ipi"
	ackTransaction.ComponentId = client.SelfComponentId
	ackTransaction.TxId = implantInitInstruction.TxId
	// TODO: This is a keypair that is unique to the implant
	// argMapping := implantInitAckArgs{Lpk: client.CurKeyPair.PubKey, Txid: implantInitInstruction.TxId}
	// argMap, _ := json.Marshal(argMapping)
	ackInstruction = *instructions.CreateInstructionFrame(ackTransaction, false)
	ackInstruction.Pk = client.InitalKeyPair.PubKey
	// ackInstruction.CmdArgs = string(argMap)
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

	client.CurKeyPair, err = asymmetric.GenerateKeypair()
	if err != nil {
		log.Println("failed to generate new keypair")
	}
	// Don't need to use this for instruction requests
	// var instructionCmdArgs reqCmdArgs
	// err = json.Unmarshal([]byte(requestInstruction.CmdArgs), &instructionCmdArgs)
	// if err != nil {
	// 	log.Println("failed to unmarshal")
	// }
	client.CurPubKey = requestInstruction.Pk

	return requestInstruction, err
}

func SendOutput(server transport.RegisteredComponent, event instructions.EventHist) instructions.Transaction {
	var outputArgs instructions.CmdOutput
	var OutputTransaction instructions.Transaction
	OutputTransaction.Cmd = "fcmd"
	OutputTransaction.ComponentId = server.Id
	outputArgs.Ipk = server.CurPubKey
	outputArgs.EventHistory = event
	rawOutputArgs := new(bytes.Buffer)
	json.NewEncoder(rawOutputArgs).Encode(outputArgs)
	OutputTransaction.Arg = rawOutputArgs.Bytes()
	return OutputTransaction
}
