package component

import (
	"shlyuz/internal/debugLp/pkg/config"
	// routine "shlyuz/pkg/crypto"
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
	XorKey             int
	TmpChannel         chan []byte
	CmdProcessingQueue []byte
	CmdDoneQueue       []byte
}

type ComponentManifest struct {
	Id       string
	Os       string
	Hostname string
	Arch     string
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

func AckCmd(frame instructions.Transaction, component Component) bool {
	return true
}
