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

func GenerateInitFrame(component component.Component) instructions.InstructionFrame {
	var initFrame instructions.Transaction
	var initArgs initFrameArgs

	initFrame.Cmd = "ii"
	initArgs.manifest = component.Manifest
	argMapping := initFrameArgs{manifest: component.Manifest, lpk: component.InitalKeypair.PubKey}
	argMap, _ := json.Marshal(argMapping)
	initFrame.Arg = argMap
	initFrame.ComponentId = component.Config.Id
	instructionFrame := instructions.CreateInstructionFrame(initFrame)
	return *instructionFrame
}

// TODO: Finish me
func DecodeInitFrame(initFrame []byte) {
	var implantManifest implantInitFrameArgs
	err := json.Unmarshal(initFrame, &implantManifest)
	if err != nil {
		log.Println("failed to decode received init frame: ", err)
	}
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
	log.Println("Awaiting response for init")
	return shlyuzComponent
}

func RetrieveInitFrame(shlyuzComponent *component.Component, shlyuzTransport transport.TransportMethod) []byte {
	data, boolSuccess, err := shlyuzTransport.Recv(shlyuzComponent)
	if boolSuccess == false {
		log.Println("failed to receive from channel: ", err)
	}
	// implantInitFrame := routine.UnwrapSealedFrame(data, shlyuzComponent.InitalKeypair.PrivKey, shlyuzComponent.XorKey)
	routine.UnwrapSealedFrame(data, shlyuzComponent.InitalKeypair.PrivKey, shlyuzComponent.XorKey, shlyuzComponent.Config.InitSignature)
	return nil
}

func rekey(frame routine.EncryptedFrame) {

}
