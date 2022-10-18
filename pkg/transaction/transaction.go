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
	Manifest component.ComponentManifest
	Ipk      asymmetric.PublicKey
}

type initAckFrameArgs struct {
	Lpk  asymmetric.PublicKey
	Txid string
}

type reqCmdArgs struct {
	Ipk  asymmetric.PublicKey
	TxId string
}

func GenerateInitFrame(component component.Component) instructions.InstructionFrame {
	var initFrame instructions.Transaction
	var initArgs initFrameArgs

	initFrame.Cmd = "ii"
	initArgs.Manifest = component.Manifest
	argMapping := initFrameArgs{Manifest: component.Manifest, Ipk: component.InitalKeypair.PubKey}
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
	transmitFrame, _ := routine.PrepareSealedFrame(frameMap, shlyuzComponent.InitalRemotePubkey, shlyuzComponent.Config.CryptoConfig.XorKey, shlyuzComponent.Config.InitSignature)
	// shlyuzComponent.CurrentKeypair = frameKeyPair
	shlyuzComponent.TmpChannel = make(chan []byte)
	go writeToChannel(shlyuzComponent.TmpChannel, transmitFrame)
	boolSuccess, err := shlyuzTransport.Send(shlyuzComponent.TmpChannel)
	if !boolSuccess {
		log.Fatalln("failed to send init: ", err)
	}
	log.Println("Sent init frame.")
	return shlyuzComponent
}

func RelayInstructionFrame(server *transport.RegisteredServer, instruction instructions.InstructionFrame) *transport.RegisteredServer {
	dataFrame, _ := json.Marshal(instruction)
	transmitFrame, _ := routine.PrepareTransmitFrame(dataFrame, server.CurPubKey, server.CurKeyPair.PrivKey, server.XorKey)
	// server.CurKeyPair = frameKeyPair
	go writeToChannel(server.CmdChannel, transmitFrame)
	boolSuccess, err := server.Transport.Send(server.CmdChannel)
	if !boolSuccess {
		log.Fatalln("failed to send instruction: ", err)
	}
	log.Println("sent instruction")
	return server
}

func RetrieveInitFrame(shlyuzComponent *component.Component, shlyuzTransport transport.TransportMethod) (transport.RegisteredServer, bool) {
	var lpInit transport.RegisteredServer
	lpInit.CmdChannel = make(chan []byte)
	data, boolSuccess, err := shlyuzTransport.Recv(lpInit.CmdChannel)
	if !boolSuccess {
		log.Println("failed to receive from channel: ", err)
		return lpInit, false
	}
	lpInit.CurKeyPair = shlyuzComponent.InitalKeypair
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

	var lpInitArgs initAckFrameArgs
	err = json.Unmarshal([]byte(lpInitInstruction.CmdArgs), &lpInitArgs)
	if err != nil {
		log.Println("[WARNING] failed to decode init args: ", err)
		return lpInit, false
	}
	lpInit.CurPubKey = lpInitArgs.Lpk
	lpInit.Transport = shlyuzTransport
	lpInit.Id = lpInitInstruction.ComponentId
	lpInit.ComponentId = shlyuzComponent.ComponentId
	lpInit.XorKey = shlyuzComponent.Config.CryptoConfig.XorKey
	lpInit.InitSignature = shlyuzComponent.Config.InitSignature
	return lpInit, true
}

func RetrieveInstruction(server *transport.RegisteredServer) (instructions.InstructionFrame, error) {
	var instruction instructions.InstructionFrame
	var err error
	data, boolSuccess, err := server.Transport.Recv(server.CmdChannel)
	if !boolSuccess {
		return instruction, err
	}
	transactionFrame := routine.UnwrapSealedFrame(data, server.CurKeyPair.PrivKey, server.CurPubKey, server.XorKey, server.InitSignature)
	instruction = decodeTransactionFrame(transactionFrame)
	return instruction, nil
}

func RequestInstruction(server *transport.RegisteredServer) (instructions.InstructionFrame, asymmetric.AsymmetricKeyPair) {
	var transactionFrame instructions.Transaction
	var rCmdArgs reqCmdArgs
	transactionFrame.Cmd = "icmdr"
	retKeyPair, err := asymmetric.GenerateKeypair()
	if err != nil {
		log.Println("failed to generate new keys")
	}
	rCmdArgs.Ipk = retKeyPair.PubKey
	rCmdArgs.TxId = idgen.GenerateTxId()
	transactionFrame.ComponentId = server.ComponentId
	argMap, _ := json.Marshal(rCmdArgs)
	transactionFrame.Arg = argMap
	transactionFrame.TxId = rCmdArgs.TxId
	instructionFrame := instructions.CreateInstructionFrame(transactionFrame)
	return *instructionFrame, retKeyPair
}

func readFromTransport(server transport.RegisteredServer, shlyuzComponent *component.Component) ([]byte, bool, error) {
	data, boolSuccess, err := server.Transport.Recv(server.CmdChannel)
	if !boolSuccess {
		log.Println("failed to receive from channel: ", err)
		return data, false, err
	}
	return data, true, nil
}

func rekey(frame routine.EncryptedFrame) {

}
