package transaction

import (
	"encoding/json"
	"log"
	"shlyuz/pkg/component"
	routine "shlyuz/pkg/crypto"
	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/instructions"
	"shlyuz/pkg/transport"
	"shlyuz/pkg/utils/idgen"
)

type initFrameArgs struct {
	manifest component.ComponentManifest
	ipk      asymmetric.PublicKey
}

type initAckFrameArgs struct {
	Lpk  asymmetric.PublicKey
	Txid string
}

type reqCmdArgs struct {
	ipk  asymmetric.PublicKey
	txId string
}

type RegisteredServer struct {
	InitArgs  initAckFrameArgs
	CurPubKey asymmetric.PublicKey
	Interface transport.TransportMethod
	Id        string
}

func GenerateInitFrame(component component.Component) instructions.InstructionFrame {
	var initFrame instructions.Transaction
	var initArgs initFrameArgs

	initFrame.Cmd = "ii"
	initArgs.manifest = component.Manifest
	argMapping := initFrameArgs{manifest: component.Manifest, ipk: component.InitalKeypair.PubKey}
	argMap, _ := json.Marshal(argMapping)
	initFrame.Arg = argMap
	initFrame.ComponentId = component.Config.Id
	instructionFrame := instructions.CreateInstructionFrame(initFrame)
	return *instructionFrame
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

func RelayInitFrame(shlyuzComponent *component.Component, initFrame instructions.InstructionFrame, shlyuzTransport transport.TransportMethod) *component.Component {
	frameMap, _ := json.Marshal(initFrame)
	transmitFrame, frameKeyPair := routine.PrepareSealedFrame(frameMap, shlyuzComponent.CurrentLpPubkey, shlyuzComponent.XorKey, shlyuzComponent.Config.InitSignature)
	shlyuzComponent.CurrentKeypair = frameKeyPair
	go writeToChannel(shlyuzComponent.CmdChannel, transmitFrame)
	boolSuccess, err := shlyuzTransport.Send(shlyuzComponent)
	if !boolSuccess {
		log.Fatalln("failed to send init: ", err)
	}
	log.Println("Sent init frame.")
	return shlyuzComponent
}

func RelayInstructionFrame(shlyuzComponent *component.Component, instruction instructions.InstructionFrame, shlyuzTransport transport.TransportMethod) *component.Component {
	dataFrame, _ := json.Marshal(instruction)
	transmitFrame, frameKeyPair := routine.PrepareTransmitFrame(dataFrame, shlyuzComponent.CurrentLpPubkey, shlyuzComponent.XorKey)
	shlyuzComponent.CurrentKeypair = frameKeyPair
	go writeToChannel(shlyuzComponent.CmdChannel, transmitFrame)
	boolSuccess, err := shlyuzTransport.Send(shlyuzComponent)
	if !boolSuccess {
		log.Fatalln("failed to send instruction: ", err)
	}
	log.Println("sent instruction")
	return shlyuzComponent
}

func RetrieveInitFrame(shlyuzComponent *component.Component, shlyuzTransport transport.TransportMethod) (RegisteredServer, bool) {
	var lpInit RegisteredServer
	data, boolSuccess, err := shlyuzTransport.Recv(shlyuzComponent)
	if !boolSuccess {
		log.Println("failed to receive from channel: ", err)
		return lpInit, false
	}
	lpInitFrame := routine.UnwrapSealedFrame(data, shlyuzComponent.InitalKeypair.PrivKey, shlyuzComponent.InitalKeypair.PubKey, shlyuzComponent.XorKey, shlyuzComponent.Config.InitSignature)
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

	var lpInitArgs initAckFrameArgs
	err = json.Unmarshal([]byte(lpInitInstruction.CmdArgs), &lpInitArgs)
	if err != nil {
		log.Println("[WARNING] failed to decode init args: ", err)
		return lpInit, false
	}
	lpInit.InitArgs = lpInitArgs
	lpInit.CurPubKey = lpInit.InitArgs.Lpk
	lpInit.Interface = shlyuzTransport
	lpInit.Id = lpInitInstruction.ComponentId
	return lpInit, true
}

func RetrieveInstruction(shlyuzComponent *component.Component, shlyuzTransport transport.TransportMethod) (instructions.InstructionFrame, error) {
	var instruction instructions.InstructionFrame
	var err error
	data, boolSuccess, err := shlyuzTransport.Recv(shlyuzComponent)
	if !boolSuccess {
		return instruction, err
	}
	transactionFrame := routine.UnwrapSealedFrame(data, shlyuzComponent.CurrentKeypair.PrivKey, shlyuzComponent.CurrentLpPubkey, shlyuzComponent.XorKey, shlyuzComponent.Config.InitSignature)
	instruction = decodeTransactionFrame(transactionFrame)
	return instruction, nil
}

func RequestInstruction(shlyuzComponent *component.Component, shlyuzTransport transport.TransportMethod) instructions.InstructionFrame {
	var transactionFrame instructions.Transaction
	var rCmdArgs reqCmdArgs
	transactionFrame.Cmd = "icmdr"
	rCmdArgs.ipk = shlyuzComponent.CurrentKeypair.PubKey
	rCmdArgs.txId = idgen.GenerateTxId()
	transactionFrame.ComponentId = shlyuzComponent.ComponentId
	argMap, _ := json.Marshal(rCmdArgs)
	transactionFrame.Arg = argMap
	transactionFrame.TxId = rCmdArgs.txId
	instructionFrame := instructions.CreateInstructionFrame(transactionFrame)
	return *instructionFrame
}

func readFromTransport(server RegisteredServer, shlyuzComponent *component.Component) ([]byte, bool, error) {
	data, boolSuccess, err := server.Interface.Recv(shlyuzComponent)
	if !boolSuccess {
		log.Println("failed to receive from channel: ", err)
		return data, false, err
	}
	return data, true, nil
}

func rekey(frame routine.EncryptedFrame) {

}
