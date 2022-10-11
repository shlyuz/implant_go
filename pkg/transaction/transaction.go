package transaction

import (
	"encoding/json"
	"shlyuz/pkg/component"
	"shlyuz/pkg/crypto"
	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/instructions"
)

type initFrameArgs struct {
	manifest component.ComponentManifest
	ipk      asymmetric.PublicKey
}

func generateInitFrame(component component.Component) instructions.InstructionFrame {
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

func rekey(frame routine.EncryptedFrame) {

}
