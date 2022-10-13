package component

import (
	"bytes"
	"encoding/json"

	"shlyuz/internal/debugLp/pkg/config"
	// routine "shlyuz/pkg/crypto"
	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/instructions"
)

type Component struct {
	// logger             log.Logger
	ConfigFile         string
	ConfigKey          []byte
	Config             config.LpConfig
	ComponentId        string
	Manifest           ComponentManifest
	InitalKeypair      asymmetric.AsymmetricKeyPair
	CurrentKeypair     asymmetric.AsymmetricKeyPair
	CurrentImpPubkey   *[32]byte
	XorKey             int
	CmdChannel         chan []byte
	CmdProcessingQueue []byte
	CmdDoneQueue       []byte
}

type ComponentManifest struct {
	Lp_id       string
	Lp_os       string
	Lp_hostname string
}

type ImplantManifest struct {
	Implant_id       string
	Implant_os       string
	Implant_hostname string
}

// func Rekey(frame instructions.Transaction, component Component) (Component, []byte) {
// 	component.CurrentLpPubkey = component.Config.CryptoConfig.LpPk
// 	rawRekeyFrame := instructions.CreateInstructionFrame(frame)
// 	rawFrameBytes := new(bytes.Buffer)
// 	json.NewEncoder(rawFrameBytes).Encode(rawRekeyFrame)
// 	rekeyFrame, newComponentKeypair := routine.PrepareTransmitFrame(rawFrameBytes.Bytes(), component.CurrentLpPubkey, component.XorKey)
// 	component.CurrentKeypair = newComponentKeypair
// 	return component, rekeyFrame
// }

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
